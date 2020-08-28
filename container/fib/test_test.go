package fib_test

import (
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/container/fib/fibtestenv"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
)

func TestMain(m *testing.M) {
	ealtestenv.Init()
	os.Exit(m.Run())
}

var (
	makeAR    = testenv.MakeAR
	nameEqual = ndntestenv.NameEqual
	makeEntry = fibtestenv.MakeEntry
)