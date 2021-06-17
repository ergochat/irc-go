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
