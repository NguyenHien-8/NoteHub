package share

import (
	"context"
	"net"
	"net/http"
	"time"
)

type Server struct {
	http *http.Server
	ln   net.Listener
}

// Start binds only to the address the caller chooses. A desktop app should use
// 127.0.0.1:0 by default and require explicit user action before binding to a
// LAN interface such as 0.0.0.0:8787.
func Start(address string, handler http.Handler) (*Server, error) {
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	s := &Server{ln: ln, http: &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}}
	go func() { _ = s.http.Serve(ln) }()
	return s, nil
}

func (s *Server) Addr() net.Addr                     { return s.ln.Addr() }
func (s *Server) Shutdown(ctx context.Context) error { return s.http.Shutdown(ctx) }
