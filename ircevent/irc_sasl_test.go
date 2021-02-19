package ircevent

import (
	"crypto/tls"
	"fmt"
	"os"
	"testing"
)

const (
	serverEnvVar = "IRCEVENT_SERVER"
	saslEnvVar   = "IRCEVENT_SASL_LOGIN"
	saslPassVar  = "IRCEVENT_SASL_PASSWORD"
)

func getSaslCreds() (account, password string) {
	return os.Getenv(saslEnvVar), os.Getenv(saslPassVar)
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
	SASLLogin, SASLPassword := getSaslCreds()
	if useSASL {
		if SASLLogin == "" {
			t.Skip("Define SASLLogin and SASLPasword environment varables to test SASL")
		}
	}

	irccon := connForTesting("go-eventirc", "go-eventirc", true)
	irccon.Debug = true
	irccon.UseTLS = true
	if useSASL {
		irccon.UseSASL = true
		irccon.SASLLogin = SASLLogin
		irccon.SASLPassword = SASLPassword
	}
	irccon.RequestCaps = caps
	irccon.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	irccon.AddCallback("001", func(e Event) { irccon.Join("#go-eventirc") })

	irccon.AddCallback("366", func(e Event) {
		irccon.Privmsg("#go-eventirc", "Test Message SASL\n")
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
