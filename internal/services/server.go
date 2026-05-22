package services

import (
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/xiaotiancaipro/nextunnel-server/internal/clients"
	"github.com/xiaotiancaipro/nextunnel-server/internal/configs"
	"github.com/xiaotiancaipro/nextunnel-server/internal/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const unknownIp = "UNKNOWN_IP"

type Server struct {
	Config      *configs.Server
	Logger      *zap.Logger
	DB          *gorm.DB
	GeoIP       *clients.GeoIP
	pendingMu   sync.Mutex
	pendingWork map[string]*pendingWorkItem
}

type pendingWorkItem struct {
	userConn net.Conn
	certFP   [sha256.Size]byte
}

func (s *Server) Listen() (net.Listener, error) {
	listener, err := net.Listen("tcp", s.AddrStr())
	if err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to listen on %s: %v", s.AddrStr(), err))
		return nil, fmt.Errorf("failed to listen")
	}
	return listener, nil
}

func (s *Server) AddrStr() string {
	return fmt.Sprintf(":%d", s.Config.Port)
}

func (s *Server) EstablishConn(connRaw net.Conn, tlsConfig *tls.Config) (net.Conn, error) {
	conn := tls.Server(connRaw, tlsConfig)
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	if err := conn.Handshake(); err != nil {
		_ = conn.SetDeadline(time.Time{})
		s.Logger.Error(fmt.Sprintf("Failed to handshake with TLS connection: %v", err))
		return nil, fmt.Errorf("tls handshake failed")
	}
	_ = conn.SetDeadline(time.Time{})
	return conn, nil
}

func (s *Server) Login(conn net.Conn, payload []byte) (*string, *string, error) {

	var loginMsg utils.LoginMsg
	if err := utils.Decode(payload, &loginMsg); err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to parse LoginMsg: %v", err))
		return nil, nil, fmt.Errorf("failed to parse LoginMsg")
	}

	if loginMsg.Id == "" {
		_ = utils.WriteMsg(conn, utils.MsgLoginResp, utils.LoginRespMsg{Error: "client_id cannot be empty"})
		return nil, nil, fmt.Errorf("client_id is empty")
	}

	runID := uuid.New().String()
	if err := utils.WriteMsg(conn, utils.MsgLoginResp, utils.LoginRespMsg{RunID: runID}); err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to send LoginResp: %v", err))
		return nil, nil, fmt.Errorf("failed to send LoginResp")
	}

	return &loginMsg.Id, &runID, nil

}

func (s *Server) ProxiesApply(conn net.Conn, payload []byte, clientIdP *string, serverStopCh, clientStopCh chan struct{}) error {

	var msg utils.ProxiesApplyMsg
	if err := utils.Decode(payload, &msg); err != nil {
		e := fmt.Sprintf("failed to parse ApplyConfigMsg: %v", err)
		_ = utils.WriteMsg(conn, utils.MsgProxiesApplyResp, utils.ProxiesApplyRespMsg{Error: e})
		s.Logger.Error(e)
		return fmt.Errorf("failed to parse ApplyConfigMsg")
	}

	desired := make(map[string]utils.ProxiesApplyMsgItem, len(msg.Proxies))
	for _, proxy := range msg.Proxies {
		if proxy.Name == "" {
			e := fmt.Sprintf("[%s]Proxy name is empty", proxy.Name)
			_ = utils.WriteMsg(conn, utils.MsgProxiesApplyResp, utils.ProxiesApplyRespMsg{Error: e})
			s.Logger.Error(e)
			return fmt.Errorf("proxy name is empty")
		}
		if proxy.Type != "tcp" {
			e := fmt.Sprintf("[%s]Proxy type is invalid", proxy.Name)
			_ = utils.WriteMsg(conn, utils.MsgProxiesApplyResp, utils.ProxiesApplyRespMsg{Error: e})
			s.Logger.Error(e)
			return fmt.Errorf("proxy type is invalid")
		}
		if proxy.RemotePort <= 0 || proxy.RemotePort > 65535 {
			e := fmt.Sprintf("[%s]Proxy remote port is invalid", proxy.Name)
			_ = utils.WriteMsg(conn, utils.MsgProxiesApplyResp, utils.ProxiesApplyRespMsg{Error: e})
			s.Logger.Error(e)
			return fmt.Errorf("proxy remote port is invalid")
		}
		if _, exists := desired[proxy.Name]; exists {
			e := fmt.Sprintf("[%s]Proxy name is duplicated", proxy.Name)
			_ = utils.WriteMsg(conn, utils.MsgProxiesApplyResp, utils.ProxiesApplyRespMsg{Error: e})
			s.Logger.Error(e)
			return fmt.Errorf("proxy name is duplicated")
		}
		desired[proxy.Name] = proxy
	}

	opened := make(map[string]net.Listener)
	openedClose := func() {
		for _, ln := range opened {
			_ = ln.Close()
		}
	}
	for _, proxy := range msg.Proxies {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", proxy.RemotePort))
		if err != nil {
			openedClose()
			e := fmt.Sprintf("Failed to listen on port %d: %v", proxy.RemotePort, err)
			_ = utils.WriteMsg(conn, utils.MsgProxiesApplyResp, utils.ProxiesApplyRespMsg{Error: e})
			s.Logger.Error(e)
			return fmt.Errorf("failed to listen on port %d", proxy.RemotePort)
		}
		opened[proxy.Name] = ln
	}

	for name, listener := range opened {
		ln := listener
		go func() {
			select {
			case <-serverStopCh:
			case <-clientStopCh:
			}
			_ = ln.Close()
		}()
		go s.ProxyAcceptLoop(conn, name, ln, serverStopCh, clientStopCh)
	}

	_ = utils.WriteMsg(conn, utils.MsgProxiesApplyResp, utils.ProxiesApplyRespMsg{Error: ""})
	s.Logger.Info(fmt.Sprintf("Client config applied: clientID=%s", *clientIdP))
	return nil

}

