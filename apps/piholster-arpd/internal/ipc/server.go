package ipc

import (
	"bufio"
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"github.com/piholster/piholster/apps/piholster-arpd/internal/arp"
)

// wireDevice is the on-wire JSON representation for each device event.
type wireDevice struct {
	MAC      string    `json:"mac"`
	IP       string    `json:"ip"`
	Hostname string    `json:"hostname"`
	SeenAt   time.Time `json:"seen_at"`
}

// Server listens on a Unix socket and broadcasts ARP events to every connected
// piholsterd client.
type Server struct {
	sockPath string
	events   <-chan arp.Device

	mu      sync.RWMutex
	known   []wireDevice        // snapshot for new connections
	clients map[net.Conn]*bufio.Writer
}

// NewServer creates a Server and binds the Unix socket at sockPath. The socket
// is created with mode 0660 so that both the arpd user and the piholsterd group
// can connect.
func NewServer(sockPath string, events <-chan arp.Device) (*Server, error) {
	// Remove a stale socket from a previous run.
	if err := os.Remove(sockPath); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return &Server{
		sockPath: sockPath,
		events:   events,
		clients:  make(map[net.Conn]*bufio.Writer),
	}, nil
}

// Serve accepts connections and fans events out to all connected clients.
// It returns when ctx is cancelled.
func (s *Server) Serve(ctx context.Context) error {
	ln, err := net.Listen("unix", s.sockPath)
	if err != nil {
		return err
	}
	// 0660: arpd user + piholster group.
	if err := os.Chmod(s.sockPath, 0660); err != nil {
		ln.Close()
		return err
	}
	defer ln.Close()

	slog.Info("ipc: listening", "path", s.sockPath)

	// Fan-out goroutine: reads from the scanner channel and broadcasts.
	go s.fanOut(ctx)

	// Accept goroutine closer: unblocks ln.Accept when ctx fires.
	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			slog.Error("ipc: accept error", "err", err)
			continue
		}
		go s.handleConn(ctx, conn)
	}
}

// handleConn sends the current snapshot to the newly connected client, then
// keeps the connection open for further events. The connection is tracked in
// the clients map so fanOut can reach it.
func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	bw := bufio.NewWriter(conn)

	s.mu.Lock()
	snapshot := make([]wireDevice, len(s.known))
	copy(snapshot, s.known)
	s.clients[conn] = bw
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, conn)
		s.mu.Unlock()
	}()

	// Send existing devices so the client is caught up immediately.
	for _, d := range snapshot {
		if err := writeDevice(bw, d); err != nil {
			slog.Debug("ipc: snapshot write error", "err", err)
			return
		}
	}
	if err := bw.Flush(); err != nil {
		slog.Debug("ipc: flush error after snapshot", "err", err)
		return
	}

	// Block until the client disconnects or ctx is done.
	<-ctx.Done()
}

// fanOut distributes incoming device events to all connected clients and
// updates the known-devices snapshot.
func (s *Server) fanOut(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case dev, ok := <-s.events:
			if !ok {
				return
			}
			wd := wireDevice{
				MAC:      dev.MAC,
				IP:       dev.IP,
				Hostname: dev.Hostname,
				SeenAt:   time.Now().UTC(),
			}

			s.mu.Lock()
			s.upsertKnown(wd)
			writers := make([]*bufio.Writer, 0, len(s.clients))
			var conns []net.Conn
			for conn, bw := range s.clients {
				writers = append(writers, bw)
				conns = append(conns, conn)
			}
			s.mu.Unlock()

			for i, bw := range writers {
				if err := writeDevice(bw, wd); err != nil {
					slog.Debug("ipc: write error, removing client", "err", err)
					conns[i].Close()
					continue
				}
				if err := bw.Flush(); err != nil {
					slog.Debug("ipc: flush error", "err", err)
					conns[i].Close()
				}
			}
		}
	}
}

// upsertKnown updates the in-memory snapshot. Must be called with s.mu held.
func (s *Server) upsertKnown(wd wireDevice) {
	for i, existing := range s.known {
		if existing.MAC == wd.MAC {
			s.known[i] = wd
			return
		}
	}
	s.known = append(s.known, wd)
}

// writeDevice serialises wd as a JSON line into bw.
func writeDevice(bw *bufio.Writer, wd wireDevice) error {
	data, err := json.Marshal(wd)
	if err != nil {
		return err
	}
	if _, err := bw.Write(data); err != nil {
		return err
	}
	return bw.WriteByte('\n')
}
