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

var equalRFC3454Tests = []testcase{
	{"#TeStChAn", "#testchan"},
	{"#beßtchannEL", "#besstchannel"},
	{"３４５６3456", "34563456"},
}

var equalRFC7613Tests = []testcase{
	{"#TeStChAn", "#testchan"},
	{"#beßtchannEL", "#beßtchannel"},
	{"３４５６3456", "34563456"},
}

func TestASCII(t *testing.T) {
	for _, pair := range equalASCIITests {
		val, err := Casefold(ASCII, pair.raw)

		if err != nil {
			t.Error(
				"For", pair.raw,
				"expected", pair.folded,
				"but we got an error:", err.Error(),
			)
		}
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
		val, err := Casefold(RFC1459, pair.raw)

		if err != nil {
			t.Error(
				"For", pair.raw,
				"expected", pair.folded,
				"but we got an error:", err.Error(),
			)
		}
		if val != pair.folded {
			t.Error(
				"For", pair.raw,
				"expected", pair.folded,
				"got", val,
			)
		}
	}
}

func TestRFC3454(t *testing.T) {
	for _, pair := range equalRFC3454Tests {
		val, err := Casefold(RFC3454, pair.raw)

		if err != nil {
			t.Error(
				"For", pair.raw,
				"expected", pair.folded,
				"but we got an error:", err.Error(),
			)
		}
		if val != pair.folded {
			t.Error(
				"For", pair.raw,
				"expected", pair.folded,
				"got", val,
			)
		}
	}
}

func TestRFC7613(t *testing.T) {
	for _, pair := range equalRFC7613Tests {
		val, err := Casefold(RFC7613, pair.raw)

		if err != nil {
			t.Error(
				"For", pair.raw,
				"expected", pair.folded,
				"but we got an error:", err.Error(),
			)
		}
		if val != pair.folded {
			t.Error(
				"For", pair.raw,
				"expected", pair.folded,
				"got", val,
			)
		}
	}
}