func (s *Server) ProxyAcceptLoop(controlConn net.Conn, proxyName string, listener net.Listener, serverStopCh, clientStopCh chan struct{}) {

	defer func() {
		_ = listener.Close()
		s.Logger.Info(fmt.Sprintf("Proxy stopped: name=%s", proxyName))
	}()

	for {

		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-serverStopCh:
				return
			case <-clientStopCh:
				return
			default:
				s.Logger.Error(fmt.Sprintf("Proxy [%s] accept loop exiting: %v", proxyName, err))
				return
			}
		}

		ipP, region, err := s.ipFilter(conn.RemoteAddr())
		ip := unknownIp
		if ipP != nil {
			ip = *ipP
		}
		if err != nil {
			s.Logger.Warn(fmt.Sprintf("User connection rejected by ip filter: proxy=%s, ip=%s, region=%s, reason=%s", proxyName, ip, region, err.Error()))
			_ = conn.Close()
			continue
		}

		s.Logger.Info(fmt.Sprintf("User connection arrived: proxy=%s, ip=%s, region=%s", proxyName, ip, region))

		go s.bridgeClientConn(controlConn, conn, proxyName, serverStopCh, clientStopCh)

	}

}

func (s *Server) StartWorkConn(workTLS net.Conn, payload []byte) error {
	var msg utils.StartWorkConnMsg
	if err := utils.Decode(payload, &msg); err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to parse StartWorkConnMsg: %v", err))
		return fmt.Errorf("failed to parse StartWorkConnMsg")
	}
	if msg.WorkID == "" {
		_ = workTLS.Close()
		return fmt.Errorf("work_id is empty")
	}
	userConn, ok := s.takePendingIfCertMatches(msg.WorkID, workTLS)
	if !ok {
		s.Logger.Warn(fmt.Sprintf("No matching pending work or client certificate mismatch for work_id=%s", msg.WorkID))
		_ = workTLS.Close()
		return fmt.Errorf("unknown or expired work_id, or certificate mismatch")
	}
	go s.pipe(userConn, workTLS)
	return nil
}

func (s *Server) ipFilter(addr net.Addr) (*string, string, error) {

	host := addr.String()
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}

	ipP, err := utils.NormalizeIP(host)
	if err != nil {
		return nil, unknownIp, fmt.Errorf("failed to parse remote ip")
	}

	geo := s.GeoIP.Lookup(*ipP)
	region := s.formatRegion(geo.Country, geo.Region, geo.City)

	rulesSvc := &RulesIp{DB: s.DB}
	allowed, err := rulesSvc.IsAllowed(*ipP, geo.Country, geo.Region, geo.City)
	if err != nil {
		return nil, region, err
	}

	logsAccess := &LogsAccess{DB: s.DB}
	if err := logsAccess.Record(*ipP, geo.Country, geo.Region, geo.City); err != nil {
		s.Logger.Warn(fmt.Sprintf("Failed to record access log: ip=%s, err=%v", *ipP, err))
	}

	if !allowed {
		return ipP, region, fmt.Errorf("matched deny list")
	}

	return ipP, region, nil

}

