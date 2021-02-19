package ircevent

import (
	"crypto/tls"
	"math/rand"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/goshuirc/irc-go/ircmsg"
)

const channel = "#go-eventirc-test"
const dict = "abcdefghijklmnopqrstuvwxyz"

//Spammy
const verbose_tests = false
const debug_tests = true

func connForTesting(nick, user string, tls bool) *Connection {
	irc := &Connection{
		Nick:   nick,
		User:   user,
		Server: getServer(tls),
	}
	return irc
}

func mockEvent(command string) ircmsg.IRCMessage {
	return ircmsg.MakeMessage(nil, ":server.name", command)
}

func TestRemoveCallback(t *testing.T) {
	irccon := connForTesting("go-eventirc", "go-eventirc", false)
	debugTest(irccon)

	done := make(chan int, 10)

	irccon.AddCallback("TEST", func(e Event) { done <- 1 })
	id := irccon.AddCallback("TEST", func(e Event) { done <- 2 })
	irccon.AddCallback("TEST", func(e Event) { done <- 3 })

	// Should remove callback at index 1
	irccon.RemoveCallback(id)

	irccon.runCallbacks(mockEvent("TEST"))

	var results []int

	results = append(results, <-done)
	results = append(results, <-done)

	if !compareResults(results, 1, 3) {
		t.Error("Callback 2 not removed")
	}
}

func TestWildcardCallback(t *testing.T) {
	irccon := connForTesting("go-eventirc", "go-eventirc", false)
	debugTest(irccon)

	done := make(chan int, 10)

	irccon.AddCallback("TEST", func(e Event) { done <- 1 })
	irccon.AddCallback("*", func(e Event) { done <- 2 })

	irccon.runCallbacks(mockEvent("TEST"))

	var results []int

	results = append(results, <-done)
	results = append(results, <-done)

	if !compareResults(results, 1, 2) {
		t.Error("Wildcard callback not called")
	}
}

func TestClearCallback(t *testing.T) {
	irccon := connForTesting("go-eventirc", "go-eventirc", false)
	debugTest(irccon)

	done := make(chan int, 10)

	irccon.AddCallback("TEST", func(e Event) { done <- 0 })
	irccon.AddCallback("TEST", func(e Event) { done <- 1 })
	irccon.ClearCallback("TEST")
	irccon.AddCallback("TEST", func(e Event) { done <- 2 })
	irccon.AddCallback("TEST", func(e Event) { done <- 3 })

	irccon.runCallbacks(mockEvent("TEST"))

	var results []int

	results = append(results, <-done)
	results = append(results, <-done)

	if !compareResults(results, 2, 3) {
		t.Error("Callbacks not cleared")
	}
}

func TestIRCemptyNick(t *testing.T) {
	irccon := connForTesting("", "go-eventirc", false)
	irccon = nil
	if irccon != nil {
		t.Error("empty nick didn't result in error")
		t.Fail()
	}
}

func TestConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	rand.Seed(time.Now().UnixNano())
	ircnick1 := randStr(8)
	ircnick2 := randStr(8)
	ircnick2orig := ircnick2
	irccon1 := connForTesting(ircnick1, "IRCTest1", false)
	debugTest(irccon1)

	irccon2 := connForTesting(ircnick2, "IRCTest2", false)
	debugTest(irccon2)

	teststr := randStr(20)
	testmsgok := make(chan bool, 1)

	irccon1.AddCallback("001", func(e Event) { irccon1.Join(channel) })
	irccon2.AddCallback("001", func(e Event) { irccon2.Join(channel) })
	irccon1.AddCallback("366", func(e Event) {
		go func(e Event) {
			tick := time.NewTicker(1 * time.Second)
			i := 10
			for {
				select {
				case <-tick.C:
					irccon1.Privmsgf(channel, "%s\n", teststr)
					if i == 0 {
						t.Errorf("Timeout while wating for test message from the other thread.")
						return
					}

				case <-testmsgok:
					tick.Stop()
					irccon1.Quit()
					return
				}
				i -= 1
			}
		}(e)
	})

	irccon2.AddCallback("366", func(e Event) {
		ircnick2 = randStr(8)
		irccon2.SetNick(ircnick2)
	})

	irccon2.AddCallback("PRIVMSG", func(e Event) {
		if e.Message() == teststr {
			if e.Nick() == ircnick1 {
				testmsgok <- true
				irccon2.Quit()
			} else {
				t.Errorf("Test message came from an unexpected nickname")
			}
		} else {
			//this may fail if there are other incoming messages, unlikely.
			t.Errorf("Test message mismatch")
		}
	})

	irccon2.AddCallback("NICK", func(e Event) {
		if !(e.Nick() == ircnick2orig && e.Message() == ircnick2) {
			t.Errorf("Nick change did not work!")
		}
	})

	err := irccon1.Connect()
	if err != nil {
		t.Log(err.Error())
		t.Errorf("Can't connect to freenode.")
	}
	err = irccon2.Connect()
	if err != nil {
		t.Log(err.Error())
		t.Errorf("Can't connect to freenode.")
	}

	go irccon2.Loop()
	irccon1.Loop()
}

