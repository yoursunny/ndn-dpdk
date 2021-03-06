package eal

/*
#include "../../csrc/core/common.h"
*/
import "C"
import (
	"encoding/json"
	"strconv"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"go.uber.org/zap"
)

// NumaSocket represents a NUMA socket.
// Zero value is SOCKET_ID_ANY.
type NumaSocket struct {
	v int // socket ID + 1
}

// NumaSocketFromID converts socket ID to NumaSocket.
func NumaSocketFromID(id int) (socket NumaSocket) {
	if id < 0 || id >= C.RTE_MAX_NUMA_NODES {
		return socket
	}
	socket.v = id + 1
	return socket
}

// ID returns NUMA socket ID.
func (socket NumaSocket) ID() int {
	return socket.v - 1
}

// IsAny returns true if this represents SOCKET_ID_ANY.
func (socket NumaSocket) IsAny() bool {
	return socket.v == 0
}

// Match returns true if either NumaSocket is SOCKET_ID_ANY, or both are the same NumaSocket.
func (socket NumaSocket) Match(other NumaSocket) bool {
	return socket.IsAny() || other.IsAny() || socket.v == other.v
}

func (socket NumaSocket) String() string {
	if socket.IsAny() {
		return "any"
	}
	return strconv.Itoa(socket.ID())
}

// MarshalJSON encodes NUMA socket as number.
// Any is encoded as null.
func (socket NumaSocket) MarshalJSON() ([]byte, error) {
	if socket.IsAny() {
		return json.Marshal(nil)
	}
	return json.Marshal(socket.ID())
}

// ZapField returns a zap.Field for logging.
func (socket NumaSocket) ZapField(key string) zap.Field {
	if socket.IsAny() {
		return zap.String(key, "any")
	}
	return zap.Int(key, socket.ID())
}

// NumaSocket implements WithNumaSocket.
func (socket NumaSocket) NumaSocket() NumaSocket {
	return socket
}

// WithNumaSocket interface is implemented by types that have an associated or preferred NUMA socket.
type WithNumaSocket interface {
	NumaSocket() NumaSocket
}

// GqlWithNumaSocket is a GraphQL field for source object that implements WithNumaSocket.
var GqlWithNumaSocket = &graphql.Field{
	Type:        graphql.Int,
	Name:        "numaSocket",
	Description: "NUMA socket.",
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		socket := p.Source.(WithNumaSocket).NumaSocket()
		return gqlserver.Optional(socket.ID(), !socket.IsAny()), nil
	},
}
