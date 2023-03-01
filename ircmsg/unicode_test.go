// Copyright (c) 2021 Shivaram Lingamneni
// Released under the MIT License

package ircmsg

import (
	"testing"
)

func TestTruncateUTF8(t *testing.T) {
	assertEqual(TruncateUTF8Safe("fffff", 512), "fffff")
	assertEqual(TruncateUTF8Safe("fffff", 5), "fffff")
	assertEqual(TruncateUTF8Safe("ffffff", 5), "fffff")
	assertEqual(TruncateUTF8Safe("ffffffffff", 5), "fffff")

	assertEqual(TruncateUTF8Safe("12345🐬", 9), "12345🐬")
	assertEqual(TruncateUTF8Safe("12345🐬", 8), "12345")
	assertEqual(TruncateUTF8Safe("12345🐬", 7), "12345")
	assertEqual(TruncateUTF8Safe("12345🐬", 6), "12345")
	assertEqual(TruncateUTF8Safe("12345", 5), "12345")

	assertEqual(TruncateUTF8Safe("\xff\xff\xff\xff\xff\xff", 512), "\xff\xff\xff\xff\xff\xff")
	assertEqual(TruncateUTF8Safe("\xff\xff\xff\xff\xff\xff", 6), "\xff\xff\xff\xff\xff\xff")
	// shouldn't truncate the whole string
	assertEqual(TruncateUTF8Safe("\xff\xff\xff\xff\xff\xff", 5), "\xff\xff")
}

func BenchmarkTruncateUTF8Invalid(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TruncateUTF8Safe("\xff\xff\xff\xff\xff\xff", 5)
	}
}

func BenchmarkTruncateUTF8Valid(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TruncateUTF8Safe("12345🐬", 8)
	}
}
