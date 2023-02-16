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

func TestSanitize(t *testing.T) {
	assertEqual(SanitizeText("abc", 10), "abc")
	assertEqual(SanitizeText("abcdef", 5), "abcde")

	assertEqual(SanitizeText("shivaram\x00shivaram\x00shivarampassphrase", 400), "shivaramshivaramshivarampassphrase")

	assertEqual(SanitizeText("the quick brown fox\xffjumps over the lazy dog", 400), "the quick brown fox\xef\xbf\xbdjumps over the lazy dog")

	// \r ignored, \n is two spaces
	assertEqual(SanitizeText("the quick brown fox\r\njumps over the lazy dog", 400), "the quick brown fox  jumps over the lazy dog")
	assertEqual(SanitizeText("the quick brown fox\njumps over the lazy dog", 400), "the quick brown fox  jumps over the lazy dog")
}
