package ircevent

import (
	"crypto/tls"
	"fmt"
	"os"
	"testing"

	"github.com/ergochat/irc-go/ircmsg"
)

const (
	serverEnvVar = "IRCEVENT_SERVER"
	saslAccVar   = "IRCEVENT_SASL_LOGIN"
	saslPassVar  = "IRCEVENT_SASL_PASSWORD"
)

func setSaslTestCreds(irc *Connection, t *testing.T) {
	acc := os.Getenv(saslAccVar)
	if acc == "" {
		t.Fatalf("define %s and %s environment variables to test SASL", saslAccVar, saslPassVar)
	}
	irc.SASLLogin = acc
	irc.SASLPassword = os.Getenv(saslPassVar)
}

func getenv(key, defaultValue string) (value string) {
	value = os.Getenv(key)
	if value == "" {
		value = defaultValue
	}
	return
}

func getServer(sasl bool) string {
	port := 6667
	if sasl {
		port = 6697
	}
	return fmt.Sprintf("%s:%d", getenv(serverEnvVar, "localhost"), port)
}

// set SASLLogin and SASLPassword environment variables before testing
func runCAPTest(caps []string, useSASL bool, t *testing.T) {
	irccon := connForTesting("go-eventirc", "go-eventirc", true)
	irccon.Debug = true
	irccon.UseTLS = true
	if useSASL {
		setSaslTestCreds(irccon, t)
	}
	irccon.RequestCaps = caps
	irccon.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	irccon.AddCallback("001", func(e ircmsg.Message) { irccon.Join("#go-eventirc") })

	irccon.AddCallback("366", func(e ircmsg.Message) {
		irccon.Privmsg("#go-eventirc", "Test Message SASL")
		irccon.Quit()
	})

	err := irccon.Connect()
	if err != nil {
		t.Fatalf("SASL failed: %s", err)
	}
	irccon.Loop()
}

func TestConnectionSASL(t *testing.T) {
	runCAPTest(nil, true, t)
}

func TestConnectionSASLWithAdditionalCaps(t *testing.T) {
	runCAPTest([]string{"server-time", "message-tags", "batch", "labeled-response", "echo-message"}, true, t)
}

func TestConnectionSASLWithNonexistentCaps(t *testing.T) {
	runCAPTest([]string{"server-time", "message-tags", "batch", "labeled-response", "echo-message", "oragono.io/xyzzy"}, true, t)
}

func TestConnectionSASLWithNonexistentCapsOnly(t *testing.T) {
	runCAPTest([]string{"oragono.io/xyzzy"}, true, t)
}

func TestConnectionNonexistentCAPOnly(t *testing.T) {
	runCAPTest([]string{"oragono.io/xyzzy"}, false, t)
}

func TestConnectionNonexistentCAPs(t *testing.T) {
	runCAPTest([]string{"oragono.io/xyzzy", "server-time", "message-tags"}, false, t)
}

func TestConnectionGoodCAPs(t *testing.T) {
	runCAPTest([]string{"server-time", "message-tags"}, false, t)
}

func TestSASLFail(t *testing.T) {
	irccon := connForTesting("go-eventirc", "go-eventirc", true)
	irccon.Debug = true
	irccon.UseTLS = true
	setSaslTestCreds(irccon, t)
	irccon.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	irccon.AddCallback("001", func(e ircmsg.Message) { irccon.Join("#go-eventirc") })
	// intentionally break the password
	irccon.SASLPassword = irccon.SASLPassword + "_"
	err := irccon.Connect()
	if err == nil {
		t.Errorf("successfully connected with invalid password")
	}
}

func TestSplitResponse(t *testing.T) {
	assertEqual(splitSaslResponse([]byte{}), []string{"+"})
	assertEqual(splitSaslResponse(
		[]byte("shivaram\x00shivaram\x00shivarampassphrase")),
		[]string{"c2hpdmFyYW0Ac2hpdmFyYW0Ac2hpdmFyYW1wYXNzcGhyYXNl"},
	)

	// from the examples in the spec:
	assertEqual(
		splitSaslResponse([]byte("\x00emersion\x00Est ut beatae omnis ipsam. Quis fugiat deleniti totam qui. Ipsum quam a dolorum tempora velit laborum odit. Et saepe voluptate sed cumque vel. Voluptas sint ab pariatur libero veritatis corrupti. Vero iure omnis ullam. Vero beatae dolores facere fugiat ipsam. Ea est pariatur minima nobis sunt aut ut. Dolores ut laudantium maiores temporibus voluptates. Reiciendis impedit omnis et unde delectus quas ab. Quae eligendi necessitatibus doloribus molestias tempora magnam assumenda.")),
		[]string{
			"AGVtZXJzaW9uAEVzdCB1dCBiZWF0YWUgb21uaXMgaXBzYW0uIFF1aXMgZnVnaWF0IGRlbGVuaXRpIHRvdGFtIHF1aS4gSXBzdW0gcXVhbSBhIGRvbG9ydW0gdGVtcG9yYSB2ZWxpdCBsYWJvcnVtIG9kaXQuIEV0IHNhZXBlIHZvbHVwdGF0ZSBzZWQgY3VtcXVlIHZlbC4gVm9sdXB0YXMgc2ludCBhYiBwYXJpYXR1ciBsaWJlcm8gdmVyaXRhdGlzIGNvcnJ1cHRpLiBWZXJvIGl1cmUgb21uaXMgdWxsYW0uIFZlcm8gYmVhdGFlIGRvbG9yZXMgZmFjZXJlIGZ1Z2lhdCBpcHNhbS4gRWEgZXN0IHBhcmlhdHVyIG1pbmltYSBub2JpcyBz",
			"dW50IGF1dCB1dC4gRG9sb3JlcyB1dCBsYXVkYW50aXVtIG1haW9yZXMgdGVtcG9yaWJ1cyB2b2x1cHRhdGVzLiBSZWljaWVuZGlzIGltcGVkaXQgb21uaXMgZXQgdW5kZSBkZWxlY3R1cyBxdWFzIGFiLiBRdWFlIGVsaWdlbmRpIG5lY2Vzc2l0YXRpYnVzIGRvbG9yaWJ1cyBtb2xlc3RpYXMgdGVtcG9yYSBtYWduYW0gYXNzdW1lbmRhLg==",
		},
	)

	// 400 byte line must be followed by +:
	assertEqual(
		splitSaslResponse([]byte("slingamn\x00slingamn\x001111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111")),
		[]string{
			"c2xpbmdhbW4Ac2xpbmdhbW4AMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMQ==",
			"+",
		},
	)
}
