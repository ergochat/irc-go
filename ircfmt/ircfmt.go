// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircfmt

import (
	"strings"
)

const (
	// raw bytes and strings to do replacing with
	bold      string = "\x02"
	colour    string = "\x03"
	italic    string = "\x1d"
	underline string = "\x1f"
	reset     string = "\x0f"

	runecolour rune = '\x03'

	// valid characters for an initial colour code, for speed
	colours1 string = "0123456789"
)

var (
	// valtoescape replaces most of IRC characters with our escapes.
	// colour is not replaced here because of what we need to do involving colour names.
	valtoescape = strings.NewReplacer("$", "$$", bold, "$b", italic, "$i", underline, "$u", reset, "$r")

	// escapetoval contains most of our escapes and how they map to real IRC characters.
	escapetoval = map[byte]string{
		'$': "$",
		'b': bold,
		'i': italic,
		'u': underline,
		'r': reset,
	}

	// valid colour codes
	numtocolour = [][]string{
		{"15", "light grey"},
		{"14", "grey"},
		{"13", "pink"},
		{"12", "light blue"},
		{"11", "light cyan"},
		{"10", "cyan"},
		{"09", "light green"},
		{"08", "yellow"},
		{"07", "orange"},
		{"06", "magenta"},
		{"05", "brown"},
		{"04", "red"},
		{"03", "green"},
		{"02", "blue"},
		{"01", "black"},
		{"00", "white"},
		{"9", "light green"},
		{"8", "yellow"},
		{"7", "orange"},
		{"6", "magenta"},
		{"5", "brown"},
		{"4", "red"},
		{"3", "green"},
		{"2", "blue"},
		{"1", "black"},
		{"0", "white"},
	}

	// full and truncated colour codes
	colourcodesFull = map[string]string{
		"white":       "00",
		"black":       "01",
		"blue":        "02",
		"green":       "03",
		"red":         "04",
		"brown":       "05",
		"magenta":     "06",
		"orange":      "07",
		"yellow":      "08",
		"light green": "09",
		"cyan":        "10",
		"light cyan":  "11",
		"light blue":  "12",
		"pink":        "13",
		"grey":        "14",
		"light grey":  "15",
	}
	colourcodesTruncated = map[string]string{
		"white":       "0",
		"black":       "1",
		"blue":        "2",
		"green":       "3",
		"red":         "4",
		"brown":       "5",
		"magenta":     "6",
		"orange":      "7",
		"yellow":      "8",
		"light green": "9",
		"cyan":        "10",
		"light cyan":  "11",
		"light blue":  "12",
		"pink":        "13",
		"grey":        "14",
		"light grey":  "15",
	}
)

// Escape takes a raw IRC string and returns it with our escapes.
//
// IE, it turns this: "This is a \x02cool\x02, \x034red\x0f message!"
// into: "This is a $bcool$b, $c[red]red$r message!"
func Escape(in string) string {
	// replace all our usual escapes
	in = valtoescape.Replace(in)

	// replace colour codes
	out := ""
	var skip int
	for i, x := range in {
		// skip chars if necessary
		if 0 < skip {
			skip--
			continue
		}

		if x == runecolour {
			out += "$c"
			i++ // to refer to color code

			if len(in) < i+2 || !strings.Contains(colours1, string(in[i])) {
				out += "[]"
				continue
			}

			out += "["

			in = in[i:]
			for _, vals := range numtocolour {
				code, name := vals[0], vals[1]
				if strings.HasPrefix(in, code) {
					in = strings.TrimPrefix(in, code)
					out += name
					i = 0 // refer to char after colour code
					skip += len(code)

					if i+2 < len(in) && in[i] == ',' {
						i++ // refer to colour code after comma
						skip++
						in := in[i:]
						for _, vals = range numtocolour {
							code, name = vals[0], vals[1]
							if strings.HasPrefix(in, code) {
								out += ","
								out += name
								skip += len(code)
								break
							}
						}
					}
					break
				}
			}

			out += "]"
		} else {
			out += string(x)
		}
	}

	return out
}

// Unescape takes our escaped string and returns a raw IRC string.
//
// IE, it turns this: "This is a $bcool$b, $c[red]red$r message!"
// into this: "This is a \x02cool\x02, \x034red\x0f message!"
func Unescape(in string) string {
	out := ""

	var skip int
	for i, x := range in {
		// skip if necessary
		if 0 < skip {
			skip--
			continue
		}

		// chars exist and formatting code thrown our way
		i++ // to now refer to the formatting code character
		if x == '$' && 0 < len(in)-i {
			val, exists := escapetoval[in[i]]
			if exists == true {
				skip++ // to skip the formatting code character
				out += val
			} else if in[i] == 'c' {
				skip++ // to skip the formatting code character
				out += colour

				// ensure '[' follows before doing further processing
				i++ // refer to the opening bracket
				if (len(in)-i) < 1 || in[i] != '[' {
					continue
				} else {
					// strip leading '['
					skip++
				}

				var buffer string
				var colournames []string
				for j, y := range in {
					// get to color names and all
					if j <= i {
						continue
					}

					// skip this character in the real loop as well
					skip++
					// so we can refer to the char after the loop as well
					i = j

					// record color names
					if y == ']' {
						i++
						break
					} else if y == ',' {
						colournames = append(colournames, buffer)
						buffer = ""
					} else {
						buffer += string(y)
					}
				}
				colournames = append(colournames, buffer)

				if len(colournames) > 1 {
					out += colourcodesTruncated[colournames[0]]
					out += ","
					if i < len(in) && strings.Contains(colours1, string(in[i])) {
						out += colourcodesFull[colournames[1]]
					} else {
						out += colourcodesTruncated[colournames[1]]
					}
				} else {
					if i < len(in) && strings.Contains(colours1, string(in[i])) {
						out += colourcodesFull[colournames[0]]
					} else {
						out += colourcodesTruncated[colournames[0]]
					}
				}
			} else {
				// unknown formatting character, intentionally fall-through
			}
		} else {
			out += string(x)
		}
	}

	return out
}
