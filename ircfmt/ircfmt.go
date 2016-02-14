// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircfmt

import "strings"

const (
	// raw bytes and strings to do replacing with
	bold      string = "\x02"
	colour    string = "\x03"
	italic    string = "\x1d"
	underline string = "\x1f"
	reset     string = "\x0f"

	bytecolour byte = '\x03'

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
	for len(in) > 0 {
		if in[0] == bytecolour {
			out += "$c"
			in = in[1:]

			if len(in) < 1 || !strings.Contains(colours1, string(in[0])) {
				out += "[]"
				continue
			}

			out += "["

			for _, vals := range numtocolour {
				code, name := vals[0], vals[1]
				if strings.HasPrefix(in, code) {
					in = strings.TrimPrefix(in, code)
					out += name

					if len(in) > 1 && in[0] == ',' {
						searchin := in[1:]
						for _, vals = range numtocolour {
							code, name = vals[0], vals[1]
							if strings.HasPrefix(searchin, code) {
								out += ","
								out += name
								in = strings.TrimPrefix(in[1:], code)
								break
							}
						}
					}
					break
				}
			}

			out += "]"
		} else {
			out += string(in[0])
			in = in[1:]
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

	for len(in) > 0 {
		if in[0] == '$' && len(in) > 1 {
			val, exists := escapetoval[in[1]]
			if exists == true {
				out += string(val)
				in = in[2:]
			} else if in[1] == 'c' {
				out += colour

				in = in[2:]

				// ensure '[' follows before doing further processing
				if len(in) < 1 || in[0] != '[' {
					continue
				} else {
					// strip leading '['
					in = in[1:]
				}

				splitin := strings.SplitN(in, "]", 2)
				colournames := strings.Split(splitin[0], ",")
				in = splitin[1]

				if len(colournames) > 1 {
					out += colourcodesTruncated[colournames[0]]
					out += ","
					if len(in) > 0 && strings.Contains(colours1, string(in[0])) {
						out += colourcodesFull[colournames[1]]
					} else {
						out += colourcodesTruncated[colournames[1]]
					}
				} else {
					if len(in) > 0 && strings.Contains(colours1, string(in[0])) {
						out += colourcodesFull[colournames[0]]
					} else {
						out += colourcodesTruncated[colournames[0]]
					}
				}
			} else {
				out += string(in[1])
				in = in[2:]
			}
		} else {
			out += string(in[0])
			in = in[1:]
		}
	}

	return out
}
