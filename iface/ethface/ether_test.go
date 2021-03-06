package ethface_test

import (
	"net"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev/ethringdev"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethface"
	"github.com/usnistgov/ndn-dpdk/iface/ifacetestenv"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go4.org/must"
)

func makeEtherLocator(dev ethdev.EthDev) (loc ethface.EtherLocator) {
	loc.Local.HardwareAddr = dev.HardwareAddr()
	loc.Remote.HardwareAddr = packettransport.MulticastAddressNDN
	return
}

type topo3 struct {
	*ifacetestenv.Fixture
	vnet                                           *ethringdev.VNet
	macA, macB, macC                               net.HardwareAddr
	faceAB, faceAC, faceAm, faceBm, faceBA, faceCA iface.Face
}

func makeTopo3(t *testing.T) (topo topo3) {
	_, require := makeAR(t)
	topo.Fixture = ifacetestenv.NewFixture(t)

	var vnetCfg ethringdev.VNetConfig
	vnetCfg.RxPool = ndni.PacketMempool.Get(eal.NumaSocket{})
	vnetCfg.NNodes = 3
	vnet, e := ethringdev.NewVNet(vnetCfg)
	require.NoError(e)
	topo.vnet = vnet

	topo.macA = vnet.Port(0).HardwareAddr()
	topo.macB, _ = net.ParseMAC("02:00:00:00:00:02")
	topo.macC, _ = net.ParseMAC("02:00:00:00:00:03")

	makeFace := func(dev ethdev.EthDev, local, remote net.HardwareAddr) iface.Face {
		loc := makeEtherLocator(dev)
		if local != nil {
			loc.Port = dev.Name()
			loc.Local.HardwareAddr = local
		}
		loc.Remote.HardwareAddr = remote
		face, e := loc.CreateFace()
		require.NoError(e, "%s %s %s", dev.Name(), local, remote)
		return face
	}

	topo.faceAB = makeFace(vnet.Port(0), nil, topo.macB)
	topo.faceAC = makeFace(vnet.Port(0), nil, topo.macC)
	topo.faceAm = makeFace(vnet.Port(0), nil, packettransport.MulticastAddressNDN)
	topo.faceBm = makeFace(vnet.Port(1), topo.macB, packettransport.MulticastAddressNDN)
	topo.faceBA = makeFace(vnet.Port(1), topo.macB, topo.macA)
	topo.faceCA = makeFace(vnet.Port(2), topo.macC, topo.macA)

	ealthread.AllocLaunch(vnet)
	time.Sleep(time.Second)
	return topo
}

func (topo *topo3) Close() error {
	must.Close(topo.Fixture)
	must.Close(topo.vnet)
	return nil
}

func TestTopoBA(t *testing.T) {
	topo := makeTopo3(t)
	defer topo.Close()

	topo.RunTest(topo.faceBA, topo.faceAB)
	topo.CheckCounters()
}

func TestTopoCA(t *testing.T) {
	topo := makeTopo3(t)
	defer topo.Close()

	topo.RunTest(topo.faceCA, topo.faceAC)
	topo.CheckCounters()
}

func TestTopoAm(t *testing.T) {
	assert, _ := makeAR(t)
	topo := makeTopo3(t)
	defer topo.Close()

	locAm := topo.faceAm.Locator().(ethface.EtherLocator)
	assert.Equal("ether", locAm.Scheme())
	assert.Equal(topo.vnet.Port(0).Name(), locAm.Port)
	assert.Equal(topo.macA, locAm.Local.HardwareAddr)
	assert.Equal(packettransport.MulticastAddressNDN, locAm.Remote.HardwareAddr)

	topo.RunTest(topo.faceAm, topo.faceBm)
	topo.CheckCounters()
}

func TestFragmentation(t *testing.T) {
	assert, require := makeAR(t)
	fixture := ifacetestenv.NewFixture(t)
	defer fixture.Close()
	fixture.PayloadLen = 6000
	fixture.DataFrames = 2

	var vnetCfg ethringdev.VNetConfig
	vnetCfg.RxPool = ndni.PacketMempool.Get(eal.NumaSocket{})
	vnetCfg.NNodes = 2
	vnetCfg.LossProbability = 0.01
	vnetCfg.Shuffle = true
	vnet, e := ethringdev.NewVNet(vnetCfg)
	require.NoError(e)
	ealthread.AllocLaunch(vnet)
	time.Sleep(time.Second)

	locA := makeEtherLocator(vnet.Port(0))
	locA.PortConfig = new(ethface.PortConfig)
	locA.PortConfig.MTU = 5000
	locA.PortConfig.DisableSetMTU = true
	faceA, e := locA.CreateFace()
	require.NoError(e)

	locB := makeEtherLocator(vnet.Port(1))
	locB.PortConfig = locA.PortConfig
	faceB, e := locB.CreateFace()
	require.NoError(e)

	fixture.RunTest(faceA, faceB)
	fixture.CheckCounters()

	cntB := faceB.Counters()
	assert.Greater(cntB.RxReassDrops, uint64(0))
}
