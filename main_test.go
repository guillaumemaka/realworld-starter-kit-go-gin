package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// TestIntegerManipulation is here to get us off the ground and understanding
// how we are going to proceed developing our API using Bahvior Driven Development
func TestIntegerManipulation(t *testing.T) {
	t.Parallel()

	Convey("Given a starting integer value", t, func() {
		x := 42

		Convey("when I call the AddOne function", func() {
			x = AddOne(x)
			Convey("The value should be greater by one", func() {
				So(x, ShouldEqual, 43)
			})
			Convey("and the value should NOT be what it used to be", func() {
				So(x, ShouldNotEqual, 42)
			})
		})
	})
}
