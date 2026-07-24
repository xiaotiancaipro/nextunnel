package apps

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/xiaotiancaipro/nextunnel/internal/server/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/server/services"
	sharedprotocol "github.com/xiaotiancaipro/nextunnel/internal/shared/protocol"
	"go.uber.org/zap"
)

const (
	handshakeTimeout       = 10 * time.Second
	controlReadIdleTimeout = 90 * time.Second // client heartbeats every 30s
)

type Conn struct {
	Config    *configs.Configs
	Logger    *zap.Logger
	Services  *services.Services
	listener  net.Listener
	mu        sync.Mutex
	tlsConf   *tls.Config
	stopCh    chan struct{}
	stopOnce  sync.Once
	ctrlConns map[net.Conn]struct{}
	ctrlWg    sync.WaitGroup
}

func (a *Conn) Init() error {
	a.stopCh = make(chan struct{})
	tlsConfig, err := a.Services.Tls.Init()
	if err != nil {
		a.Logger.Error(fmt.Sprintf("failed to initialize tls: %v", err))
		return err
	}
	a.tlsConf = tlsConfig
	a.Logger.Info("tls config loaded")
	return nil
}

func (a *Conn) Start() error {

	addr := fmt.Sprintf(":%d", a.Config.Server.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		a.Logger.Error(fmt.Sprintf("failed to listen on %s: %v", addr, err))
		return fmt.Errorf("failed to listen")
	}
	a.mu.Lock()
	a.listener = listener
	a.mu.Unlock()
	a.Logger.Info("[conn] listening on " + listener.Addr().String())

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
		go a.handle(connRaw)
	}

}

func (a *Conn) Stop(ctx context.Context) error {
	var closeErr error
	a.stopOnce.Do(func() {
		a.Logger.Info("conn server stopping")
		if a.stopCh != nil {
			close(a.stopCh)
		}
		a.mu.Lock()
		ln := a.listener
		a.listener = nil
		for c := range a.ctrlConns {
			_ = c.Close()
		}
		a.mu.Unlock()
		if ln != nil {
			closeErr = ln.Close()
		}
		done := make(chan struct{})
		go func() {
			a.ctrlWg.Wait()
			close(done)
		}()
		select {
		case <-done:
			a.Logger.Info("all control connections cleaned up")
		case <-ctx.Done():
			a.Logger.Warn("timed out waiting for control connections to clean up")
		}
	})
	return closeErr
}

func (a *Conn) handle(connRaw net.Conn) {

	defer func() {
		if r := recover(); r != nil {
			a.Logger.Error(fmt.Sprintf("panic in conn handler from %s: %v", connRaw.RemoteAddr(), r))
		}
	}()

	conn := tls.Server(connRaw, a.tlsConf)
	owned := false
	defer func() {
		if !owned {
			_ = conn.Close()
		}
	}()

	_ = conn.SetDeadline(time.Now().Add(handshakeTimeout))
	if err := conn.Handshake(); err != nil {
		a.Logger.Warn(fmt.Sprintf("failed to establish tls connection from %s: %v", connRaw.RemoteAddr(), err))
		return
	}

	_ = conn.SetDeadline(time.Now().Add(handshakeTimeout))
	msgType, payload, err := sharedprotocol.ReadMsg(conn)
	if err != nil {
		a.Logger.Warn(fmt.Sprintf("failed to read first message from %s: %v", conn.RemoteAddr(), err))
		return
	}
	_ = conn.SetDeadline(time.Time{})

	switch msgType {
	case sharedprotocol.MsgLogin:
		owned = true
		a.serveControl(conn, payload)
	case sharedprotocol.MsgStartWorkConn:
		if err := a.Services.ProxyBroker.StartWorkConn(conn, payload); err != nil {
			a.Logger.Warn(fmt.Sprintf("failed to start work connection: %v", err))
			return
		}
		owned = true
	default:
		a.Logger.Warn(fmt.Sprintf("unknown first message type 0x%02x from %s", msgType, conn.RemoteAddr()))
	}

}

func (a *Conn) serveControl(conn net.Conn, loginPayload []byte) {

	if !a.registerCtrlConn(conn) {
		_ = conn.Close()
		return
	}
	defer a.unregisterCtrlConn(conn)

	defer func() { _ = conn.Close() }()

	_ = conn.SetDeadline(time.Now().Add(handshakeTimeout))
	clientID, runID, err := a.Services.Session.Login(conn, loginPayload)
	_ = conn.SetDeadline(time.Time{})
	if err != nil {
		a.Logger.Warn(fmt.Sprintf("client login failed from %s: %v", conn.RemoteAddr(), err))
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
	}()

	var ctrlWriteMu sync.Mutex
	for {
		if err := conn.SetReadDeadline(time.Now().Add(controlReadIdleTimeout)); err != nil {
			a.Logger.Warn(fmt.Sprintf("failed to set control read deadline: client_id=%s, err=%v", clientID, err))
			return
		}
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

func (a *Conn) registerCtrlConn(conn net.Conn) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	select {
	case <-a.stopCh:
		return false
	default:
	}
	if a.ctrlConns == nil {
		a.ctrlConns = make(map[net.Conn]struct{})
	}
	a.ctrlConns[conn] = struct{}{}
	a.ctrlWg.Add(1)
	return true
}

func (a *Conn) unregisterCtrlConn(conn net.Conn) {
	a.mu.Lock()
	delete(a.ctrlConns, conn)
	a.mu.Unlock()
	a.ctrlWg.Done()
}
