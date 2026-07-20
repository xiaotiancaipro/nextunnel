package services

type Services struct {
	AccessLog   *AccessLog
	AccessRule  *AccessRule
	Client      *Client
	ClientCert  *ClientCert
	ClientProxy *ClientProxy
	Server      *Server
	Tls         *Tls
}
