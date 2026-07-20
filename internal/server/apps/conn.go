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
	Logger     *zap.Logger
	Services   *services.Services
	listener   net.Listener
	listenerMu sync.Mutex
	stopCh     chan struct{}
	stopOnce   sync.Once
}

func (a *Conn) Init() error {
	a.stopCh = make(chan struct{})
	return nil
}

func (a *Conn) Start() error {

	listener, err := a.Services.Server.Listen()
	if err != nil {
		return err
	}
	a.listenerMu.Lock()
	a.listener = listener
	a.listenerMu.Unlock()

	a.Logger.Info("conn server listening on " + listener.Addr().String())

	tlsConfig, err := a.Services.Tls.Init()
	if err != nil {
		a.Logger.Error(fmt.Sprintf("Failed to initialize TLS connection: %v", err))
		return err
	}
	a.Logger.Info("TLS connection established")

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
			a.Logger.Error(fmt.Sprintf("Failed to accept connection: %v", err))
			return err
		}
		go a.handle(connRaw, tlsConfig)
	}

}

func (a *Conn) Stop(_ context.Context) error {
	var closeErr error
	a.stopOnce.Do(func() {
		if a.stopCh != nil {
			close(a.stopCh)
		}
		a.listenerMu.Lock()
		ln := a.listener
		a.listener = nil
		a.listenerMu.Unlock()
		if ln != nil {
			closeErr = ln.Close()
		}
	})
	return closeErr
}

func (a *Conn) handle(connRaw net.Conn, tlsConfig *tls.Config) {
	conn, err := a.Services.Server.EstablishConn(connRaw, tlsConfig)
	if err != nil {
		a.Logger.Error(fmt.Sprintf("Failed to incoming TLS connection: %v", err))
		_ = connRaw.Close()
		return
	}
	a.accepted(conn)
}

func (a *Conn) accepted(conn net.Conn) {

	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	msgType, payload, err := sharedprotocol.ReadMsg(conn)
	if err != nil {
		a.Logger.Error(fmt.Sprintf("Failed to read first message [%s]: %v", conn.RemoteAddr(), err))
		_ = conn.Close()
		return
	}
	_ = conn.SetDeadline(time.Time{})

	switch msgType {
	case sharedprotocol.MsgLogin:
		clientIdP, runIdP, err := a.Services.Server.Login(conn, payload)
		if err != nil {
			a.Logger.Error(fmt.Sprintf("Failed to login: %v", err))
			_ = conn.Close()
			return
		}
		clientID := *clientIdP
		clientStopCh := make(chan struct{})
		defer func() {
			close(clientStopCh)
			if err := a.Services.Server.SetClientProxiesOffline(clientID); err != nil {
				a.Logger.Warn(fmt.Sprintf("Failed to mark client proxies offline: clientID=%s, err=%v", clientID, err))
			}
			_ = conn.Close()
		}()
		var ctrlWriteMu sync.Mutex
		for {
			msgType_, payload_, err := sharedprotocol.ReadMsg(conn)
			if err != nil {
				a.Logger.Error(fmt.Sprintf("Client control connection disconnected, clientID=%s, runID=%s: %v", *clientIdP, *runIdP, err))
				return
			}
			switch msgType_ {
			case sharedprotocol.MsgProxiesApply:
				if err := a.Services.Server.ProxiesApply(conn, &ctrlWriteMu, payload_, clientIdP, a.stopCh, clientStopCh); err != nil {
					a.Logger.Error(fmt.Sprintf("Failed to apply proxies: %v", err))
					return
				}
			case sharedprotocol.MsgHeartbeat:
				if err := sharedprotocol.WriteMsgWithLock(&ctrlWriteMu, conn, sharedprotocol.MsgHeartbeatResp, sharedprotocol.HeartbeatRespMsg{}); err != nil {
					a.Logger.Error(fmt.Sprintf("Failed to send HeartbeatRespMsg: %v", err))
					return
				}
			default:
				a.Logger.Error(fmt.Sprintf("Unknown message received on control connection 0x%02x runID=%s", msgType_, *runIdP))
			}
		}
	case sharedprotocol.MsgStartWorkConn:
		if err := a.Services.Server.StartWorkConn(conn, payload); err != nil {
			a.Logger.Error(fmt.Sprintf("Failed to start work connection: %v", err))
			_ = conn.Close()
			return
		}
		return
	default:
		a.Logger.Error(fmt.Sprintf("Unknown first message type 0x%02x [%s]", msgType, conn.RemoteAddr()))
		_ = conn.Close()
	}

}
