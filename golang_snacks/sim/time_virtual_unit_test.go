//go:build sim.virtual_time

package sim_test

import (
	"context"
	"testing"

	"github.com/james-orcales/golang_snacks/sim"
	"github.com/james-orcales/golang_snacks/snap"
)

func init() {
	sim.UniversalTime = nil
}

func TestBatteryIsNotSet(t *testing.T) {
	snap.Init(`🚨 Assertion Failure 🚨: Battery is initialized`).ExpectPanic(t, func() {
		sim.NewTime[struct{}](nil).Start(context.Background())
	})
}

func TestGreedy(t *testing.T) {
	vtime := sim.NewTime[any](nil)
	vtime.Battery = make(chan any, 1)
	ctx, cancel := context.WithCancel(context.Background())
	vtime.Charge = func() any {
		return "fuck"
	}
	vtime.Drain = func(_ any) {
		vtime.Propel(3 * sim.Day)
	}

	start := vtime.NowSystem()
	vtime.Start(ctx)
	end := vtime.NowSystem()

	snap.Init(`1970-01-01 00:00:00 +0000 UTC`).Expect(t, start.Stdtime())
	snap.Init(`1970-01-01 00:00:17 +0000 UTC`).Expect(t, end.Stdtime())
}

//
// func TestBasic(t *testing.T) {
// 	sim.UniversalTime.Battery = func() {
// 		sim.Cpu(1)
// 	}
//
// 	start := sim.NowSystem()
// 	snap.Init(`2020-04-09 16:15:00 +0000 UTC`).Expect(t, start.Stdtime())
//
// 	sim.Main(func() {
// 		sim.Coast(2 * sim.Minute)
// 		end := sim.NowSystem()
// 		snap.Edit(`2020-04-09 16:17:16 +0000 UTC`).Expect(t, end.Stdtime())
// 	})
// }
