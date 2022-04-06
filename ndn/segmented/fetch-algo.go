package segmented

import (
	"container/list"
	"math"
	"time"

	"github.com/zyedidia/generic"
)

const (
	cubicIw    = 2
	cubicC     = 0.4
	cubicBeta  = 0.7
	cubicAlpha = 3 * (1 - cubicBeta) / (1 + cubicBeta)
)

type cubic struct {
	t0       int64
	cwnd     float64
	wMax     float64
	wLastMax float64
	k        float64
	ssthresh float64
}

func (ca *cubic) Cwnd() int {
	return generic.Max(cubicIw, int(ca.cwnd))
}

func (ca *cubic) Increase(now time.Time, rtt time.Duration) {
	nowV := now.UnixNano()
	if nowV <= ca.t0 {
		return
	}

	if ca.cwnd < ca.ssthresh { // slow start
		ca.cwnd += 1.0
		return
	}

	t := float64(nowV-ca.t0) / float64(time.Second)
	rttV := rtt.Seconds()
	wCubic := cubicC*math.Pow(t-ca.k, 3) + ca.wMax
	wEst := ca.wMax*cubicBeta + cubicAlpha*(t/rttV)
	if wCubic < wEst { // TCP friendly region
		ca.cwnd = wEst
		return
	}

	// concave region or convex region
	// note: RFC8312 specifies `(W_cubic(t+RTT) - cwnd) / cwnd`, but NDN-DPDK benchmark shows
	//       that using `(W_cubic(t) - cwnd) / cwnd` increases throughput by 10%
	ca.cwnd += (wCubic - ca.cwnd) / ca.cwnd
}

func (ca *cubic) Decrease(now time.Time) {
	ca.t0 = now.UnixNano()
	if ca.cwnd < ca.wLastMax {
		ca.wLastMax = ca.cwnd
		ca.wMax = ca.cwnd * (1 + cubicBeta) / 2
	} else {
		ca.wMax = ca.cwnd
		ca.wLastMax = ca.cwnd
	}
	ca.k = math.Cbrt(ca.wMax * (1 - cubicBeta) / cubicC)
	ca.cwnd *= cubicBeta
	ca.ssthresh = generic.Max(ca.cwnd, 2)
}

func newCubic() (ca *cubic) {
	return &cubic{
		cwnd:     cubicIw,
		ssthresh: math.Inf(1),
	}
}

type fetchSeg struct {
	TxTime      time.Time
	RtoExpiry   time.Time
	NRetx       int
	RetxElement *list.Element
}

func (fs *fetchSeg) setTimeNow(rto time.Duration) {
	fs.TxTime = time.Now()
	fs.RtoExpiry = fs.TxTime.Add(rto)
}
