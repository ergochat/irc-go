package ircutils

import (
	"testing"
)

func TestSplitResponse(t *testing.T) {
	assertEqual(EncodeSASLResponse([]byte{}), []string{"+"})
	assertEqual(EncodeSASLResponse(
		[]byte("shivaram\x00shivaram\x00shivarampassphrase")),
		[]string{"c2hpdmFyYW0Ac2hpdmFyYW0Ac2hpdmFyYW1wYXNzcGhyYXNl"},
	)

	// from the examples in the spec:
	assertEqual(
		EncodeSASLResponse([]byte("\x00emersion\x00Est ut beatae omnis ipsam. Quis fugiat deleniti totam qui. Ipsum quam a dolorum tempora velit laborum odit. Et saepe voluptate sed cumque vel. Voluptas sint ab pariatur libero veritatis corrupti. Vero iure omnis ullam. Vero beatae dolores facere fugiat ipsam. Ea est pariatur minima nobis sunt aut ut. Dolores ut laudantium maiores temporibus voluptates. Reiciendis impedit omnis et unde delectus quas ab. Quae eligendi necessitatibus doloribus molestias tempora magnam assumenda.")),
		[]string{
			"AGVtZXJzaW9uAEVzdCB1dCBiZWF0YWUgb21uaXMgaXBzYW0uIFF1aXMgZnVnaWF0IGRlbGVuaXRpIHRvdGFtIHF1aS4gSXBzdW0gcXVhbSBhIGRvbG9ydW0gdGVtcG9yYSB2ZWxpdCBsYWJvcnVtIG9kaXQuIEV0IHNhZXBlIHZvbHVwdGF0ZSBzZWQgY3VtcXVlIHZlbC4gVm9sdXB0YXMgc2ludCBhYiBwYXJpYXR1ciBsaWJlcm8gdmVyaXRhdGlzIGNvcnJ1cHRpLiBWZXJvIGl1cmUgb21uaXMgdWxsYW0uIFZlcm8gYmVhdGFlIGRvbG9yZXMgZmFjZXJlIGZ1Z2lhdCBpcHNhbS4gRWEgZXN0IHBhcmlhdHVyIG1pbmltYSBub2JpcyBz",
			"dW50IGF1dCB1dC4gRG9sb3JlcyB1dCBsYXVkYW50aXVtIG1haW9yZXMgdGVtcG9yaWJ1cyB2b2x1cHRhdGVzLiBSZWljaWVuZGlzIGltcGVkaXQgb21uaXMgZXQgdW5kZSBkZWxlY3R1cyBxdWFzIGFiLiBRdWFlIGVsaWdlbmRpIG5lY2Vzc2l0YXRpYnVzIGRvbG9yaWJ1cyBtb2xlc3RpYXMgdGVtcG9yYSBtYWduYW0gYXNzdW1lbmRhLg==",
		},
	)

	// 400 byte line must be followed by +:
	assertEqual(
		EncodeSASLResponse([]byte("slingamn\x00slingamn\x001111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111")),
		[]string{
			"c2xpbmdhbW4Ac2xpbmdhbW4AMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMQ==",
			"+",
		},
	)
}

