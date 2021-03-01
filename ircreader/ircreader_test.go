// Copyright (c) 2020 Shivaram Lingamneni <slingamn@cs.stanford.edu>
// released under the MIT license

package ircreader

import (
	"fmt"
	"io"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

// mockConn is a fake io.Reader that yields len(counts) lines,
// each consisting of counts[i] 'a' characters and a terminating '\n'
type mockConn struct {
	counts []int
}

func min(i, j int) (m int) {
	if i < j {
		return i
	} else {
		return j
	}
}

func (c *mockConn) Read(b []byte) (n int, err error) {
	for len(b) > 0 {
		if len(c.counts) == 0 {
			return n, io.EOF
		}
		if c.counts[0] == 0 {
			b[0] = '\n'
			c.counts = c.counts[1:]
			b = b[1:]
			n += 1
			continue
		}
		size := min(c.counts[0], len(b))
		for i := 0; i < size; i++ {
			b[i] = 'a'
		}
		c.counts[0] -= size
		b = b[size:]
		n += size
	}
	return n, nil
}

func (c *mockConn) Write(b []byte) (n int, err error) {
	return
}

func (c *mockConn) Close() error {
	c.counts = nil
	return nil
}

func newMockConn(counts []int) *mockConn {
	cpCounts := make([]int, len(counts))
	copy(cpCounts, counts)
	return &mockConn{
		counts: cpCounts,
	}
}

// construct a mock reader with some number of \n-terminated lines,
// verify that IRCStreamConn can read and split them as expected
func doLineReaderTest(counts []int, t *testing.T) {
	c := newMockConn(counts)
	r := NewIRCReader(c)
	var readCounts []int
	for {
		line, err := r.ReadLine()
		if err == nil {
			readCounts = append(readCounts, len(line))
		} else if err == io.EOF {
			break
		} else {
			panic(err)
		}
	}

	if !reflect.DeepEqual(counts, readCounts) {
		t.Errorf("expected %#v, got %#v", counts, readCounts)
	}
}

const (
	maxMockReaderLen     = 100
	maxMockReaderLineLen = 4096 + 511
)

func TestLineReader(t *testing.T) {
	counts := []int{44, 428, 3, 0, 200, 2000, 0, 4044, 33, 3, 2, 1, 0, 1, 2, 3, 48, 555}
	doLineReaderTest(counts, t)

	// fuzz
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 1000; i++ {
		countsLen := r.Intn(maxMockReaderLen) + 1
		counts := make([]int, countsLen)
		for i := 0; i < countsLen; i++ {
			counts[i] = r.Intn(maxMockReaderLineLen)
		}
		doLineReaderTest(counts, t)
	}
}

type mockConnLimits struct {
	// simulates the arrival of data via TCP;
	// each Read() call will read from at most one of the slices
	reads [][]byte
}

func (c *mockConnLimits) Read(b []byte) (n int, err error) {
	if len(c.reads) == 0 {
		return n, io.EOF
	}
	readLen := min(len(c.reads[0]), len(b))
	copy(b[:readLen], c.reads[0][:readLen])
	c.reads[0] = c.reads[0][readLen:]
	if len(c.reads[0]) == 0 {
		c.reads = c.reads[1:]
	}
	return readLen, nil
}

func makeLine(length int, ending bool) (result []byte) {
	totalLen := length
	if ending {
		totalLen++
	}
	result = make([]byte, totalLen)
	for i := 0; i < length; i++ {
		result[i] = 'a'
	}
	if ending {
		result[len(result)-1] = '\n'
	}
	return
}

func assertEqual(found, expected interface{}) {
	if !reflect.DeepEqual(found, expected) {
		panic(fmt.Sprintf("expected %#v, found %#v", expected, found))
	}
}

func TestRegression(t *testing.T) {
	var c mockConnLimits
	// this read fills up the buffer with a terminated line:
	c.reads = append(c.reads, makeLine(4605, true))
	// this is a large, unterminated read:
	c.reads = append(c.reads, makeLine(4095, false))
	// this terminates the previous read, within the acceptable limit:
	c.reads = append(c.reads, makeLine(500, true))

	var cc IRCReader
	cc.Initialize(&c, 512, 4096+512)

	line, err := cc.ReadLine()
	assertEqual(len(line), 4605)
	assertEqual(err, nil)

	line, err = cc.ReadLine()
	assertEqual(len(line), 4595)
	assertEqual(err, nil)

	line, err = cc.ReadLine()
	assertEqual(err, io.EOF)
}
