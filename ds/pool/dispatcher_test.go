package pool_test

import (
	"testing"

	"github.com/smartystreets/goconvey/convey"
	"go.tempura.ink/co/ds/pool"
)

func TestDispatchPool(t *testing.T) {
	convey.Convey("given a sequential tasks", t, func() {
		markers := make([]bool, 1000)

		p := pool.NewDispatchPool[any](10)

		for i := 0; i < 1000; i++ {
			func(idx int) {
				p.AddJob(func() any {
					markers[idx] = true
					return nil
				})
			}(i)
		}
		convey.Convey("On wait", func() {
			p.Wait()

			convey.Convey("Each markers should be marked", func() {
				for i := 0; i < 1000; i++ {
					convey.So(markers[i], convey.ShouldEqual, true)
				}
			})
		})
	})
}
