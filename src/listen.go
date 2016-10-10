package main

import (
	"log"
	"net"
	"net/http"
	"time"

	"github.com/armon/go-proxyproto"
)

func listenAndServeHTTP(h http.Handler) {
	srv := &http.Server{
		Handler: h,
		Addr:    "localhost:4576",
	}

	if err := serve(srv); err != nil {
		log.Fatal("[FATAL] ", err)
	}
}

func serve(srv *http.Server) error {
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		log.Fatal("[FATAL] ", err)
	}

	ln = &proxyproto.Listener{Listener: tcpKeepAliveListener{ln.(*net.TCPListener)}}

	return srv.Serve(ln)
}

// copied from http://golang.org/src/net/http/server.go?s=54604:54695#L1967
// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	if err = tc.SetKeepAlive(true); err != nil {
		return
	}
	if err = tc.SetKeepAlivePeriod(3 * time.Minute); err != nil {
		return
	}
	return tc, nil
}
