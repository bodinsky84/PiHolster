package arp

import (
	"net"
	"testing"

	mdarp "github.com/mdlayher/arp"
)

// TestParseARPReply verifies that parseDevice correctly extracts MAC and IP from
// a synthetic ARP reply packet without touching any real network interface.
func TestParseARPReply(t *testing.T) {
	mac, err := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	if err != nil {
		t.Fatal(err)
	}
	ip := net.ParseIP("192.168.1.42").To4()

	pkt := &mdarp.Packet{
		Operation:      mdarp.OperationReply,
		SenderHardwareAddr: mac,
		SenderIP:       ip,
		TargetHardwareAddr: make(net.HardwareAddr, 6),
		TargetIP:       net.ParseIP("192.168.1.1").To4(),
	}

	dev := parseDevice(pkt)

	if dev.MAC != mac.String() {
		t.Errorf("MAC: got %q, want %q", dev.MAC, mac.String())
	}
	if dev.IP != ip.String() {
		t.Errorf("IP: got %q, want %q", dev.IP, ip.String())
	}
	// Hostname is best-effort; we only check it doesn't contain the IP itself.
	if dev.Hostname == ip.String() {
		t.Errorf("Hostname should not equal IP, got %q", dev.Hostname)
	}
}

// TestUpsertPublishesNewDevice verifies that the first observation of a MAC
// results in an event on the devices channel.
func TestUpsertPublishesNewDevice(t *testing.T) {
	s := &Scanner{
		known:   make(map[string]*knownDevice),
		devices: make(chan Device, 1),
	}

	dev := Device{MAC: "11:22:33:44:55:66", IP: "10.0.0.5", Hostname: "toaster"}
	s.upsert(dev)

	select {
	case got := <-s.devices:
		if got.MAC != dev.MAC {
			t.Errorf("MAC: got %q, want %q", got.MAC, dev.MAC)
		}
		if got.IP != dev.IP {
			t.Errorf("IP: got %q, want %q", got.IP, dev.IP)
		}
	default:
		t.Fatal("expected device on channel, got nothing")
	}
}

// TestUpsertDeduplicates verifies that a second upsert with identical fields
// does NOT produce a second event.
func TestUpsertDeduplicates(t *testing.T) {
	s := &Scanner{
		known:   make(map[string]*knownDevice),
		devices: make(chan Device, 8),
	}

	dev := Device{MAC: "aa:aa:aa:aa:aa:aa", IP: "172.16.0.1", Hostname: "router"}
	s.upsert(dev)
	s.upsert(dev) // identical — should not emit again

	count := len(s.devices)
	if count != 1 {
		t.Errorf("expected 1 event, got %d", count)
	}
}

// TestUpsertPublishesOnIPChange verifies that a MAC seen on a new IP causes a
// second event (device moved to new address).
func TestUpsertPublishesOnIPChange(t *testing.T) {
	s := &Scanner{
		known:   make(map[string]*knownDevice),
		devices: make(chan Device, 8),
	}

	mac := "bb:bb:bb:bb:bb:bb"
	s.upsert(Device{MAC: mac, IP: "10.0.0.10"})
	s.upsert(Device{MAC: mac, IP: "10.0.0.11"}) // IP changed

	count := len(s.devices)
	if count != 2 {
		t.Errorf("expected 2 events for IP change, got %d", count)
	}
}

// TestInterfaceCIDRNil verifies that interfaceCIDR returns nil for an interface
// with no IPv4 addresses (simulated by passing a dummy interface).
func TestInterfaceCIDRNil(t *testing.T) {
	dummy := &net.Interface{
		Index: 999,
		Name:  "dummy0",
		Flags: net.FlagUp,
	}
	// dummy has no addresses assigned; interfaceCIDR must return nil, not panic.
	cidr := interfaceCIDR(dummy)
	if cidr != nil {
		t.Errorf("expected nil CIDR for address-less interface, got %v", cidr)
	}
}
