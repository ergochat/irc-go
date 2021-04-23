// Copyright (c) 2021 Shivaram Lingamneni
// Released under the MIT License

package ircutils

import (
	"fmt"
	"reflect"
	"testing"
)

func assertEqual(found, expected interface{}) {
	if !reflect.DeepEqual(found, expected) {
		panic(fmt.Sprintf("expected %#v, found %#v", expected, found))
	}
}
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

func TestSanitize(t *testing.T) {
	assertEqual(SanitizeText("abc", 10), "abc")
	assertEqual(SanitizeText("abcdef", 5), "abcde")

	assertEqual(SanitizeText("shivaram\x00shivaram\x00shivarampassphrase", 400), "shivaramshivaramshivarampassphrase")

	assertEqual(SanitizeText("the quick brown fox\xffjumps over the lazy dog", 400), "the quick brown fox\xef\xbf\xbdjumps over the lazy dog")

	// \r ignored, \n is two spaces
	assertEqual(SanitizeText("the quick brown fox\r\njumps over the lazy dog", 400), "the quick brown fox  jumps over the lazy dog")
	assertEqual(SanitizeText("the quick brown fox\njumps over the lazy dog", 400), "the quick brown fox  jumps over the lazy dog")
}
