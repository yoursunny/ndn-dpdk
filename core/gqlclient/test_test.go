package gqlclient_test

import (
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/gqlclient"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
)

var (
	makeAR = testenv.MakeAR

	serverConfig gqlclient.Config
)

func TestMain(m *testing.M) {
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		panic(e)
	}

	gqlserver.Prepare()
	go http.Serve(listener, nil)
	time.Sleep(100 * time.Millisecond)

	serverConfig.HTTPUri = "http://" + listener.Addr().String()
	os.Exit(m.Run())
}
