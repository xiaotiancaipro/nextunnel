package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/xiaotiancaipro/nextunnel/internal/utils"
)

// ServerParams Server 初始化参数
type ServerParams struct {
	BindPort int
	Token    string
	Logger   *logrus.Logger
}

// Server 内网穿透服务端核心
type Server struct {
	bindPort int
	token    string
	logger   *logrus.Logger

	listener net.Listener

	mu      sync.RWMutex
	clients map[string]*controlSession // runID → 控制会话
	proxies map[string]*proxyEntry     // proxyName → 代理条目

	// 等待工作连接: workID → channel(工作连接)
	pendingWork   map[string]chan net.Conn
	pendingWorkMu sync.Mutex

	stopCh chan struct{}
}

// controlSession 代表一个 client 的控制连接会话
type controlSession struct {
	runID  string
	conn   net.Conn
	mu     sync.Mutex
	stopCh chan struct{}
}

// proxyEntry 一个已注册的 TCP 代理
type proxyEntry struct {
	name       string
	remotePort int
	runID      string       // 归属的 client runID
	listener   net.Listener // 服务端在 remotePort 上的监听器
}

func NewServer(p *ServerParams) (*Server, error) {
	if p.BindPort <= 0 || p.BindPort > 65535 {
		return nil, fmt.Errorf("无效的绑定端口: %d", p.BindPort)
	}
	return &Server{
		bindPort:    p.BindPort,
		token:       p.Token,
		logger:      p.Logger,
		clients:     make(map[string]*controlSession),
		proxies:     make(map[string]*proxyEntry),
		pendingWork: make(map[string]chan net.Conn),
		stopCh:      make(chan struct{}),
	}, nil
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", s.bindPort))
	if err != nil {
		return fmt.Errorf("监听端口 %d 失败: %w", s.bindPort, err)
	}
	s.listener = ln

	go s.acceptLoop()
	return nil
}

// Stop 停止服务端，关闭所有连接
func (s *Server) Stop() {
	close(s.stopCh)
	_ = s.listener.Close()

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, proxy := range s.proxies {
		if proxy.listener != nil {
			_ = proxy.listener.Close()
		}
	}
	for _, sess := range s.clients {
		_ = sess.conn.Close()
	}
}

// acceptLoop 接受所有入站 TCP 连接（控制连接 + 工作连接）
func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopCh:
				return
			default:
				s.logger.Errorf("Accept 失败: %v", err)
				continue
			}
		}
		go s.handleIncoming(conn)
	}
}

// handleIncoming 读取第一条消息，判断是控制连接还是工作连接
func (s *Server) handleIncoming(conn net.Conn) {
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	msgType, payload, err := utils.ReadMsg(conn)
	if err != nil {
		s.logger.Errorf("读取首条消息失败 [%s]: %v", conn.RemoteAddr(), err)
		_ = conn.Close()
		return
	}
	_ = conn.SetDeadline(time.Time{})

	switch msgType {
	case utils.MsgLogin:
		s.handleControlConn(conn, payload)
	case utils.MsgStartWorkConn:
		s.handleWorkConn(conn, payload)
	default:
		s.logger.Warnf("未知首条消息类型 0x%02x [%s]", msgType, conn.RemoteAddr())
		_ = conn.Close()
	}
}

// handleControlConn 处理 client 控制连接：认证、注册代理、心跳
func (s *Server) handleControlConn(conn net.Conn, payload []byte) {
	var loginMsg utils.LoginMsg
	if err := utils.Decode(payload, &loginMsg); err != nil {
		s.logger.Errorf("解析 LoginMsg 失败: %v", err)
		_ = conn.Close()
		return
	}

	if loginMsg.Token != s.token {
		s.logger.Warnf("认证失败 [%s]: token 不匹配", conn.RemoteAddr())
		_ = utils.WriteMsg(conn, utils.MsgLoginResp, utils.LoginRespMsg{Error: "认证失败"})
		_ = conn.Close()
		return
	}

	runID := genID()
	sess := &controlSession{
		runID:  runID,
		conn:   conn,
		stopCh: make(chan struct{}),
	}

	s.mu.Lock()
	s.clients[runID] = sess
	s.mu.Unlock()

	s.logger.Infof("client 已连接 [%s], runID=%s", conn.RemoteAddr(), runID)

	if err := utils.WriteMsg(conn, utils.MsgLoginResp, utils.LoginRespMsg{RunID: runID}); err != nil {
		s.logger.Errorf("发送 LoginResp 失败: %v", err)
		s.removeClient(runID)
		return
	}

	s.controlLoop(sess)
}

// controlLoop 持续读取来自 client 的控制消息
func (s *Server) controlLoop(sess *controlSession) {
	defer s.removeClient(sess.runID)

	for {
		msgType, payload, err := utils.ReadMsg(sess.conn)
		if err != nil {
			s.logger.Infof("client 控制连接断开 runID=%s: %v", sess.runID, err)
			return
		}

		switch msgType {
		case utils.MsgNewProxy:
			s.handleNewProxy(sess, payload)
		case utils.MsgPing:
			_ = utils.WriteMsg(sess.conn, utils.MsgPong, utils.PongMsg{})
		default:
			s.logger.Warnf("控制连接收到未知消息 0x%02x runID=%s", msgType, sess.runID)
		}
	}
}