func (s *Server) formatRegion(country, region, city string) string {
	parts := make([]string, 0, 3)
	if country != "" {
		parts = append(parts, country)
	}
	if region != "" {
		parts = append(parts, region)
	}
	if city != "" {
		parts = append(parts, city)
	}
	if len(parts) == 0 {
		return unknownIp
	}
	return strings.Join(parts, "/")
}

func (s *Server) bridgeClientConn(controlConn, conn net.Conn, proxyName string, serverStopCh, clientStopCh chan struct{}) {

	certFP, err := s.clientLeafCertSHA256(controlConn)
	if err != nil {
		s.Logger.Warn(fmt.Sprintf("Cannot bind work channel to control TLS cert: %v", err))
		_ = conn.Close()
		return
	}

	workID := uuid.New().String()
	s.registerPendingWork(workID, conn, certFP)

	select {
	case <-serverStopCh:
		if c := s.removePendingWork(workID); c != nil {
			_ = c.Close()
		}
		return
	case <-clientStopCh:
		if c := s.removePendingWork(workID); c != nil {
			_ = c.Close()
		}
		return
	default:
	}

	msg := utils.NewWorkConnMsg{
		WorkID:    workID,
		ProxyName: proxyName,
	}
	if err := utils.WriteMsg(controlConn, utils.MsgNewWorkConn, msg); err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to notify client (NewWorkConn): %v", err))
		if c := s.removePendingWork(workID); c != nil {
			_ = c.Close()
		}
		return
	}

}

func (s *Server) registerPendingWork(workID string, userConn net.Conn, certFP [sha256.Size]byte) {

	s.pendingMu.Lock()
	if s.pendingWork == nil {
		s.pendingWork = make(map[string]*pendingWorkItem)
	}
	s.pendingWork[workID] = &pendingWorkItem{
		userConn: userConn,
		certFP:   certFP,
	}
	s.pendingMu.Unlock()

	time.AfterFunc(15*time.Second, func() {
		s.pendingMu.Lock()
		p, ok := s.pendingWork[workID]
		if ok {
			delete(s.pendingWork, workID)
		}
		s.pendingMu.Unlock()
		if ok {
			_ = p.userConn.Close()
			s.Logger.Warn(fmt.Sprintf("Timed out waiting for work channel; closed user connection: workID=%s", workID))
		}
	})

}

func (s *Server) removePendingWork(workID string) net.Conn {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	if s.pendingWork == nil {
		return nil
	}
	p, ok := s.pendingWork[workID]
	if !ok {
		return nil
	}
	delete(s.pendingWork, workID)
	return p.userConn
}

func (s *Server) takePendingIfCertMatches(workID string, workTLS net.Conn) (net.Conn, bool) {
	workFP, err := s.clientLeafCertSHA256(workTLS)
	if err != nil {
		s.Logger.Warn(fmt.Sprintf("StartWorkConn: read work TLS client cert: %v", err))
		return nil, false
	}
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	if s.pendingWork == nil {
		return nil, false
	}
	p, ok := s.pendingWork[workID]
	if !ok {
		return nil, false
	}
	if subtle.ConstantTimeCompare(p.certFP[:], workFP[:]) != 1 {
		s.Logger.Warn(fmt.Sprintf("StartWorkConn rejected: client certificate does not match control channel (work_id=%s)", workID))
		return nil, false
	}
	delete(s.pendingWork, workID)
	return p.userConn, true
}

func (s *Server) clientLeafCertSHA256(conn net.Conn) ([sha256.Size]byte, error) {
	var z [sha256.Size]byte
	tc, ok := conn.(*tls.Conn)
	if !ok {
		return z, fmt.Errorf("not a TLS connection")
	}
	state := tc.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return z, fmt.Errorf("no peer certificate")
	}
	return sha256.Sum256(state.PeerCertificates[0].Raw), nil
}

func (s *Server) pipe(a, b net.Conn) {
	defer func() { _ = a.Close() }()
	defer func() { _ = b.Close() }()
	done := make(chan struct{}, 2)
	copyFn := func(dst, src net.Conn) {
		_, _ = io.Copy(dst, src)
		done <- struct{}{}
	}
	go copyFn(a, b)
	go copyFn(b, a)
	<-done
}