func TestBuffer(t *testing.T) {
	b := NewSASLBuffer(1200)

	// less than 400 bytes
	done, output, err := b.Add("c2hpdmFyYW0Ac2hpdmFyYW0Ac2hpdmFyYW1wYXNzcGhyYXNl")
	assertEqual(done, true)
	assertEqual(output, []byte("shivaram\x00shivaram\x00shivarampassphrase"))
	assertEqual(err, nil)

	// 400 bytes exactly plus a continuation +:
	done, output, err = b.Add("c2xpbmdhbW4Ac2xpbmdhbW4AMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMQ==")
	assertEqual(done, false)
	assertEqual(output, []byte(nil))
	assertEqual(err, nil)
	done, output, err = b.Add("+")
	assertEqual(done, true)
	assertEqual(output, []byte("slingamn\x00slingamn\x001111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111"))
	assertEqual(err, nil)

	// over 400 bytes
	done, output, err = b.Add("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==")
	assertEqual(done, true)
	assertEqual(output, []byte(nil))
	assertEqual(err, ErrSASLTooLong)

	// a single +
	done, output, err = b.Add("+")
	assertEqual(done, true)
	assertEqual(output, []byte(nil))
	assertEqual(err, nil)

	// length limit
	for i := 0; i < 4; i++ {
		done, output, err = b.Add("AGVtZXJzaW9uAEVzdCB1dCBiZWF0YWUgb21uaXMgaXBzYW0uIFF1aXMgZnVnaWF0IGRlbGVuaXRpIHRvdGFtIHF1aS4gSXBzdW0gcXVhbSBhIGRvbG9ydW0gdGVtcG9yYSB2ZWxpdCBsYWJvcnVtIG9kaXQuIEV0IHNhZXBlIHZvbHVwdGF0ZSBzZWQgY3VtcXVlIHZlbC4gVm9sdXB0YXMgc2ludCBhYiBwYXJpYXR1ciBsaWJlcm8gdmVyaXRhdGlzIGNvcnJ1cHRpLiBWZXJvIGl1cmUgb21uaXMgdWxsYW0uIFZlcm8gYmVhdGFlIGRvbG9yZXMgZmFjZXJlIGZ1Z2lhdCBpcHNhbS4gRWEgZXN0IHBhcmlhdHVyIG1pbmltYSBub2JpcyBz")
		assertEqual(done, false)
		assertEqual(output, []byte(nil))
		assertEqual(err, nil)
	}
	done, output, err = b.Add("AA==")
	assertEqual(done, true)
	assertEqual(output, []byte(nil))
	assertEqual(err, ErrSASLLimitExceeded)

	// invalid base64
	done, output, err = b.Add("!!!")
	assertEqual(done, true)
	assertEqual(len(output), 0)
	if err == nil {
		t.Errorf("expected non-nil error from invalid base64")
	}

	// two lines
	done, output, err = b.Add("AGVtZXJzaW9uAEVzdCB1dCBiZWF0YWUgb21uaXMgaXBzYW0uIFF1aXMgZnVnaWF0IGRlbGVuaXRpIHRvdGFtIHF1aS4gSXBzdW0gcXVhbSBhIGRvbG9ydW0gdGVtcG9yYSB2ZWxpdCBsYWJvcnVtIG9kaXQuIEV0IHNhZXBlIHZvbHVwdGF0ZSBzZWQgY3VtcXVlIHZlbC4gVm9sdXB0YXMgc2ludCBhYiBwYXJpYXR1ciBsaWJlcm8gdmVyaXRhdGlzIGNvcnJ1cHRpLiBWZXJvIGl1cmUgb21uaXMgdWxsYW0uIFZlcm8gYmVhdGFlIGRvbG9yZXMgZmFjZXJlIGZ1Z2lhdCBpcHNhbS4gRWEgZXN0IHBhcmlhdHVyIG1pbmltYSBub2JpcyBz")
	assertEqual(done, false)
	assertEqual(output, []byte(nil))
	assertEqual(err, nil)
	done, output, err = b.Add("dW50IGF1dCB1dC4gRG9sb3JlcyB1dCBsYXVkYW50aXVtIG1haW9yZXMgdGVtcG9yaWJ1cyB2b2x1cHRhdGVzLiBSZWljaWVuZGlzIGltcGVkaXQgb21uaXMgZXQgdW5kZSBkZWxlY3R1cyBxdWFzIGFiLiBRdWFlIGVsaWdlbmRpIG5lY2Vzc2l0YXRpYnVzIGRvbG9yaWJ1cyBtb2xlc3RpYXMgdGVtcG9yYSBtYWduYW0gYXNzdW1lbmRhLg==")
	assertEqual(done, true)
	assertEqual(output, []byte("\x00emersion\x00Est ut beatae omnis ipsam. Quis fugiat deleniti totam qui. Ipsum quam a dolorum tempora velit laborum odit. Et saepe voluptate sed cumque vel. Voluptas sint ab pariatur libero veritatis corrupti. Vero iure omnis ullam. Vero beatae dolores facere fugiat ipsam. Ea est pariatur minima nobis sunt aut ut. Dolores ut laudantium maiores temporibus voluptates. Reiciendis impedit omnis et unde delectus quas ab. Quae eligendi necessitatibus doloribus molestias tempora magnam assumenda."))
	assertEqual(err, nil)
}
