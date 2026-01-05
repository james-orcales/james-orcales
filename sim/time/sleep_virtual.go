//go:build sim || sim.virtual_time

package time

import (
	"math/rand/v2"

	"github.com/james-orcales/golang_snacks/sim"
)

var t = sim.UniversalTime

func Sleep(d Duration) {
	if d <= 0 {
		return
	}
	latency := t.SleepLatencyMin + sim.Duration(rand.Int64N(int64(t.SleepLatencyMax-t.SleepLatencyMin+1)))
	sim.Coast(sim.Duration(d) + latency)
}

// type Timer struct {
// 	C         <-chan Time
// 	initTimer bool
// }
//
// func (t *Timer) Stop() bool {
// 	if !t.initTimer {
// 		panic("time: Stop called on uninitialized Timer")
// 	}
// 	return stopTimer(t)
// }
//
// func NewTimer(d Duration) *Timer {
// 	c := make(chan Time, 1)
// 	return &Timer{c, true}
// }
//
// func (t *Timer) Reset(d Duration) bool {
// 	if !t.initTimer {
// 		panic("time: Reset called on uninitialized Timer")
// 	}
// 	w := when(d)
// 	return resetTimer(t, w, 0)
// }
//
// func After(d Duration) <-chan Time {
// 	return NewTimer(d).C
// }
//
// func AfterFunc(d Duration, f func()) *Timer {
// 	return (*Timer)(newTimer(when(d), 0, goFunc, f, nil))
// }
