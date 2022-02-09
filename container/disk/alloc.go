package disk

/*
#include "../../csrc/disk/alloc.h"
*/
import "C"
import (
	"errors"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// Alloc is a disk slot allocator.
type Alloc C.DiskAlloc

func (a *Alloc) ptr() *C.DiskAlloc {
	return (*C.DiskAlloc)(a)
}

// Ptr returns *C.DiskAlloc pointer.
func (a *Alloc) Ptr() unsafe.Pointer {
	return unsafe.Pointer(a)
}

// Close destroys the Alloc.
func (a *Alloc) Close() error {
	eal.Free(a)
	return nil
}

// SlotRange returns a range of possible slot numbers.
func (a *Alloc) SlotRange() (min, max uint64) {
	return uint64(a.min), uint64(a.min + a.total - 1)
}

// Capacity returns total number of slots.
func (a *Alloc) Capacity() int {
	return int(a.total)
}

// CountAvailable returns number of available slots.
func (a *Alloc) CountAvailable() int {
	return int(a.count)
}

// Alloc allocates a disk slot.
func (a *Alloc) Alloc() (slot uint64, e error) {
	s := C.DiskAlloc_Alloc(a.ptr())
	if s == 0 {
		return 0, errors.New("disk slot unavailable")
	}
	return uint64(s), nil
}

// Free frees a disk slot.
func (a *Alloc) Free(slot uint64) {
	C.DiskAlloc_Free(a.ptr(), C.uint64_t(slot))
}

// NewAlloc creates an Alloc.
func NewAlloc(min, max uint64, socket eal.NumaSocket) *Alloc {
	a := C.DiskAlloc_New(C.uint64_t(min), C.uint64_t(max), C.int(socket.ID()))
	return (*Alloc)(a)
}

// SizeCalc calculates Store and Alloc sizes.
type SizeCalc struct {
	// NThreads is number of threads.
	NThreads int
	// NPackets is number of packets (capacity) per thread.
	NPackets int
	// PacketSize is size of each packet.
	PacketSize int
}

// BlocksPerSlot returns number of blocks per packet slot.
func (calc SizeCalc) BlocksPerSlot() int {
	return (calc.PacketSize + BlockSize - 1) / BlockSize
}

// StoreBlocks calculates minimum number of blocks required in the Store.
func (calc SizeCalc) MinBlocks() int64 {
	return int64(calc.BlocksPerSlot()) * int64(1+calc.NThreads*calc.NPackets)
}

// CreateAlloc creates an Alloc for i-th thread.
func (calc SizeCalc) CreateAlloc(i int, socket eal.NumaSocket) *Alloc {
	min := 1 + uint64(i)*uint64(calc.NPackets)
	max := min + uint64(calc.NPackets) - 1
	return NewAlloc(min, max, socket)
}