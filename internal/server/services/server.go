package services

import (
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/xiaotiancaipro/nextunnel/internal/server/clients"
	"github.com/xiaotiancaipro/nextunnel/internal/server/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/server/models"
	sharednetwork "github.com/xiaotiancaipro/nextunnel/internal/shared/network"
	sharedprotocol "github.com/xiaotiancaipro/nextunnel/internal/shared/protocol"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const ruleCacheTTL = 10 * time.Second

type Server struct {
	Config             *configs.Server
	Logger             *zap.Logger
	DB                 *gorm.DB
	IPLocation         *clients.IPLocation
	ClientService      *Client
	ClientProxyService *ClientProxy
	AccessRuleService  *AccessRule
	AccessLogService   *AccessLog
	pendingMu          sync.Mutex
	pendingWork        map[string]*pendingWorkItem
	ruleCacheMu        sync.RWMutex
	ruleCache          []models.AccessRule
	ruleCacheAt        time.Time
}

type pendingWorkItem struct {
	userConn net.Conn
	certFP   [sha256.Size]byte
}

func (s *Server) Listen() (net.Listener, error) {
	addr := fmt.Sprintf(":%d", s.Config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to listen on %s: %v", addr, err))
		return nil, fmt.Errorf("failed to listen")
	}
	return listener, nil
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

	var loginMsg sharedprotocol.LoginMsg
	if err := sharedprotocol.Decode(payload, &loginMsg); err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to parse LoginMsg: %v", err))
		return nil, nil, fmt.Errorf("failed to parse LoginMsg")
	}

	if loginMsg.Id == "" {
		_ = sharedprotocol.WriteMsg(conn, sharedprotocol.MsgLoginResp, sharedprotocol.LoginRespMsg{Error: "client_id cannot be empty"})
		return nil, nil, fmt.Errorf("client_id is empty")
	}
	if _, err := s.ClientService.ResolveClientId(s.DB, loginMsg.Id); err != nil {
		_ = sharedprotocol.WriteMsg(conn, sharedprotocol.MsgLoginResp, sharedprotocol.LoginRespMsg{Error: "client_id is invalid"})
		return nil, nil, fmt.Errorf("client_id is invalid")
	}

	runID := uuid.New().String()
	if err := sharedprotocol.WriteMsg(conn, sharedprotocol.MsgLoginResp, sharedprotocol.LoginRespMsg{RunID: runID}); err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to send LoginResp: %v", err))
		return nil, nil, fmt.Errorf("failed to send LoginResp")
	}

	return &loginMsg.Id, &runID, nil

}

func (s *Server) ProxiesApply(conn net.Conn, ctrlWriteMu *sync.Mutex, payload []byte, clientIdP *string, serverStopCh, clientStopCh chan struct{}) error {

	replyErr := func(e string) {
		_ = sharedprotocol.WriteMsgWithLock(ctrlWriteMu, conn, sharedprotocol.MsgProxiesApplyResp, sharedprotocol.ProxiesApplyRespMsg{Error: e})
		s.Logger.Error(e)
	}

	var msg sharedprotocol.ProxiesApplyMsg
	if err := sharedprotocol.Decode(payload, &msg); err != nil {
		replyErr(fmt.Sprintf("failed to parse ApplyConfigMsg: %v", err))
		return fmt.Errorf("failed to parse ApplyConfigMsg")
	}

	desired := make(map[string]sharedprotocol.ProxiesApplyMsgItem, len(msg.Proxies))
	usedPorts := make(map[int]string, len(msg.Proxies))
	for _, proxy := range msg.Proxies {
		if proxy.Name == "" {
			replyErr("Proxy name is empty")
			return fmt.Errorf("proxy name is empty")
		}
		if proxy.Type != "tcp" {
			replyErr(fmt.Sprintf("[%s]Proxy type is invalid", proxy.Name))
			return fmt.Errorf("proxy type is invalid")
		}
		if proxy.LocalIP == "" {
			replyErr(fmt.Sprintf("[%s] local_ip is empty", proxy.Name))
			return fmt.Errorf("local_ip is empty")
		}
		if proxy.LocalPort < 1 || proxy.LocalPort > 65535 {
			replyErr(fmt.Sprintf("[%s] local_port is invalid", proxy.Name))
			return fmt.Errorf("local_port is invalid")
		}
		if proxy.RemotePort < 1 || proxy.RemotePort > 65535 {
			replyErr(fmt.Sprintf("[%s] remote_port is invalid", proxy.Name))
			return fmt.Errorf("remote_port is invalid")
		}
		if _, exists := desired[proxy.Name]; exists {
			replyErr(fmt.Sprintf("[%s]Proxy name is duplicated", proxy.Name))
			return fmt.Errorf("proxy name is duplicated")
		}
		if other, exists := usedPorts[proxy.RemotePort]; exists {
			replyErr(fmt.Sprintf("[%s]Proxy remote port %d is already requested by [%s]", proxy.Name, proxy.RemotePort, other))
			return fmt.Errorf("proxy remote port is duplicated")
		}
		desired[proxy.Name] = proxy
		usedPorts[proxy.RemotePort] = proxy.Name
	}

	clientUUID, err := s.ClientService.ResolveClientId(s.DB, *clientIdP)
	if err != nil {
		replyErr("client_id is invalid")
		return fmt.Errorf("client_id is invalid")
	}

	var client models.Client
	if err := s.DB.Where("id = ?", clientUUID).First(&client).Error; err != nil {
		replyErr("client_id is invalid")
		return fmt.Errorf("client not found")
	}
	for name, proxy := range desired {
		if !s.ClientService.ClientPortAllowed(client, proxy.RemotePort) {
			replyErr(fmt.Sprintf("[%s] remote port %d is outside allocated range %d-%d", name, proxy.RemotePort, client.PortStart, client.PortEnd))
			return fmt.Errorf("remote port out of range")
		}
	}

	if err := s.ClientProxyService.SyncFromApply(clientUUID, desired); err != nil {
		replyErr(fmt.Sprintf("failed to sync proxies: %v", err))
		return err
	}

	opened := make(map[string]net.Listener)
	openedClose := func() {
		for _, ln := range opened {
			_ = ln.Close()
		}
	}
	for name, proxy := range desired {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", proxy.RemotePort))
		if err != nil {
			openedClose()
			replyErr(fmt.Sprintf("Failed to listen on port %d: %v", proxy.RemotePort, err))
			return fmt.Errorf("failed to listen on port %d", proxy.RemotePort)
		}
		opened[name] = ln
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
		go s.proxyAcceptLoop(conn, ctrlWriteMu, *clientIdP, name, ln, serverStopCh, clientStopCh)
	}

	_ = sharedprotocol.WriteMsgWithLock(ctrlWriteMu, conn, sharedprotocol.MsgProxiesApplyResp, sharedprotocol.ProxiesApplyRespMsg{Error: ""})
	s.Logger.Info(fmt.Sprintf("Client config applied: clientID=%s, proxies=%d", *clientIdP, len(opened)))
	return nil

}

