package ircmap

import "testing"

type testcase struct {
	raw    string
	folded string
}

var equalASCIITests = []testcase{
	{"Tes4tstsASFd", "tes4tstsasfd"},
	{"ONsot{[}]sadf", "onsot{[}]sadf"},
	{"#K03jmn0r-4GD", "#k03jmn0r-4gd"},
}

var equalRFC1459Tests = []testcase{
	{"rTes4tstsASFd", "rtes4tstsasfd"},
	{"rONsot{[}]sadf", "ronsot{{}}sadf"},
	{"#rK03j\\mn0r-4GD", "#rk03j|mn0r-4gd"},
}

func TestASCII(t *testing.T) {
	for _, pair := range equalASCIITests {
		val := Casefold(ASCII, pair.raw)

		if val != pair.folded {
			t.Error(
				"For", pair.raw,
				"expected", pair.folded,
				"got", val,
			)
		}
	}
}

func TestRFC1459(t *testing.T) {
	for _, pair := range equalRFC1459Tests {
		val := Casefold(RFC1459, pair.raw)

		if val != pair.folded {
			t.Error(
				"For", pair.raw,
				"expected", pair.folded,
				"got", val,
			)
		}
	}
}
