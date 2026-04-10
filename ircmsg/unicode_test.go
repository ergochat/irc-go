// Copyright (c) 2021 Shivaram Lingamneni
// Released under the MIT License

package ircmsg

import (
	"strings"
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

type wordWrapTest struct {
	input  string
	width  int
	output []string
}

var wordWrapTests = []wordWrapTest{
	{
		`🍩 boo baz`,
		10,
		[]string{
			`🍩 boo `,
			`baz`,
		},
	},
	{
		`boo baz bar`,
		4,
		[]string{
			`boo `,
			`baz `,
			`bar`,
		},
	},
	{
		`boo baz bar`,
		10,
		[]string{
			`boo baz `,
			`bar`,
		},
	},
	{
		"YOU may remember, my dear friend, that when we lately spent that happy day in the delightful garden and sweet society of the Moulin Joly, I stopped a little in one of our walks, and stayed some time behind the company. We had been shown numberless skeletons of a kind of little fly, called an ephemera, whose successive generations, we were told, were bred and expired within the day. I happened to see a living company of them on a leaf, who appeared to be engaged in conversation. You know I understand all the inferior animal tongues. My too great application to the study of them is the best excuse I can give for the little progress I have made in your charming language. I listened through curiosity to the discourse of these little creatures; but as they, in their national vivacity, spoke three or four together, I could make but little of their conversation. I found, however, by some broken expressions that I heard now and then, they were disputing warmly on the merit of two foreign musicians, one a cousin, the other a moscheto; in which dispute they spent their time, seemingly as regardless of the shortness of life as if they had been sure of living a month. Happy people! thought I; you are certainly under a wise, just, and mild government, since you have no public grievances to complain of, nor any subject of contention but the perfections and imperfections of foreign music. I turned my head from them to an old gray-headed one, who was single on another leaf, and talking to himself. Being amused with his soliloquy, I put it down in writing, in hopes it will likewise amuse her to whom I am so much indebted for the most pleasing of all amusements, her delicious company and heavenly harmony.",
		512,
		[]string{
			"YOU may remember, my dear friend, that when we lately spent that happy day in the delightful garden and sweet society of the Moulin Joly, I stopped a little in one of our walks, and stayed some time behind the company. We had been shown numberless skeletons of a kind of little fly, called an ephemera, whose successive generations, we were told, were bred and expired within the day. I happened to see a living company of them on a leaf, who appeared to be engaged in conversation. You know I understand all ",
			"the inferior animal tongues. My too great application to the study of them is the best excuse I can give for the little progress I have made in your charming language. I listened through curiosity to the discourse of these little creatures; but as they, in their national vivacity, spoke three or four together, I could make but little of their conversation. I found, however, by some broken expressions that I heard now and then, they were disputing warmly on the merit of two foreign musicians, one a cousin, ",
			"the other a moscheto; in which dispute they spent their time, seemingly as regardless of the shortness of life as if they had been sure of living a month. Happy people! thought I; you are certainly under a wise, just, and mild government, since you have no public grievances to complain of, nor any subject of contention but the perfections and imperfections of foreign music. I turned my head from them to an old gray-headed one, who was single on another leaf, and talking to himself. Being amused with his ",
			"soliloquy, I put it down in writing, in hopes it will likewise amuse her to whom I am so much indebted for the most pleasing of all amusements, her delicious company and heavenly harmony."},
	},
}

func TestWordWrap(t *testing.T) {
	for _, test := range wordWrapTests {
		output := WordWrap(test.input, test.width)
		assertEqual(test.output, output)
		// Ensure that if we unwrap the string, it is the same as the original
		assertEqual(test.input, strings.Join(output, ""))
	}
}
