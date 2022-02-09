// Package disk provides a disk-based Data packet store.
package disk

/*
#include "../../csrc/disk/store.h"

extern int go_getDataCallback(Packet* npkt, uintptr_t ctx);
*/
import "C"
import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/bdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/mempool"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

var logger = logging.New("disk")

// BlockSize is the supported bdev block size.
const BlockSize = C.DiskStore_BlockSize

var (
	// StoreGetDataCallback is a C function type for store.GetData callback.
	StoreGetDataCallback = cptr.FunctionType{"Packet"}

	// StoreGetDataGo is a StoreGetDataCallback implementation for receiving the Data in Go code.
	StoreGetDataGo = StoreGetDataCallback.C(unsafe.Pointer(C.go_getDataCallback), uintptr(0))

	getDataReplyMap sync.Map
)

//export go_getDataCallback
func go_getDataCallback(npkt *C.Packet, ctx C.uintptr_t) C.int {
	reply, ok := getDataReplyMap.LoadAndDelete(npkt)
	if !ok {
		panic("unexpected invocation")
	}
	close(reply.(chan struct{}))
	return 0
}

// Store represents a disk-backed Data Store.
type Store struct {
	c  *C.DiskStore
	mp *mempool.Mempool
	bd *bdev.Bdev
	th *spdkenv.Thread

	getDataCbRevoke func()
	getDataGo       bool
}

// Ptr returns *C.DiskStore pointer.
func (store *Store) Ptr() unsafe.Pointer {
	return unsafe.Pointer(store.c)
}

// Close closes this Store.
func (store *Store) Close() error {
	if ch := store.c.ch; ch != nil {
		// this would panic if SPDK thread is closed
		store.th.Post(cptr.Func0.Void(func() { C.spdk_put_io_channel(ch) }))
	}
	eal.Free(store.c)
	store.c = nil
	store.getDataCbRevoke()
	return multierr.Append(
		store.mp.Close(),
		store.bd.Close(),
	)
}

// SlotRange returns a range of possible slot numbers.
func (store *Store) SlotRange() (min, max uint64) {
	return 1, uint64(store.bd.DevInfo().CountBlocks()/int64(store.c.nBlocksPerSlot) - 1)
}

// PutData asynchronously stores a Data packet.
func (store *Store) PutData(slotID uint64, data *ndni.Packet) {
	C.DiskStore_PutData(store.c, C.uint64_t(slotID), (*C.Packet)(data.Ptr()))
}

// GetData retrieves a Data packet from specified slot and waits for completion.
// This can be used only if the Store was created with StoreGetDataGo.
func (store *Store) GetData(slotID uint64, interest *ndni.Packet, dataBuf *pktmbuf.Packet) (data *ndni.Packet) {
	if !store.getDataGo {
		logger.Panic("Store is not created with StoreGetDataGo, cannot GetData")
	}

	interestC := (*C.Packet)(interest.Ptr())
	pinterest := C.Packet_GetInterestHdr(interestC)

	reply := make(chan struct{})
	_, dup := getDataReplyMap.LoadOrStore(interestC, reply)
	if dup {
		logger.Panic("ongoing GetData on the same mbuf")
	}

	C.DiskStore_GetData(store.c, C.uint64_t(slotID), interestC, (*C.struct_rte_mbuf)(dataBuf.Ptr()))
	<-reply

	if retSlot := uint64(pinterest.diskSlot); retSlot != slotID {
		logger.Panic("unexpected PInterest.diskSlot",
			zap.Uint64("request-slot", slotID),
			zap.Uint64("return-slot", slotID),
		)
	}
	return ndni.PacketFromPtr(unsafe.Pointer(pinterest.diskData))
}

// NewStore creates a Store.
func NewStore(device bdev.Device, th *spdkenv.Thread, nBlocksPerSlot int, getDataCb cptr.Function) (store *Store, e error) {
	bdi := device.DevInfo()
	if bdi.BlockSize() != BlockSize {
		return nil, fmt.Errorf("bdev block size must be %d", BlockSize)
	}

	store = &Store{
		th: th,
	}
	if store.mp, e = mempool.New(mempool.Config{
		Capacity:       int(math.MaxInt64(256, math.MinInt64(bdi.CountBlocks()/1024, 8192))),
		ElementSize:    C.sizeof_DiskStoreRequest,
		Socket:         th.LCore().NumaSocket(),
		SingleProducer: true,
	}); e != nil {
		return nil, e
	}
	if store.bd, e = bdev.Open(device, bdev.ReadWrite); e != nil {
		store.mp.Close()
		return nil, e
	}

	socket := th.LCore().NumaSocket()
	store.c = (*C.DiskStore)(eal.Zmalloc("DiskStore", C.sizeof_DiskStore, socket))
	store.bd.CopyToC(unsafe.Pointer(&store.c.bdev))
	store.c.mp = (*C.struct_rte_mempool)(store.mp.Ptr())
	store.c.th = (*C.struct_spdk_thread)(th.Ptr())
	store.c.nBlocksPerSlot = C.uint64_t(nBlocksPerSlot)

	f, ctx, revoke := StoreGetDataCallback.CallbackReuse(getDataCb)
	store.c.getDataCb, store.c.getDataCtx, store.getDataCbRevoke = C.DiskStore_GetDataCb(f), C.uintptr_t(ctx), revoke
	store.getDataGo = getDataCb == StoreGetDataGo

	// SPDK thread processes messages in order, so that this would occur before any other DiskStore usage
	th.Post(cptr.Func0.Void(func() {
		if store.c == nil {
			// in case store is closed without launching the SPDK thread
			return
		}

		store.c.ch = C.spdk_bdev_get_io_channel(store.c.bdev.desc)
		if store.c.ch == nil {
			logger.Panic("spdk_bdev_get_io_channel failed")
		}

		logger.Info("DiskStore ready",
			zap.Uintptr("store", uintptr(unsafe.Pointer(store.c))),
			zap.String("bdev", store.bd.DevInfo().Name()),
			zap.Uintptr("ch", uintptr(unsafe.Pointer(store.c.th))),
		)
	}))
	return store, nil
}