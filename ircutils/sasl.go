package ircutils

import (
	"encoding/base64"
)

// EncodeSASLResponse encodes a raw SASL response as parameters to successive
// AUTHENTICATE commands, as described in the IRCv3 SASL specification.
func EncodeSASLResponse(raw []byte) (result []string) {
	// https://ircv3.net/specs/extensions/sasl-3.1#the-authenticate-command
	// "The response is encoded in Base64 (RFC 4648), then split to 400-byte chunks,
	// and each chunk is sent as a separate AUTHENTICATE command. Empty (zero-length)
	// responses are sent as AUTHENTICATE +. If the last chunk was exactly 400 bytes
	// long, it must also be followed by AUTHENTICATE + to signal end of response."

	if len(raw) == 0 {
		return []string{"+"}
	}

	response := base64.StdEncoding.EncodeToString(raw)
	lastLen := 0
	for len(response) > 0 {
		// TODO once we require go 1.21, this can be: lastLen = min(len(response), 400)
		lastLen = len(response)
		if lastLen > 400 {
			lastLen = 400
		}
		result = append(result, response[:lastLen])
		response = response[lastLen:]
	}

	if lastLen == 400 {
		result = append(result, "+")
	}

	return result
}
