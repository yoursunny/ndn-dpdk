package pktmbuf_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"go4.org/must"
)

func TestPool(t *testing.T) {
	assert, require := makeAR(t)

	mp, e := pktmbuf.NewPool(pktmbuf.PoolConfig{Capacity: 63, Dataroom: 1000}, eal.NumaSocket{})
	require.NoError(e)
	defer mp.Close()

	assert.Equal(63, mp.CountAvailable())
	assert.Equal(0, mp.CountInUse())
	assert.Equal(1000, mp.Dataroom())

	vec0, e := mp.Alloc(33)
	assert.NoError(e)
	assert.Equal(30, mp.CountAvailable())
	assert.Equal(33, mp.CountInUse())
	assert.Len(vec0, 33)

	vec1, e := mp.Alloc(30)
	assert.NoError(e)
	assert.Equal(0, mp.CountAvailable())
	assert.Equal(63, mp.CountInUse())

	vec2, e := mp.Alloc(1)
	assert.Error(e)
	assert.Len(vec2, 0)

	must.Close(vec0)
	must.Close(vec1)
	assert.Equal(63, mp.CountAvailable())
}
