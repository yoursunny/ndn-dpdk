package sockettransport

import (
	"fmt"
	"net"

	"github.com/gogf/greuse"
)

type datagramImpl struct {
	nopRedialer
}

func (datagramImpl) Read(tr *transport, buf []byte) (n int, e error) {
	return tr.conn.Read(buf)
}

type pipeImpl struct {
	datagramImpl
}

func (pipeImpl) Dial(network, local, remote string) (net.Conn, error) {
	return nil, fmt.Errorf("cannot dial %s", network)
}

type udpImpl struct {
	datagramImpl
}

func (udpImpl) Dial(network, local, remote string) (net.Conn, error) {
	nla, e := greuse.ResolveAddr(network, local)
	if e != nil {
		return nil, e
	}

	dialer := net.Dialer{
		LocalAddr: nla,
		Control:   reuseControl,
	}
	return dialer.Dial(network, remote)
}

func init() {
	implByNetwork["pipe"] = pipeImpl{}

	implByNetwork["udp"] = udpImpl{}
	implByNetwork["udp4"] = udpImpl{}
	implByNetwork["udp6"] = udpImpl{}
}
