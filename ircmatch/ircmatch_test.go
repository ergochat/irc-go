package ircmatch

import "testing"

var successfulIRCMatches = map[string][]string{
	"d?n*":   {"dan123", "ddn53f"},
	"d?n**d": {"dan123ad", "ddn53fd"},
}

var failedIRCMatches = map[string][]string{
	"d?n*":   {"dn123", "dna53f"},
	"d?**n*": {"dn123", "dna53f"},
}

func TestSuccessfulMatches(t *testing.T) {
	for globString, matches := range successfulIRCMatches {
		matcher := MakeMatch(globString)

		for _, match := range matches {
			if !matcher.Match(match) {
				t.Error(
					"Expected", globString,
					"to match on", match,
					"but it did not",
				)
			}
		}
	}
}

func TestFailedMatches(t *testing.T) {
	for globString, matches := range failedIRCMatches {
		matcher := MakeMatch(globString)

		for _, match := range matches {
			if matcher.Match(match) {
				t.Error(
					"Expected", globString,
					"to fail matching on", match,
					"but it matched",
				)
			}
		}
	}
}
