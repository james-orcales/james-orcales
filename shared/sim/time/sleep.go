//go:build !(sim || sim.virtual_time)

package time

import (
	stdtime "time"
)

func Sleep(d Duration) {
	stdtime.Sleep(stdtime.Duration(d))
}

// type Timer struct {
// 	stdtime.Timer
// }
//
// func NewTimer(d Duration) *Timer {
// 	return &Timer{stdtime.NewTimer(stdtime.Duration(d))}
// }
//
// func (t *Timer) Reset(d Duration) bool {
// 	return t.Timer.Reset(stdtime.Duration(d))
// }
//
// func After(d Duration) <-chan Time {
// 	return NewTimer(d).C
// }
//
// func AfterFunc(d Duration, f func()) *Timer {
// 	return stdtime.AfterFunc(stdtime.Duration(d), f)
// }
