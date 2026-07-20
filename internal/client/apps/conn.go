package apps

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/xiaotiancaipro/nextunnel/internal/client/services"
	sharedprotocol "github.com/xiaotiancaipro/nextunnel/internal/shared/protocol"
	"go.uber.org/zap"
)

type Conn struct {
	Logger   *zap.Logger
	Services *services.Services
	ctrlMu   sync.Mutex
	ctrlConn net.Conn
	stopCh   chan struct{}
	stopOnce sync.Once
}

type controlMsg struct {
	msgType byte
	payload []byte
	err     error
}

func (a *Conn) Init() error {
	a.stopCh = make(chan struct{})
	return nil
}

func (a *Conn) Start() error {
	tlsCfg, err := a.Services.Tls.Init()
	if err != nil {
		a.Logger.Error(fmt.Sprintf("failed to initialize tls: %v", err))
		return err
	}
	a.Logger.Info("tls config loaded")

	a.Services.Client.DialWork = func() (net.Conn, error) {
		return a.Services.Server.Dial(tlsCfg)
	}

	const (
		reconnectMinDelay = 2 * time.Second
		reconnectMaxDelay = 30 * time.Second
	)
	nextRetryDelay := reconnectMinDelay

	for {
		select {
		case <-a.stopCh:
			return nil
		default:
		}

		stopped := a.runSession(&nextRetryDelay, reconnectMinDelay)
		if stopped {
			return nil
		}

		a.Logger.Warn(fmt.Sprintf("session ended, reconnecting in %s", nextRetryDelay))
		select {
		case <-a.stopCh:
			return nil
		case <-time.After(nextRetryDelay):
		}

		if grow := nextRetryDelay * 2; grow > reconnectMaxDelay {
			nextRetryDelay = reconnectMaxDelay
		} else {
			nextRetryDelay = grow
		}
	}
}

func (a *Conn) Stop(_ context.Context) error {
	a.stopOnce.Do(func() {
		a.Logger.Info("client connection stopping")
		if a.stopCh != nil {
			close(a.stopCh)
		}
		a.ctrlMu.Lock()
		c := a.ctrlConn
		a.ctrlConn = nil
		a.ctrlMu.Unlock()
		if c != nil {
			_ = c.Close()
		}
	})
	return nil
}

func (a *Conn) runSession(nextRetryDelay *time.Duration, reconnectMin time.Duration) (stopped bool) {
	addr := a.Services.Server.AddrStr()
	conn, err := a.Services.Client.DialWork()
	if err != nil {
		a.Logger.Warn(fmt.Sprintf("failed to connect to server %s: %v", addr, err))
		return false
	}

	a.ctrlMu.Lock()
	a.ctrlConn = conn
	a.ctrlMu.Unlock()
	defer func() {
		a.ctrlMu.Lock()
		if a.ctrlConn == conn {
			a.ctrlConn = nil
		}
		a.ctrlMu.Unlock()
		_ = conn.Close()
	}()

	a.Logger.Info(fmt.Sprintf("connected to server %s", addr))

	if err := a.Services.Client.Login(conn); err != nil {
		a.Logger.Warn(fmt.Sprintf("failed to login: %v", err))
		return false
	}
	runID, err := a.Services.Client.LoginResponse(conn)
	if err != nil {
		a.Logger.Warn(fmt.Sprintf("failed to login: %v", err))
		return false
	}
	a.Logger.Info(fmt.Sprintf("login success: run_id=%s", runID))

	if err := a.Services.Client.ProxiesApply(conn); err != nil {
		a.Logger.Warn(fmt.Sprintf("failed to apply proxies: %v", err))
		return false
	}
	if err := a.Services.Client.ProxiesApplyResponse(conn); err != nil {
		a.Logger.Warn(fmt.Sprintf("failed to apply proxies: %v", err))
		return false
	}
	a.Logger.Info(fmt.Sprintf("proxies applied: count=%d", len(a.Services.Client.Proxies)))

	*nextRetryDelay = reconnectMin

	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	msgCh := make(chan controlMsg, 1)
	doneCh := make(chan struct{})
	defer close(doneCh)

	go a.controlLoop(conn, msgCh, doneCh)

	for {
		select {
		case <-a.stopCh:
			return true
		case <-heartbeatTicker.C:
			if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				a.Logger.Warn(fmt.Sprintf("failed to set heartbeat write deadline: %v", err))
				return false
			}
			err := sharedprotocol.WriteMsg(conn, sharedprotocol.MsgHeartbeat, sharedprotocol.HeartbeatMsg{})
			_ = conn.SetWriteDeadline(time.Time{})
			if err != nil {
				a.Logger.Warn(fmt.Sprintf("failed to send heartbeat: %v", err))
				return false
			}
		case result := <-msgCh:
			if result.err != nil {
				select {
				case <-a.stopCh:
					return true
				default:
				}
				a.Logger.Warn(fmt.Sprintf("failed to read control message: %v", result.err))
				return false
			}
			switch result.msgType {
			case sharedprotocol.MsgNewWorkConn:
				var msg sharedprotocol.NewWorkConnMsg
				if err := sharedprotocol.Decode(result.payload, &msg); err != nil {
					a.Logger.Error(fmt.Sprintf("failed to parse new work conn msg: %v", err))
					continue
				}
				go a.Services.Client.WorkConn(msg)
			case sharedprotocol.MsgHeartbeatResp:
			default:
				a.Logger.Warn(fmt.Sprintf("received unknown control message 0x%02x", result.msgType))
			}
		}
	}
}

func (a *Conn) controlLoop(conn net.Conn, msgCh chan controlMsg, doneCh chan struct{}) {
	for {
		if err := conn.SetReadDeadline(time.Now().Add(90 * time.Second)); err != nil {
			select {
			case msgCh <- controlMsg{err: fmt.Errorf("failed to set read deadline: %w", err)}:
			case <-a.stopCh:
			case <-doneCh:
			}
			return
		}
		msgType, payload, err := sharedprotocol.ReadMsg(conn)
		select {
		case msgCh <- controlMsg{msgType: msgType, payload: payload, err: err}:
		case <-a.stopCh:
			return
		case <-doneCh:
			return
		}
		if err != nil {
			return
		}
	}
}
