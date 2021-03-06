// Package ealinit initializes DPDK EAL and SPDK main thread.
package ealinit

/*
#include "../../csrc/dpdk/mbuf.h"
#include <rte_eal.h>
#include <rte_lcore.h>
*/
import "C"
import (
	"os"
	"runtime"
	"sync"

	"github.com/kballard/go-shellquote"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealconfig"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
	"go.uber.org/zap"
)

var logger = logging.New("ealinit")

func init() {
	ealconfig.PmdPath = C.RTE_EAL_PMD_PATH
}

var initOnce sync.Once

// Init initializes DPDK and SPDK.
// args should not include program name.
// Panics on error.
func Init(args []string) {
	initOnce.Do(func() {
		updateLogLevels()
		initLogStream()
		assignMainThread := make(chan *spdkenv.Thread)
		go func() {
			runtime.LockOSThread()
			initEal(args)
			initMbufDynfields()
			spdkenv.InitEnv()
			spdkenv.InitMainThread(assignMainThread) // never returns
		}()
		th := <-assignMainThread
		updateLogLevels()
		eal.MainThread = th
		eal.MainReadSide = th.RcuReadSide
		eal.CallMain(func() {
			logger.Debug("MainThread is running")
		})
		spdkenv.InitFinal()
	})
}

func initEal(args []string) {
	logEntry := logger.With(zap.String("args", shellquote.Join(args...)))
	exe, e := os.Executable()
	if e != nil {
		exe = os.Args[0]
	}
	argv := append([]string{exe}, args...)
	a := cptr.NewCArgs(argv)
	defer a.Close()

	C.rte_mp_disable()
	res := C.rte_eal_init(C.int(a.Argc), (**C.char)(a.Argv))
	if res < 0 {
		logEntry.Fatal("EAL init error", zap.Error(eal.GetErrno()))
		return
	}

	lcoreSockets := map[int]int{}
	for lcID := C.rte_get_next_lcore(C.RTE_MAX_LCORE, 1, 1); lcID < C.RTE_MAX_LCORE; lcID = C.rte_get_next_lcore(lcID, 1, 0) {
		lcoreSockets[int(lcID)] = int(C.rte_lcore_to_socket_id(lcID))
	}
	eal.UpdateLCoreSockets(lcoreSockets, int(C.rte_get_main_lcore()))
	logEntry.Info("EAL ready",
		eal.MainLCore.ZapField("main"),
		zap.Array("workers", eal.Workers),
		zap.Any("sockets", eal.Sockets),
	)
}

func initMbufDynfields() {
	ok := bool(C.Mbuf_RegisterDynFields())
	if !ok {
		logger.Fatal("mbuf dynfields init error",
			zap.Error(eal.GetErrno()),
		)
	}
}