func runReconnectTest(useSASL bool, t *testing.T) {
	ircnick1 := randStr(8)
	irccon := connForTesting(ircnick1, "IRCTestRe", false)
	irccon.ReconnectFreq = time.Second * 1
	saslLogin, saslPassword := getSaslCreds()
	if useSASL {
		if saslLogin == "" {
			t.Skip("Define SASL environment varables to test SASL")
		} else {
			irccon.UseSASL = true
			irccon.SASLLogin = saslLogin
			irccon.SASLPassword = saslPassword
		}
	}
	debugTest(irccon)

	connects := 0
	irccon.AddCallback("001", func(e Event) { irccon.Join(channel) })

	irccon.AddCallback("366", func(e Event) {
		connects += 1
		if connects > 2 {
			irccon.Privmsgf(channel, "Connection nr %d (test done)\n", connects)
			go irccon.Quit()
		} else {
			irccon.Privmsgf(channel, "Connection nr %d\n", connects)
			// XXX: wait for the message to actually send before we hang up
			// (can this be avoided?)
			time.Sleep(100 * time.Millisecond)
			go irccon.Reconnect()
		}
	})

	err := irccon.Connect()
	if err != nil {
		t.Log(err.Error())
		t.Errorf("Can't connect to freenode.")
	}

	irccon.Loop()
	if connects != 3 {
		t.Errorf("Reconnect test failed. Connects = %d", connects)
	}
}

func TestReconnect(t *testing.T) {
	runReconnectTest(false, t)
}

func TestReconnectWithSASL(t *testing.T) {
	runReconnectTest(true, t)
}

func TestConnectionSSL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	ircnick1 := randStr(8)
	irccon := connForTesting(ircnick1, "IRCTestSSL", true)
	debugTest(irccon)
	irccon.UseTLS = true
	irccon.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	irccon.AddCallback("001", func(e Event) { irccon.Join(channel) })

	irccon.AddCallback("366", func(e Event) {
		irccon.Privmsg(channel, "Test Message from SSL\n")
		irccon.Quit()
	})

	err := irccon.Connect()
	if err != nil {
		t.Log(err.Error())
		t.Errorf("Can't connect to freenode.")
	}

	irccon.Loop()
}

// Helper Functions
func randStr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = dict[rand.Intn(len(dict))]
	}
	return string(b)
}

func debugTest(irccon *Connection) *Connection {
	irccon.Debug = debug_tests
	return irccon
}

func compareResults(received []int, desired ...int) bool {
	if len(desired) != len(received) {
		return false
	}
	sort.IntSlice(desired).Sort()
	sort.IntSlice(received).Sort()
	for i := 0; i < len(desired); i++ {
		if desired[i] != received[i] {
			return false
		}
	}
	return true
}

func TestConnectionNickInUse(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	ircnick := randStr(8)
	irccon1 := connForTesting(ircnick, "IRCTest1", false)

	debugTest(irccon1)

	irccon2 := connForTesting(ircnick, "IRCTest2", false)
	debugTest(irccon2)

	n1 := make(chan string, 1)
	n2 := make(chan string, 1)

	// check the actual nick after 001 is processed
	irccon1.AddCallback("002", func(e Event) { n1 <- irccon1.CurrentNick() })
	irccon2.AddCallback("002", func(e Event) { n2 <- irccon2.CurrentNick() })

	err := irccon1.Connect()
	if err != nil {
		panic(err)
	}
	err = irccon2.Connect()
	if err != nil {
		panic(err)
	}

	go irccon2.Loop()
	go irccon1.Loop()
	nick1 := <-n1
	nick2 := <-n2
	irccon1.Quit()
	irccon2.Quit()
	// we should have gotten two different nicks, one a prefix of the other
	if nick1 == ircnick && len(nick1) < len(nick2) && strings.HasPrefix(nick2, nick1) {
		return
	}
	if nick2 == ircnick && len(nick2) < len(nick1) && strings.HasPrefix(nick1, nick2) {
		return
	}
	t.Errorf("expected %s and a suffixed version, got %s and %s", ircnick, nick1, nick2)
}
