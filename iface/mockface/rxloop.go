package mockface

/*
#include "../face.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

type rxLoop struct{}

var TheRxLoop iface.IRxLooper = rxLoop{}

type rxPacket struct {
	face *MockFace
	pkt  dpdk.Packet
}

var rxQueue chan rxPacket = make(chan rxPacket)
var rxStop chan struct{} = make(chan struct{})

func (rxLoop) RxLoop(burstSize int, cb unsafe.Pointer, cbarg unsafe.Pointer) {
	burst := iface.NewRxBurst(1)
	defer burst.Close()
	for {
		select {
		case rxp := <-rxQueue:
			burst.SetFrame(0, rxp.pkt)
			C.FaceImpl_RxBurst(rxp.face.getPtr(), 0, (*C.FaceRxBurst)(burst.GetPtr()), 1,
				(C.Face_RxCb)(cb), cbarg)
		case <-rxStop:
			return
		}
	}
}

func (rxLoop) StopRxLoop() error {
	rxStop <- struct{}{}
	return nil
}

func (rxLoop) ListFacesInRxLoop() (faceIds []iface.FaceId) {
	faceIds = make([]iface.FaceId, 0)
	for it := iface.IterFaces(); it.Valid(); it.Next() {
		if it.Id.GetKind() == iface.FaceKind_Mock {
			faceIds = append(faceIds, it.Id)
		}
	}
	return faceIds
}
