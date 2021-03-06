package ethface

/*
#include "../../csrc/ethface/locator.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev/ethvdev"
	"github.com/usnistgov/ndn-dpdk/iface"
)

// LocatorConflictError indicates that the locator of a new face conflicts with an existing face.
type LocatorConflictError struct {
	a, b ethLocator
}

func (e LocatorConflictError) Error() string {
	return fmt.Sprintf("locator %s conflicts with %s", iface.LocatorString(e.a), iface.LocatorString(e.b))
}

func (loc *cLocator) ptr() *C.EthLocator {
	return (*C.EthLocator)(unsafe.Pointer(loc))
}

func (loc cLocator) canCoexist(other cLocator) bool {
	return bool(C.EthLocator_CanCoexist(loc.ptr(), other.ptr()))
}

func (loc cLocator) sizeofHeader() int {
	var hdr C.EthTxHdr
	C.EthTxHdr_Prepare(&hdr, loc.ptr(), true)
	return int(hdr.len - hdr.l2len)
}

// FaceConfig contains additional face configuration.
// They appear as input-only fields of EtherLocator.
type FaceConfig struct {
	iface.Config

	// VDevConfig specifies additional configuration for virtual device creation.
	// This is only used when creating the first face on a network interface.
	VDevConfig *ethvdev.NetifConfig `json:"vdevConfig,omitempty"`

	// PortConfig specifies additional configuration for Port activation.
	// This is only used when creating the first face on an EthDev.
	PortConfig *PortConfig `json:"portConfig,omitempty"`

	// MaxRxQueues is the maximum number of RX queues for this face.
	// It is meaningful only if the face is using RxFlow dispatching.
	// It is effective in improving performance on VXLAN face only.
	//
	// Default is 1.
	// If this is greater than 1, NDNLPv2 reassembly will not work on this face.
	MaxRxQueues int `json:"maxRxQueues,omitempty"`

	// DisableTxMultiSegOffload forces every packet to be copied into a linear buffer in software.
	DisableTxMultiSegOffload bool `json:"disableTxMultiSegOffload,omitempty"`

	// DisableTxChecksumOffload disables the usage of IPv4 and UDP checksum offloads.
	DisableTxChecksumOffload bool `json:"disableTxChecksumOffload,omitempty"`

	// privFaceConfig is hidden from JSON output.
	privFaceConfig *FaceConfig
}

func (cfg FaceConfig) faceConfig() FaceConfig {
	if cfg.privFaceConfig != nil {
		return *cfg.privFaceConfig
	}
	return cfg
}

func (cfg FaceConfig) hideFaceConfigFromJSON() FaceConfig {
	return FaceConfig{privFaceConfig: &cfg}
}

type ethLocator interface {
	iface.Locator

	// cLoc converts to C.EthLocator.
	cLoc() cLocator

	faceConfig() FaceConfig
}

// LocatorCanCoexist determines whether two locators can coexist on the same port.
func LocatorCanCoexist(a, b iface.Locator) bool {
	return a.(ethLocator).cLoc().canCoexist(b.(ethLocator).cLoc())
}
