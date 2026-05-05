package arp

import (
	"context"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"strconv"
	"sync"
	"time"

	mdarp "github.com/mdlayher/arp"
)

// Device is a network host observed via ARP.
type Device struct {
	MAC      string
	IP       string
	Hostname string
}

// Scanner sends ARP probes and listens for replies on a single interface.
type Scanner struct {
	iface    *net.Interface
	client   *mdarp.Client
	interval time.Duration

	mu      sync.RWMutex
	known   map[string]*knownDevice // keyed by MAC string
	devices chan Device
}

type knownDevice struct {
	Device
	seenAt time.Time
}

// NewScanner creates a Scanner bound to the given interface name.
// If ifaceName is empty the default-route interface is auto-detected.
func NewScanner(ifaceName string) (*Scanner, error) {
	iface, err := resolveInterface(ifaceName)
	if err != nil {
		return nil, err
	}

	client, err := mdarp.Dial(iface)
	if err != nil {
		return nil, err
	}

	interval := 30 * time.Second
	if s := os.Getenv("ARP_INTERVAL_SECONDS"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			interval = time.Duration(n) * time.Second
		}
	}

	return &Scanner{
		iface:    iface,
		client:   client,
		interval: interval,
		known:    make(map[string]*knownDevice),
		devices:  make(chan Device, 64),
	}, nil
}

// Devices returns the channel on which new and updated devices are published.
func (s *Scanner) Devices() <-chan Device {
	return s.devices
}

// Run starts the scan loop and the ARP reply listener. It blocks until ctx is
// cancelled. Callers should run this in a goroutine.
func (s *Scanner) Run(ctx context.Context) error {
	defer s.client.Close()

	go s.listenReplies(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Probe immediately on start, then on every tick.
	s.probeSweep(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s.probeSweep(ctx)
		}
	}
}

// probeSweep sends ARP requests for every host in the /24 that contains the
// interface's primary IPv4 address.
func (s *Scanner) probeSweep(ctx context.Context) {
	cidr := interfaceCIDR(s.iface)
	if cidr == nil {
		slog.Warn("arp: no IPv4 address on interface, skipping sweep", "iface", s.iface.Name)
		return
	}

	base := cidr.IP.Mask(cidr.Mask)
	for i := 1; i < 255; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}
		targetIP := net.IP{base[0], base[1], base[2], byte(i)}
		targetAddr, ok := netip.AddrFromSlice(targetIP)
		if !ok {
			continue
		}
		if err := s.client.Request(targetAddr.Unmap()); err != nil {
			slog.Debug("arp: request failed", "target", targetIP, "err", err)
		}
	}
}

// listenReplies reads ARP packets from the wire and publishes new/updated
// devices.
func (s *Scanner) listenReplies(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		pkt, _, err := s.client.Read()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Debug("arp: read error", "err", err)
			continue
		}

		if pkt.Operation != mdarp.OperationReply {
			continue
		}

		dev := parseDevice(pkt)
		s.upsert(dev)
	}
}

// parseDevice converts an ARP packet into a Device, resolving hostname
// best-effort with a 500 ms deadline.
func parseDevice(pkt *mdarp.Packet) Device {
	mac := pkt.SenderHardwareAddr.String()
	ip := pkt.SenderIP.String()
	hostname := reverseLookup(ip)
	return Device{MAC: mac, IP: ip, Hostname: hostname}
}

func reverseLookup(ip string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	names, err := net.DefaultResolver.LookupAddr(ctx, ip)
	if err != nil || len(names) == 0 {
		return ""
	}
	name := names[0]
	if len(name) > 0 && name[len(name)-1] == '.' {
		name = name[:len(name)-1]
	}
	return name
}

func (s *Scanner) upsert(dev Device) {
	s.mu.Lock()
	existing, ok := s.known[dev.MAC]
	if !ok || existing.IP != dev.IP || existing.Hostname != dev.Hostname {
		s.known[dev.MAC] = &knownDevice{Device: dev, seenAt: time.Now()}
		s.mu.Unlock()
		select {
		case s.devices <- dev:
		default:
			slog.Warn("arp: device channel full, dropping event", "mac", dev.MAC)
		}
		return
	}
	existing.seenAt = time.Now()
	s.mu.Unlock()
}

// KnownDevices returns a snapshot of all currently known devices.
func (s *Scanner) KnownDevices() []Device {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Device, 0, len(s.known))
	for _, kd := range s.known {
		out = append(out, kd.Device)
	}
	return out
}

// resolveInterface finds the named interface, or auto-detects the one carrying
// the default route when name is empty.
func resolveInterface(name string) (*net.Interface, error) {
	if name != "" {
		return net.InterfaceByName(name)
	}
	return defaultRouteInterface()
}

// defaultRouteInterface returns the first non-loopback, up interface with a
// global unicast IPv4 address.
func defaultRouteInterface() (*net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}
			if ip.To4() != nil && ip.IsGlobalUnicast() {
				return &iface, nil
			}
		}
	}
	return nil, &net.AddrError{Err: "no suitable interface found", Addr: "default"}
}

// interfaceCIDR returns the first IPv4 CIDR assigned to iface, or nil.
func interfaceCIDR(iface *net.Interface) *net.IPNet {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil
	}
	for _, addr := range addrs {
		ip, cidr, err := net.ParseCIDR(addr.String())
		if err != nil {
			continue
		}
		if ip.To4() != nil {
			cidr.IP = cidr.IP.To4()
			return cidr
		}
	}
	return nil
}
