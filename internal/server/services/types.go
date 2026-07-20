package services

type Services struct {
	AccessFilter *AccessFilter
	AccessLog    *AccessLog
	AccessRule   *AccessRule
	Client       *Client
	ClientCert   *ClientCert
	ClientProxy  *ClientProxy
	Listener     *Listener
	ProxyBroker  *ProxyBroker
	Session      *Session
	Tls          *Tls
}
