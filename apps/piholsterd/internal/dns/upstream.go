package dns

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	mdns "github.com/miekg/dns"
)

// maxDNSMessageSize is the maximum DNS message size per RFC 1035 §2.3.4.
const maxDNSMessageSize = 65535

const (
	dohPrimary   = "https://dns.quad9.net/dns-query"
	dohSecondary = "https://cloudflare-dns.com/dns-query"
	dohTimeout   = 3 * time.Second
	dohMIMEType  = "application/dns-message"
)

// DoHUpstream resolves DNS queries via DNS-over-HTTPS (RFC 8484).
// It tries the primary endpoint first and falls back to secondary on failure.
// If both fail, it returns SERVFAIL — it never downgrades to plaintext DNS.
type DoHUpstream struct {
	primary   string
	secondary string
	client    *http.Client
}

// NewDoHUpstream returns a DoHUpstream using Quad9 as primary and Cloudflare as secondary.
func NewDoHUpstream() *DoHUpstream {
	return &DoHUpstream{
		primary:   dohPrimary,
		secondary: dohSecondary,
		client: &http.Client{
			Timeout: dohTimeout,
		},
	}
}

// Resolve sends req to the upstream DoH server and returns the response.
// If the primary endpoint times out or errors, it retries on the secondary.
// If both fail, it returns a SERVFAIL response rather than falling back to
// plaintext port-53, keeping traffic encrypted end-to-end.
func (u *DoHUpstream) Resolve(req *mdns.Msg) (*mdns.Msg, error) {
	resp, err := u.queryEndpoint(u.primary, req)
	if err != nil {
		slog.Warn("DoH primary failed, trying secondary", "primary", u.primary, "err", err)
		resp, err = u.queryEndpoint(u.secondary, req)
		if err != nil {
			slog.Error("DoH secondary also failed, returning SERVFAIL", "secondary", u.secondary, "err", err)
			return servfail(req), nil
		}
	}
	return resp, nil
}

// queryEndpoint encodes req as a DNS wire-format message, POSTs it to endpoint
// per RFC 8484, and decodes the response.
func (u *DoHUpstream) queryEndpoint(endpoint string, req *mdns.Msg) (*mdns.Msg, error) {
	wire, err := req.Pack()
	if err != nil {
		return nil, fmt.Errorf("pack dns message: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(wire))
	if err != nil {
		return nil, fmt.Errorf("build http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", dohMIMEType)
	httpReq.Header.Set("Accept", dohMIMEType)

	httpResp, err := u.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request to %s: %w", endpoint, err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream %s returned HTTP %d", endpoint, httpResp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(httpResp.Body, maxDNSMessageSize))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	reply := new(mdns.Msg)
	if err := reply.Unpack(body); err != nil {
		hint := ""
		if len(body) >= maxDNSMessageSize {
			hint = " (response hit size limit — message may be truncated)"
		}
		return nil, fmt.Errorf("unpack dns response from %s: %w%s", endpoint, err, hint)
	}
	return reply, nil
}

// servfail constructs a SERVFAIL response for req.
func servfail(req *mdns.Msg) *mdns.Msg {
	m := new(mdns.Msg)
	m.SetReply(req)
	m.Rcode = mdns.RcodeServerFailure
	return m
}
