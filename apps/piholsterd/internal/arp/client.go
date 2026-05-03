package arp

import (
	"bufio"
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"sync"
	"time"
)

// Device is an ARP-observed host received from piholster-arpd.
type Device struct {
	MAC      string
	IP       string
	Hostname string
	SeenAt   time.Time
}

// wireDevice mirrors the JSON format produced by piholster-arpd's IPC server.
type wireDevice struct {
	MAC      string    `json:"mac"`
	IP       string    `json:"ip"`
	Hostname string    `json:"hostname"`
	SeenAt   time.Time `json:"seen_at"`
}

// Client connects to the piholster-arpd Unix socket and streams Device events.
// It reconnects automatically with exponential backoff when the socket is
// unavailable.
type Client struct {
	sockPath string

	mu   sync.Mutex
	conn net.Conn

	out chan Device
}

// NewClient creates a Client that will connect to sockPath. Call Connect to
// start streaming.
func NewClient(sockPath string) *Client {
	return &Client{
		sockPath: sockPath,
		out:      make(chan Device, 64),
	}
}

// Connect starts the background read loop. It returns immediately; device
// events are delivered on the channel returned by Devices(). Connect blocks
// until the first successful dial or ctx is cancelled.
func (c *Client) Connect(ctx context.Context) error {
	conn, err := c.dialWithBackoff(ctx)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	go c.readLoop(ctx, conn)
	return nil
}

// Devices returns the channel on which Device events are published.
// The channel is never closed; callers should stop reading when ctx is done.
func (c *Client) Devices() <-chan Device {
	return c.out
}

// Close shuts down the underlying connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// readLoop reads newline-delimited JSON from conn. When the connection drops it
// dials again with exponential backoff.
func (c *Client) readLoop(ctx context.Context, initial net.Conn) {
	conn := initial
	defer conn.Close()

	for {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			var wd wireDevice
			if err := json.Unmarshal(scanner.Bytes(), &wd); err != nil {
				slog.Debug("arp client: decode error", "err", err)
				continue
			}
			dev := Device{
				MAC:      wd.MAC,
				IP:       wd.IP,
				Hostname: wd.Hostname,
				SeenAt:   wd.SeenAt,
			}
			select {
			case c.out <- dev:
			case <-ctx.Done():
				return
			default:
				slog.Warn("arp client: output channel full, dropping device", "mac", dev.MAC)
			}
		}

		if ctx.Err() != nil {
			return
		}
		slog.Info("arp client: connection lost, reconnecting", "sock", c.sockPath)
		conn.Close()

		newConn, err := c.dialWithBackoff(ctx)
		if err != nil {
			// ctx was cancelled during backoff.
			return
		}
		conn = newConn

		c.mu.Lock()
		c.conn = conn
		c.mu.Unlock()
	}
}

// dialWithBackoff attempts to dial the Unix socket with exponential backoff
// starting at 500ms, capped at 30s. It returns the first successful connection
// or an error if ctx is cancelled.
func (c *Client) dialWithBackoff(ctx context.Context) (net.Conn, error) {
	backoff := 500 * time.Millisecond
	const maxBackoff = 30 * time.Second

	for {
		conn, err := (&net.Dialer{}).DialContext(ctx, "unix", c.sockPath)
		if err == nil {
			return conn, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		slog.Debug("arp client: dial failed, retrying", "backoff", backoff, "err", err)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}
