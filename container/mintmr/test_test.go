package mintmrtest

import (
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal/ealtestenv"
)

func TestMain(m *testing.M) {
	ealtestenv.Init()

	os.Exit(m.Run())
}

func TestMinTmr(t *testing.T) {
	testMinTmr(t)
}