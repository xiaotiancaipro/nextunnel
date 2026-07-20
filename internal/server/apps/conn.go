package apps

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/xiaotiancaipro/nextunnel/internal/server/services"
	sharedprotocol "github.com/xiaotiancaipro/nextunnel/internal/shared/protocol"
	"go.uber.org/zap"
)

type Conn struct {
	Logger   *zap.Logger
	Services *services.Services
	listener net.Listener
	mu       sync.Mutex
	stopCh   chan struct{}
	stopOnce sync.Once
}

func (a *Conn) Init() error {
	a.stopCh = make(chan struct{})
	return nil
}

func (a *Conn) Start() error {
	listener, err := a.Services.Listener.Listen()
	if err != nil {
		return err
	}
	a.mu.Lock()
	a.listener = listener
	a.mu.Unlock()

	a.Logger.Info("conn server listening on " + listener.Addr().String())

	tlsConfig, err := a.Services.Tls.Init()
	if err != nil {
		a.Logger.Error(fmt.Sprintf("failed to initialize tls: %v", err))
		return err
	}
	a.Logger.Info("tls config loaded")

	for {
		connRaw, err := listener.Accept()
		if err != nil {
			select {
			case <-a.stopCh:
				return nil
			default:
			}
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			a.Logger.Error(fmt.Sprintf("failed to accept connection: %v", err))
			return err
		}
		go a.handle(connRaw, tlsConfig)
	}

}

func (a *Conn) Stop(_ context.Context) error {
	var closeErr error
	a.stopOnce.Do(func() {
		a.Logger.Info("conn server stopping")
		if a.stopCh != nil {
			close(a.stopCh)
		}
		a.mu.Lock()
		ln := a.listener
		a.listener = nil
		a.mu.Unlock()
		if ln != nil {
			closeErr = ln.Close()
		}
	})
	return closeErr
}

func (a *Conn) handle(connRaw net.Conn, tlsConfig *tls.Config) {
	conn, err := a.Services.Listener.Establish(connRaw, tlsConfig)
	if err != nil {
		a.Logger.Warn(fmt.Sprintf("failed to establish tls connection from %s: %v", connRaw.RemoteAddr(), err))
		_ = connRaw.Close()
		return
	}
	a.dispatch(conn)
}

func (a *Conn) dispatch(conn net.Conn) {
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	msgType, payload, err := sharedprotocol.ReadMsg(conn)
	if err != nil {
		a.Logger.Warn(fmt.Sprintf("failed to read first message from %s: %v", conn.RemoteAddr(), err))
		_ = conn.Close()
		return
	}
	_ = conn.SetDeadline(time.Time{})
	switch msgType {
	case sharedprotocol.MsgLogin:
		a.serveControl(conn, payload)
	case sharedprotocol.MsgStartWorkConn:
		if err := a.Services.ProxyBroker.StartWorkConn(conn, payload); err != nil {
			a.Logger.Warn(fmt.Sprintf("failed to start work connection: %v", err))
			_ = conn.Close()
			return
		}
	default:
		a.Logger.Warn(fmt.Sprintf("unknown first message type 0x%02x from %s", msgType, conn.RemoteAddr()))
		_ = conn.Close()
	}
}

func (a *Conn) serveControl(conn net.Conn, loginPayload []byte) {

	clientID, runID, err := a.Services.Session.Login(conn, loginPayload)
	if err != nil {
		a.Logger.Warn(fmt.Sprintf("client login failed from %s: %v", conn.RemoteAddr(), err))
		_ = conn.Close()
		return
	}

	clientStopCh := make(chan struct{})
	proxyListeners := new(services.ProxyListeners)
	defer func() {
		close(clientStopCh)
		proxyListeners.CloseAll()
		if err := a.Services.Session.SetClientProxiesOffline(clientID); err != nil {
			a.Logger.Warn(fmt.Sprintf("failed to mark client proxies offline: client_id=%s, err=%v", clientID, err))
		} else {
			a.Logger.Info(fmt.Sprintf("client proxies marked offline: client_id=%s", clientID))
		}
		_ = conn.Close()
	}()

	var ctrlWriteMu sync.Mutex
	for {
		msgType, payload, err := sharedprotocol.ReadMsg(conn)
		if err != nil {
			a.Logger.Info(fmt.Sprintf("client control disconnected: client_id=%s, run_id=%s, err=%v", clientID, runID, err))
			return
		}
		switch msgType {
		case sharedprotocol.MsgProxiesApply:
			if err := a.Services.Session.ProxiesApply(conn, &ctrlWriteMu, payload, clientID, proxyListeners, a.stopCh, clientStopCh); err != nil {
				a.Logger.Error(fmt.Sprintf("failed to apply proxies: client_id=%s, err=%v", clientID, err))
				return
			}
		case sharedprotocol.MsgHeartbeat:
			if err := sharedprotocol.WriteMsgWithLock(&ctrlWriteMu, conn, sharedprotocol.MsgHeartbeatResp, sharedprotocol.HeartbeatRespMsg{}); err != nil {
				a.Logger.Warn(fmt.Sprintf("failed to send heartbeat response: client_id=%s, err=%v", clientID, err))
				return
			}
		default:
			a.Logger.Warn(fmt.Sprintf("unknown control message 0x%02x: client_id=%s, run_id=%s", msgType, clientID, runID))
		}
	}

}
