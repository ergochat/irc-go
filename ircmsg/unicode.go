// Copyright (c) 2021 Shivaram Lingamneni
// Released under the MIT License

package ircmsg

import (
	"strings"
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

// TruncateUTF8Safe truncates a message, respecting UTF8 boundaries. If a message
// was originally valid UTF8, TruncateUTF8Safe will not make it invalid; instead
// it will truncate additional bytes as needed, back to the last valid
// UTF8-encoded codepoint. If a message is not UTF8, TruncateUTF8Safe will truncate
// at most 3 additional bytes before giving up.
func TruncateUTF8Safe(message string, byteLimit int) (result string) {
	if len(message) <= byteLimit {
		return message
	}
	message = message[:byteLimit]
	for i := 0; i < (utf8.UTFMax - 1); i++ {
		r, n := utf8.DecodeLastRuneInString(message)
		if r == utf8.RuneError && n <= 1 {
			message = message[:len(message)-1]
		} else {
			break
		}
	}
	return message
}

// WordWrap modified version from tview[1]
// - Removed parsing of tview style tags
// - Changed the width paramater from number of screen cells to byte length
//   as needed by IRC
//
// [1] https://github.com/rivo/tview/blob/v0.42.0/strings.go#L551

// stepState represents the current state of the parser implemented in [step].
type stepState struct {
	unisegState int // The state of the uniseg parser.
	boundaries  int // Information about boundaries, as returned by uniseg.Step.
}

// LineBreak returns whether the string can be broken into the next line after
// the returned grapheme cluster. If optional is true, the line break is
// optional. If false, the line break is mandatory, e.g. after a newline
// character.
func (s *stepState) LineBreak() (lineBreak, optional bool) {
	switch s.boundaries & uniseg.MaskLine {
	case uniseg.LineCanBreak:
		return true, true
	case uniseg.LineMustBreak:
		return true, false
	}
	return false, false // uniseg.LineDontBreak.
}

// step uses uniseg.Step to iterate over the grapheme clusters of a string
//
// This function can be called consecutively to extract all grapheme clusters
// from str. The return values are the first grapheme cluster, the remaining
// string, and the new state. Pass the remaining string and the returned state
// to the next call. If the rest string is empty, parsing is complete. Call the
// returned state's methods for boundary and cluster width information.
//
// Pass nil for state on the first call.
//
// There is no need to call uniseg.HasTrailingLineBreakInString on the last
// non-empty cluster as this function will do this for you and adjust the
// returned boundaries accordingly.
func step(str string, state *stepState) (cluster, rest string, newState *stepState) {
	// Set up initial state.
	if state == nil {
		state = &stepState{
			unisegState: -1,
		}
	}
	if len(str) == 0 {
		newState = state
		return
	}

	// Get a grapheme cluster.
	preState := state.unisegState
	cluster, rest, state.boundaries, state.unisegState = uniseg.StepString(str, preState)
	if rest == "" {
		if !uniseg.HasTrailingLineBreakInString(cluster) {
			state.boundaries &^= uniseg.MaskLine
		}
	}

	newState = state
	return
}

// WordWrap splits a text such that each resulting line does not exceed the
// given number of bytes. Split points are determined using the algorithm
// described in [Unicode Standard Annex #14].
//
// [Unicode Standard Annex #14]: https://www.unicode.org/reports/tr14/
func WordWrap(text string, byteWidth int) (lines []string) {
	if byteWidth <= 0 {
		return
	}

	var (
		state      *stepState
		lineLength int
		lastOption int
		cluster    string
	)
	str := text
	for len(str) > 0 {
		// Parse the next character.
		cluster, str, state = step(str, state)
		graphemeBytes := len(cluster)

		// Would it exceed the line byteWidth?
		if lineLength+graphemeBytes > byteWidth {
			if lastOption == 0 {
				// No split point so far. Just split at the current position.
				lines = append(lines, text[:lineLength])
				text = text[lineLength:]
				lineLength, lastOption = 0, 0
			} else {
				// Split at the last split point.
				lines = append(lines, text[:lastOption])
				text = text[lastOption:]
				lineLength -= lastOption
				lastOption = 0
			}
		}

		// Move ahead.
		lineLength += graphemeBytes

		// Check for split points.
		if lineBreak, optional := state.LineBreak(); lineBreak {
			if optional {
				// Remember this split point.
				lastOption = lineLength
			} else {
				// We must split here.
				lines = append(lines, strings.TrimRight(text[:lineLength], "\n\r"))
				text = text[lineLength:]
				lineLength, lastOption = 0, 0
			}
		}
	}
	lines = append(lines, text)

	return
}
