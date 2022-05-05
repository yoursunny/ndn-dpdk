//go:build linux

package memiftransport

import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/zyedidia/generic"
	"go.uber.org/multierr"
)

// Bridge bridges two memif interfaces.
// The memifs can operate in either server or client mode.
//
// This is mainly useful for unit testing.
// It is impossible to run both memif peers in the same process, so the test program should run this bridge in a separate process.
type Bridge struct {
	hdlA    *handle
	hdlB    *handle
	closing chan struct{}
}

// NewBridge creates a Bridge.
func NewBridge(locA, locB Locator) (bridge *Bridge, e error) {
	if e = locA.Validate(); e != nil {
		return nil, fmt.Errorf("LocatorA %w", e)
	}
	locA.ApplyDefaults(RoleServer)
	if e = locB.Validate(); e != nil {
		return nil, fmt.Errorf("LocatorB %w", e)
	}
	locB.ApplyDefaults(RoleServer)

	bridge = &Bridge{
		closing: make(chan struct{}),
	}
	bridge.hdlA, e = newHandle(locA, func(l3.TransportState) {})
	if e != nil {
		return nil, fmt.Errorf("newHandleA %w", e)
	}
	bridge.hdlB, e = newHandle(locB, func(l3.TransportState) {})
	if e != nil {
		bridge.hdlA.Close()
		return nil, fmt.Errorf("newHandleB %w", e)
	}

	go bridge.transferLoop(bridge.hdlA, bridge.hdlB)
	go bridge.transferLoop(bridge.hdlB, bridge.hdlA)
	return bridge, nil
}

func (bridge *Bridge) transferLoop(src, dst *handle) {
	buf := make([]byte, generic.Max(src.loc.Dataroom, dst.loc.Dataroom))
	for {
		select {
		case <-bridge.closing:
			return
		default:
		}

		n, e := src.Read(buf)
		if e == nil && n > 0 {
			dst.Write(buf[:n])
		}
	}
}

// Close stops the bridge.
func (bridge *Bridge) Close() error {
	close(bridge.closing)
	return multierr.Append(bridge.hdlA.Close(), bridge.hdlB.Close())
}