func (s *Server) SetClientProxiesOffline(clientId string) error {
	clientUUID, err := s.ClientService.ResolveClientId(s.DB, clientId)
	if err != nil {
		return err
	}
	return s.ClientProxyService.SetAllOffline(clientUUID)
}

func (s *Server) StartWorkConn(workTLS net.Conn, payload []byte) error {
	var msg sharedprotocol.StartWorkConnMsg
	if err := sharedprotocol.Decode(payload, &msg); err != nil {
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
	go sharednetwork.Pipe(userConn, workTLS)
	return nil
}

func (s *Server) proxyAcceptLoop(controlConn net.Conn, ctrlWriteMu *sync.Mutex, clientId, proxyName string, listener net.Listener, serverStopCh, clientStopCh chan struct{}) {

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

		ipP, region, err := s.ipFilter(conn.RemoteAddr(), clientId, proxyName)
		ip := sharednetwork.UnknownIP
		if ipP != nil {
			ip = *ipP
		}
		if err != nil {
			s.Logger.Warn(fmt.Sprintf("User connection rejected by ip filter: proxy=%s, ip=%s, region=%s, reason=%s", proxyName, ip, region, err.Error()))
			_ = conn.Close()
			continue
		}

		s.Logger.Info(fmt.Sprintf("User connection arrived: proxy=%s, ip=%s, region=%s", proxyName, ip, region))

		go s.bridgeClientConn(controlConn, ctrlWriteMu, conn, proxyName, serverStopCh, clientStopCh)

	}

}

func (s *Server) ipFilter(addr net.Addr, clientId, proxyName string) (*string, string, error) {

	host := addr.String()
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}

	ipP, err := sharednetwork.NormalizeIP(host)
	if err != nil {
		return nil, sharednetwork.UnknownIP, fmt.Errorf("failed to parse remote ip")
	}

	geo := s.IPLocation.Lookup(*ipP)
	region := s.formatRegion(geo.Country, geo.Region, geo.City)
	isLocal := sharednetwork.IsLocalIP(*ipP)

	rules, err := s.cachedRules()
	if err != nil {
		return nil, region, err
	}
	allowed := s.AccessRuleService.evaluate(rules, *ipP, geo.Country, geo.Region, geo.City, isLocal)

	status := int16(0)
	if allowed {
		status = 1
	}
	if err := s.AccessLogService.Record(clientId, proxyName, *ipP, geo.Country, geo.Region, geo.City, isLocal, status); err != nil {
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
		return sharednetwork.UnknownIP
	}
	return strings.Join(parts, "/")
}

func (s *Server) bridgeClientConn(controlConn net.Conn, ctrlWriteMu *sync.Mutex, conn net.Conn, proxyName string, serverStopCh, clientStopCh chan struct{}) {

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

	msg := sharedprotocol.NewWorkConnMsg{
		WorkID:    workID,
		ProxyName: proxyName,
	}
	if err := sharedprotocol.WriteMsgWithLock(ctrlWriteMu, controlConn, sharedprotocol.MsgNewWorkConn, msg); err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to notify client (NewWorkConn): %v", err))
		if c := s.removePendingWork(workID); c != nil {
			_ = c.Close()
		}
		return
	}

}

func (s *Server) cachedRules() ([]models.AccessRule, error) {

	s.ruleCacheMu.RLock()
	if s.ruleCache != nil && time.Since(s.ruleCacheAt) < ruleCacheTTL {
		rules := s.ruleCache
		s.ruleCacheMu.RUnlock()
		return rules, nil
	}
	s.ruleCacheMu.RUnlock()

	var rules []models.AccessRule
	if err := s.DB.Where("is_delete = ?", false).Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("failed to query access_rules: %w", err)
	}

	s.ruleCacheMu.Lock()
	s.ruleCache = rules
	s.ruleCacheAt = time.Now()
	s.ruleCacheMu.Unlock()

	return rules, nil

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
