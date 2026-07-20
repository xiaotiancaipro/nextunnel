package services

import (
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/google/uuid"
	"github.com/xiaotiancaipro/nextunnel/internal/server/clients"
	"github.com/xiaotiancaipro/nextunnel/internal/server/models"
	sharedcerts "github.com/xiaotiancaipro/nextunnel/internal/shared/certs"
	sharedprotocol "github.com/xiaotiancaipro/nextunnel/internal/shared/protocol"
	"go.uber.org/zap"
)

type Session struct {
	Logger              *zap.Logger
	Database            *clients.Database
	ClientService       *Client
	ClientProxyService  *ClientProxy
	ProxyBrokerService  *ProxyBroker
	AccessFilterService *AccessFilter
}

func (s *Session) Login(conn net.Conn, payload []byte) (clientID, runID string, err error) {
	var loginMsg sharedprotocol.LoginMsg
	if err := sharedprotocol.Decode(payload, &loginMsg); err != nil {
		s.Logger.Error(fmt.Sprintf("failed to parse login msg: %v", err))
		return "", "", fmt.Errorf("failed to parse LoginMsg")
	}

	if loginMsg.Id == "" {
		_ = sharedprotocol.WriteMsg(conn, sharedprotocol.MsgLoginResp, sharedprotocol.LoginRespMsg{Error: "client_id cannot be empty"})
		s.Logger.Warn("client login rejected: client_id is empty")
		return "", "", fmt.Errorf("client_id is empty")
	}
	if _, err := s.ClientService.ResolveClientId(s.Database.DB, loginMsg.Id); err != nil {
		_ = sharedprotocol.WriteMsg(conn, sharedprotocol.MsgLoginResp, sharedprotocol.LoginRespMsg{Error: "client_id is invalid"})
		s.Logger.Warn(fmt.Sprintf("client login rejected: client_id=%s is invalid", loginMsg.Id))
		return "", "", fmt.Errorf("client_id is invalid")
	}

	runID = uuid.New().String()
	if err := sharedprotocol.WriteMsg(conn, sharedprotocol.MsgLoginResp, sharedprotocol.LoginRespMsg{RunID: runID}); err != nil {
		s.Logger.Error(fmt.Sprintf("failed to send login response: client_id=%s, err=%v", loginMsg.Id, err))
		return "", "", fmt.Errorf("failed to send LoginResp")
	}

	s.Logger.Info(fmt.Sprintf("client logged in: client_id=%s, run_id=%s, remote=%s", loginMsg.Id, runID, conn.RemoteAddr()))
	return loginMsg.Id, runID, nil
}

func (s *Session) SetClientProxiesOffline(clientID string) error {
	clientUUID, err := s.ClientService.ResolveClientId(s.Database.DB, clientID)
	if err != nil {
		return err
	}
	return s.ClientProxyService.SetAllOffline(clientUUID)
}

func (s *Session) ProxiesApply(conn net.Conn, ctrlWriteMu *sync.Mutex, payload []byte, clientID string, listeners *ProxyListeners, serverStopCh, clientStopCh chan struct{}) error {
	replyErr := func(e string) {
		_ = sharedprotocol.WriteMsgWithLock(ctrlWriteMu, conn, sharedprotocol.MsgProxiesApplyResp, sharedprotocol.ProxiesApplyRespMsg{Error: e})
		s.Logger.Error(fmt.Sprintf("proxies apply rejected: client_id=%s, reason=%s", clientID, e))
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

	clientUUID, err := s.ClientService.ResolveClientId(s.Database.DB, clientID)
	if err != nil {
		replyErr("client_id is invalid")
		return fmt.Errorf("client_id is invalid")
	}

	var client models.Client
	if err := s.Database.DB.Where("id = ?", clientUUID).First(&client).Error; err != nil {
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

	opened, err := listeners.reconcile(desired)
	if err != nil {
		replyErr(fmt.Sprintf("Failed to listen: %v", err))
		return err
	}

	for name, listener := range opened {
		ln := listener
		proxy := desired[name]
		s.Logger.Info(fmt.Sprintf("proxy listener opened: client_id=%s, name=%s, remote_port=%d", clientID, name, proxy.RemotePort))
		go func() {
			select {
			case <-serverStopCh:
			case <-clientStopCh:
			}
			_ = ln.Close()
		}()
		go s.proxyAcceptLoop(conn, ctrlWriteMu, clientID, name, ln, serverStopCh, clientStopCh)
	}

	_ = sharedprotocol.WriteMsgWithLock(ctrlWriteMu, conn, sharedprotocol.MsgProxiesApplyResp, sharedprotocol.ProxiesApplyRespMsg{Error: ""})
	s.Logger.Info(fmt.Sprintf("client config applied: client_id=%s, proxies=%d, newly_opened=%d", clientID, len(desired), len(opened)))
	return nil
}

func (s *Session) proxyAcceptLoop(controlConn net.Conn, ctrlWriteMu *sync.Mutex, clientID, proxyName string, listener net.Listener, serverStopCh, clientStopCh chan struct{}) {
	defer func() {
		_ = listener.Close()
		s.Logger.Info(fmt.Sprintf("proxy stopped: client_id=%s, name=%s", clientID, proxyName))
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			select {
			case <-serverStopCh:
				return
			case <-clientStopCh:
				return
			default:
				s.Logger.Error(fmt.Sprintf("proxy accept loop exiting: name=%s, err=%v", proxyName, err))
				return
			}
		}

		ip, region, err := s.AccessFilterService.Check(conn.RemoteAddr(), clientID, proxyName)
		if err != nil {
			s.Logger.Warn(fmt.Sprintf("user connection rejected: proxy=%s, ip=%s, region=%s, reason=%s", proxyName, ip, region, err.Error()))
			_ = conn.Close()
			continue
		}

		s.Logger.Info(fmt.Sprintf("user connection arrived: proxy=%s, ip=%s, region=%s", proxyName, ip, region))
		go s.bridgeClientConn(controlConn, ctrlWriteMu, conn, proxyName, serverStopCh, clientStopCh)
	}
}

func (s *Session) bridgeClientConn(controlConn net.Conn, ctrlWriteMu *sync.Mutex, conn net.Conn, proxyName string, serverStopCh, clientStopCh chan struct{}) {
	certFP, err := sharedcerts.ClientLeafCertSHA256(controlConn)
	if err != nil {
		s.Logger.Warn(fmt.Sprintf("cannot bind work channel to control tls cert: %v", err))
		_ = conn.Close()
		return
	}

	workID := uuid.New().String()
	s.ProxyBrokerService.Register(workID, conn, certFP)

	select {
	case <-serverStopCh:
		if c := s.ProxyBrokerService.Remove(workID); c != nil {
			_ = c.Close()
		}
		return
	case <-clientStopCh:
		if c := s.ProxyBrokerService.Remove(workID); c != nil {
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
		s.Logger.Error(fmt.Sprintf("failed to notify client new work conn: proxy=%s, work_id=%s, err=%v", proxyName, workID, err))
		if c := s.ProxyBrokerService.Remove(workID); c != nil {
			_ = c.Close()
		}
		return
	}
	s.Logger.Info(fmt.Sprintf("work conn requested: proxy=%s, work_id=%s", proxyName, workID))
}
