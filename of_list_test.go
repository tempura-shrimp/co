package co_test

import (
	"testing"

	"github.com/smartystreets/goconvey/convey"
	"go.tempura.ink/co"
)

func TestAsyncList(t *testing.T) {
	convey.Convey("given a sequential int", t, func() {
		expected := []int{1, 4, 5, 6, 7, 2, 2, 3, 4, 5, 12, 4, 2, 3, 43, 127, 37598, 34, 34, 123, 123}
		aList := co.OfListWith(expected...)

		convey.Convey("expect resolved list to be identical with given values", func() {
			actual := []int{}
			for data := range aList.Iter() {
				actual = append(actual, data)
			}
			convey.So(actual, convey.ShouldResemble, expected)
		})
	})
}
