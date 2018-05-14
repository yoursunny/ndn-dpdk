package fib

/*
#include "fib.h"
*/
import "C"
import (
	"ndn-dpdk/core/urcu"
	"ndn-dpdk/ndn"
)

// List all FIB entry names.
func (fib *Fib) ListNames() (names []*ndn.Name) {
	fib.postCommand(func(rs *urcu.ReadSide) error {
		names = make([]*ndn.Name, 0)
		fib.treeRoot.Walk("", func(name string, isEntry bool) {
			if isEntry {
				n, _ := ndn.NewName(ndn.TlvBytes(name))
				names = append(names, n)
			}
		})
		return nil
	})
	return names
}

func findC(fibC *C.Fib, nameV ndn.TlvBytes) (entryC *C.FibEntry) {
	return C.__Fib_Find(fibC, C.uint16_t(len(nameV)), (*C.uint8_t)(nameV.GetPtr()))
}

// Perform an exact match lookup.
func (fib *Fib) Find(name *ndn.Name) (entry *Entry) {
	_, partition := fib.ndt.Lookup(name)
	return fib.FindInPartition(name, int(partition))
}

// Perform an exact match lookup in specified partition.
func (fib *Fib) FindInPartition(name *ndn.Name, partition int) (entry *Entry) {
	fib.postCommand(func(rs *urcu.ReadSide) error {
		rs.Lock()
		defer rs.Unlock()
		entryC := findC(fib.c[partition], name.GetValue())
		if entryC != nil {
			entry = &Entry{*entryC}
		}
		return nil
	})
	return entry
}

// Perform a longest prefix match lookup.
func (fib *Fib) Lpm(name *ndn.Name) (entry *Entry) {
	_, partition := fib.ndt.Lookup(name)
	return fib.LpmInPartition(name, int(partition))
}

// Perform a longest prefix match lookup in specified partition.
func (fib *Fib) LpmInPartition(name *ndn.Name, partition int) (entry *Entry) {
	fib.postCommand(func(rs *urcu.ReadSide) error {
		rs.Lock()
		defer rs.Unlock()
		entryC := C.__Fib_Lpm(fib.c[partition], (*C.PName)(name.GetPNamePtr()),
			(*C.uint8_t)(name.GetValue().GetPtr()))
		if entryC != nil {
			entry = &Entry{*entryC}
		}
		return nil
	})
	return entry
}
