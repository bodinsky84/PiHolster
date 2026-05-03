package dns

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	mdns "github.com/miekg/dns"
)

// Server is a DNS server that blocks listed domains and forwards the rest via DoH.
type Server struct {
	blocklist *Blocklist
	upstream  *DoHUpstream
	udpServer *mdns.Server
	tcpServer *mdns.Server
	port      string
	udpReady  chan struct{}
	tcpReady  chan struct{}
}

// NewServer constructs a Server. It reads DNS_PORT from the environment,
// defaulting to 5300 for unprivileged dev use.
func NewServer(bl *Blocklist, upstream *DoHUpstream) *Server {
	port := os.Getenv("DNS_PORT")
	if port == "" {
		port = "5300"
	}
	s := &Server{
		blocklist: bl,
		upstream:  upstream,
		port:      port,
	}

	mux := mdns.NewServeMux()
	mux.HandleFunc(".", s.handle)

	addr := fmt.Sprintf(":%s", port)
	udpReady := make(chan struct{})
	tcpReady := make(chan struct{})
	s.udpServer = &mdns.Server{
		Addr:              addr,
		Net:               "udp",
		Handler:           mux,
		NotifyStartedFunc: func() { close(udpReady) },
	}
	s.tcpServer = &mdns.Server{
		Addr:              addr,
		Net:               "tcp",
		Handler:           mux,
		NotifyStartedFunc: func() { close(tcpReady) },
	}
	s.udpReady = udpReady
	s.tcpReady = tcpReady
	return s
}

// Start launches UDP and TCP listeners and blocks until both have bound their
// port. NotifyStartedFunc closes the ready channels once the OS has assigned
// the socket, so the caller can trust the port is live when Start returns.
func (s *Server) Start() error {
	go func() {
		slog.Info("DNS UDP listener starting", "port", s.port)
		if err := s.udpServer.ListenAndServe(); err != nil {
			slog.Error("DNS UDP listener exited", "err", err)
		}
	}()

	go func() {
		slog.Info("DNS TCP listener starting", "port", s.port)
		if err := s.tcpServer.ListenAndServe(); err != nil {
			slog.Error("DNS TCP listener exited", "err", err)
		}
	}()

	select {
	case <-s.udpReady:
	case <-time.After(5 * time.Second):
		return fmt.Errorf("dns: udp server failed to start within 5s")
	}
	select {
	case <-s.tcpReady:
	case <-time.After(5 * time.Second):
		return fmt.Errorf("dns: tcp server failed to start within 5s")
	}
	return nil
}

// Shutdown gracefully stops both listeners using the provided context.
func (s *Server) Shutdown(ctx context.Context) error {
	var udpErr, tcpErr error
	if err := s.udpServer.ShutdownContext(ctx); err != nil {
		udpErr = err
	}
	if err := s.tcpServer.ShutdownContext(ctx); err != nil {
		tcpErr = err
	}
	return errors.Join(udpErr, tcpErr)
}

// handle processes a single DNS query.
// Pipeline: strip trailing FQDN dot → check blocklist → NXDOMAIN or upstream.
func (s *Server) handle(w mdns.ResponseWriter, r *mdns.Msg) {
	if len(r.Question) == 0 {
		mdns.HandleFailed(w, r)
		return
	}

	q := r.Question[0]
	domain := strings.ToLower(strings.TrimSuffix(q.Name, "."))

	if s.blocklist.IsBlocked(domain) {
		slog.Debug("blocked", "domain", domain)
		m := new(mdns.Msg)
		m.SetReply(r)
		m.Rcode = mdns.RcodeNameError // NXDOMAIN
		if err := w.WriteMsg(m); err != nil {
			slog.Error("write NXDOMAIN response", "err", err)
		}
		return
	}

	reply, err := s.upstream.Resolve(r)
	if err != nil {
		slog.Error("upstream resolve error", "domain", domain, "err", err)
		mdns.HandleFailed(w, r)
		return
	}

	if err := w.WriteMsg(reply); err != nil {
		slog.Error("write upstream response", "domain", domain, "err", err)
	}
}
