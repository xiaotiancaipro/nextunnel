package utils

import (
	"io"
	"net"
)

func LocalIP(ip string) string {
	if ip == "" {
		return "127.0.0.1"
	}
	return ip
}

func Pipe(a, b net.Conn) {
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