// handleNewProxy 处理 client 注册新 TCP 代理请求
func (s *Server) handleNewProxy(sess *controlSession, payload []byte) {
	var msg utils.NewProxyMsg
	if err := utils.Decode(payload, &msg); err != nil {
		s.sendProxyResp(sess, "", "解析 NewProxyMsg 失败")
		return
	}

	if msg.Type != "tcp" {
		s.sendProxyResp(sess, msg.Name, fmt.Sprintf("不支持的代理类型: %s", msg.Type))
		return
	}

	if msg.RemotePort <= 0 || msg.RemotePort > 65535 {
		s.sendProxyResp(sess, msg.Name, fmt.Sprintf("无效的远程端口: %d", msg.RemotePort))
		return
	}

	s.mu.Lock()
	if _, exists := s.proxies[msg.Name]; exists {
		s.mu.Unlock()
		s.sendProxyResp(sess, msg.Name, "代理名称已存在")
		return
	}
	s.mu.Unlock()

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", msg.RemotePort))
	if err != nil {
		s.logger.Errorf("监听远程端口 %d 失败: %v", msg.RemotePort, err)
		s.sendProxyResp(sess, msg.Name, fmt.Sprintf("监听端口 %d 失败: %v", msg.RemotePort, err))
		return
	}

	entry := &proxyEntry{
		name:       msg.Name,
		remotePort: msg.RemotePort,
		runID:      sess.runID,
		listener:   ln,
	}

	s.mu.Lock()
	s.proxies[msg.Name] = entry
	s.mu.Unlock()

	s.logger.Infof("代理注册成功: name=%s, remotePort=%d, runID=%s", msg.Name, msg.RemotePort, sess.runID)
	s.sendProxyResp(sess, msg.Name, "")

	// 启动协程监听用户连接
	go s.proxyAcceptLoop(entry, sess)
}

func (s *Server) sendProxyResp(sess *controlSession, name, errMsg string) {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	_ = utils.WriteMsg(sess.conn, utils.MsgNewProxyResp, utils.NewProxyRespMsg{
		Name:  name,
		Error: errMsg,
	})
}

// proxyAcceptLoop 在 remotePort 上接受用户连接，并通知 client 建立工作连接
func (s *Server) proxyAcceptLoop(entry *proxyEntry, sess *controlSession) {
	defer func() {
		_ = entry.listener.Close()
		s.mu.Lock()
		delete(s.proxies, entry.name)
		s.mu.Unlock()
		s.logger.Infof("代理已停止: name=%s", entry.name)
	}()

	for {
		userConn, err := entry.listener.Accept()
		if err != nil {
			select {
			case <-s.stopCh:
				return
			case <-sess.stopCh:
				return
			default:
				s.logger.Errorf("代理 [%s] Accept 失败: %v", entry.name, err)
				return
			}
		}
		s.logger.Infof("用户连接到达: proxy=%s, src=%s", entry.name, userConn.RemoteAddr())
		go s.bridgeUserConn(userConn, entry, sess)
	}
}

// bridgeUserConn 为一个用户连接请求 client 的工作连接，完成数据桥接
func (s *Server) bridgeUserConn(userConn net.Conn, entry *proxyEntry, sess *controlSession) {
	defer userConn.Close()

	workID := genID()
	workCh := make(chan net.Conn, 1)

	s.pendingWorkMu.Lock()
	s.pendingWork[workID] = workCh
	s.pendingWorkMu.Unlock()

	defer func() {
		s.pendingWorkMu.Lock()
		delete(s.pendingWork, workID)
		s.pendingWorkMu.Unlock()
	}()

	// 通知 client 建立工作连接
	sess.mu.Lock()
	err := utils.WriteMsg(sess.conn, utils.MsgNewWorkConn, utils.NewWorkConnMsg{
		WorkID:    workID,
		ProxyName: entry.name,
	})
	sess.mu.Unlock()
	if err != nil {
		s.logger.Errorf("发送 NewWorkConn 失败: %v", err)
		return
	}

	// 等待 client 的工作连接，超时 10 秒
	select {
	case workConn := <-workCh:
		s.logger.Debugf("工作连接就绪: workID=%s, proxy=%s", workID, entry.name)
		pipe(userConn, workConn)
	case <-time.After(10 * time.Second):
		s.logger.Warnf("等待工作连接超时: workID=%s, proxy=%s", workID, entry.name)
	}
}

// handleWorkConn 处理 client 发起的工作连接
func (s *Server) handleWorkConn(conn net.Conn, payload []byte) {
	var msg utils.StartWorkConnMsg
	if err := utils.Decode(payload, &msg); err != nil {
		s.logger.Errorf("解析 StartWorkConnMsg 失败: %v", err)
		_ = conn.Close()
		return
	}

	s.pendingWorkMu.Lock()
	ch, ok := s.pendingWork[msg.WorkID]
	s.pendingWorkMu.Unlock()

	if !ok {
		s.logger.Warnf("收到未知工作连接 workID=%s", msg.WorkID)
		_ = conn.Close()
		return
	}

	ch <- conn
}

// removeClient 清理 client 控制会话及其所有代理
func (s *Server) removeClient(runID string) {
	s.mu.Lock()
	sess, ok := s.clients[runID]
	if ok {
		close(sess.stopCh)
		_ = sess.conn.Close()
		delete(s.clients, runID)
	}

	for name, proxy := range s.proxies {
		if proxy.runID == runID {
			_ = proxy.listener.Close()
			delete(s.proxies, name)
			s.logger.Infof("已移除代理: name=%s (client 断连)", name)
		}
	}
	s.mu.Unlock()

	s.logger.Infof("client 会话已清理: runID=%s", runID)
}

// genID 生成随机唯一标识
func genID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
