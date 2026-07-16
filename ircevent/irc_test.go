package ircevent

import (
	"crypto/tls"
	"math/rand"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ergochat/irc-go/ircmsg"
)

const channel = "#go-eventirc-test"
const dict = "abcdefghijklmnopqrstuvwxyz"

// Spammy
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

func mockEvent(command string) ircmsg.Message {
	return ircmsg.MakeMessage(nil, ":server.name", command)
}

func TestRemoveCallback(t *testing.T) {
	irccon := connForTesting("go-eventirc", "go-eventirc", false)
	debugTest(irccon)

	done := make(chan int, 10)

	irccon.AddCallback("TEST", func(e ircmsg.Message) { done <- 1 })
	id := irccon.AddCallback("TEST", func(e ircmsg.Message) { done <- 2 })
	irccon.AddCallback("TEST", func(e ircmsg.Message) { done <- 3 })

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

func TestClearCallback(t *testing.T) {
	irccon := connForTesting("go-eventirc", "go-eventirc", false)
	debugTest(irccon)

	done := make(chan int, 10)

	irccon.AddCallback("TEST", func(e ircmsg.Message) { done <- 0 })
	irccon.AddCallback("TEST", func(e ircmsg.Message) { done <- 1 })
	irccon.ClearCallback("TEST")
	irccon.AddCallback("TEST", func(e ircmsg.Message) { done <- 2 })
	irccon.AddCallback("TEST", func(e ircmsg.Message) { done <- 3 })

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

func TestIRCMaxMsgByteLen(t *testing.T) {
	ircnick1 := randStr(8)
	irccon := connForTesting(ircnick1, "go-eventirc", false)
	debugTest(irccon)
	irccon.FetchUserHost = true
	gotUserhost := make(chan struct{})
	done := make(chan struct{})
	irccon.AddCallback(RPL_WHOREPLY, func(e ircmsg.Message) {
		// hack for synchronization, wait until RPL_WHOREPLY was received and processed
		// (otherwise we're just testing the fallback value)
		close(gotUserhost)
	})
	expectedFail := false
	irccon.AddCallback(ERR_INPUTTOOLONG, func(e ircmsg.Message) {
		if e.Params[0] == ircnick1 {
			if !expectedFail {
				t.Errorf("ERR_INPUTTOOLONG: %v", e.Params[1])
			}
			done <- struct{}{}
		}
	})
	var rcvdMsg string
	irccon.AddCallback("PRIVMSG", func(e ircmsg.Message) {
		if e.Nick() == ircnick1 {
			rcvdMsg = e.Params[1]
			done <- struct{}{}
		}
	})
	err := irccon.Connect()
	if err != nil {
		t.Log(err.Error())
		t.Errorf("Can't connect to testing ircd.")
	}
	<-gotUserhost
	// now MaxMsgByteLen should return the real upper bound
	maxMsgByteLen := irccon.MaxMsgByteLen(ircnick1)
	msg := randStr(maxMsgByteLen)
	err = irccon.Privmsg(ircnick1, msg)
	if err != nil {
		t.Errorf("Unable to send privmsg: %v", err)
	}
	<-done
	if msg != rcvdMsg {
		t.Errorf("Messages do not match: sent: '%v', received: '%v'", msg, rcvdMsg)
	}

	// test that the bound is sharp by adding one byte to the message
	expectedFail = true
	msg += "!"
	rcvdMsg = ""
	err = irccon.Privmsg(ircnick1, msg)
	if err != nil {
		t.Errorf("Unable to send privmsg: %v", err)
	}
	<-done
	// message should either be relayed with truncation, or rejected,
	// depending on implementation
	if msg == rcvdMsg {
		t.Errorf("Successfully relayed message over MaxMsgByteLen() bytes: %d", len(msg))
	}
	irccon.Quit()
}

func TestConnection(t *testing.T) {
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

	irccon1.AddCallback("001", func(e ircmsg.Message) { irccon1.Join(channel) })
	irccon2.AddCallback("001", func(e ircmsg.Message) { irccon2.Join(channel) })
	irccon1.AddCallback("366", func(e ircmsg.Message) {
		go func(e ircmsg.Message) {
			tick := time.NewTicker(1 * time.Second)
			i := 10
			for {
				select {
				case <-tick.C:
					irccon1.Privmsgf(channel, "%s", teststr)
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

	irccon2.AddCallback("366", func(e ircmsg.Message) {
		ircnick2 = randStr(8)
		irccon2.SetNick(ircnick2)
	})

	irccon2.AddCallback("PRIVMSG", func(e ircmsg.Message) {
		if e.Params[1] == teststr {
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

	irccon2.AddCallback("NICK", func(e ircmsg.Message) {
		if !(e.Nick() == ircnick2orig && e.Params[0] == ircnick2) {
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
	if useSASL {
		setSaslTestCreds(irccon, t)
	}
	debugTest(irccon)

	connects := 0
	irccon.AddCallback("001", func(e ircmsg.Message) { irccon.Join(channel) })

	irccon.AddCallback("366", func(e ircmsg.Message) {
		connects += 1
		if connects > 2 {
			irccon.Privmsgf(channel, "Connection nr %d (test done)", connects)
			go irccon.Quit()
		} else {
			irccon.Privmsgf(channel, "Connection nr %d", connects)
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
	irccon.AddCallback("001", func(e ircmsg.Message) { irccon.Join(channel) })

	irccon.AddCallback("366", func(e ircmsg.Message) {
		irccon.Privmsg(channel, "Test Message from SSL")
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
	irccon1.AddCallback("002", func(e ircmsg.Message) { n1 <- irccon1.CurrentNick() })
	irccon2.AddCallback("002", func(e ircmsg.Message) { n2 <- irccon2.CurrentNick() })

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

func TestConnectionCallbacks(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	ircnick := randStr(8)
	irccon1 := connForTesting(ircnick, "IRCTest1", false)
	debugTest(irccon1)
	resultChan := make(chan map[string]string, 1)
	disconnectCalled := false
	irccon1.AddConnectCallback(func(e ircmsg.Message) {
		resultChan <- irccon1.ISupport()
	})
	irccon1.AddDisconnectCallback(func(e ircmsg.Message) {
		disconnectCalled = true
	})
	err := irccon1.Connect()
	if err != nil {
		panic(err)
	}
	loopExited := make(chan empty)
	go func() {
		irccon1.Loop()
		close(loopExited)
	}()
	isupport := <-resultChan
	if casemapping := isupport["CASEMAPPING"]; casemapping == "" {
		t.Errorf("casemapping not detected in 005 RPL_ISUPPORT output; this is unheard of")
	}
	assertEqual(disconnectCalled, false)
	irccon1.Quit()
	<-loopExited
	assertEqual(disconnectCalled, true)
}

func TestCAPHandling(t *testing.T) {
	var testCases = []struct {
		name   string
		caps   []string
		result []string
	}{
		{
			name:   "no caps",
			caps:   nil,
			result: nil,
		},
		{
			name:   "one invalid cap",
			caps:   []string{"ergo.chat/nonexistent"},
			result: nil,
		},
		{
			name:   "one valid cap",
			caps:   []string{"message-tags"},
			result: []string{"message-tags"},
		},
		{
			name:   "one valid cap, one invalid",
			caps:   []string{"ergo.chat/nonexistent", "message-tags"},
			result: []string{"message-tags"},
		},
		{
			name:   "multiple caps, one invalid",
			caps:   []string{"server-time", "batch", "echo-message", "labeled-response", "account-tag", "ergo.chat/nonexistent", "message-tags"},
			result: []string{"account-tag", "batch", "echo-message", "labeled-response", "message-tags", "server-time"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rand.Seed(time.Now().UnixNano())
			ircnick := randStr(8)
			irccon1 := connForTesting(ircnick, "IRCTest1", false)
			irccon1.RequestCaps = tc.caps
			debugTest(irccon1)
			resultChan := make(chan map[string]string, 1)
			disconnectCalled := false
			irccon1.AddConnectCallback(func(e ircmsg.Message) {
				resultChan <- irccon1.ISupport()
			})
			irccon1.AddDisconnectCallback(func(e ircmsg.Message) {
				disconnectCalled = true
			})
			err := irccon1.Connect()
			if err != nil {
				panic(err)
			}
			loopExited := make(chan empty)
			go func() {
				irccon1.Loop()
				close(loopExited)
			}()
			<-resultChan
			// should successfully req message-tags, ignoring the nonexistent cap
			capsAcked := sortAckedCaps(irccon1.capsAcked)
			assertEqual(capsAcked, tc.result)
			assertEqual(disconnectCalled, false)
			irccon1.Quit()
			<-loopExited
			assertEqual(disconnectCalled, true)
		})
	}
}

func sortAckedCaps(caps map[string]string) (result []string) {
	if len(caps) == 0 {
		return nil
	}
	result = make([]string, 0, len(caps))
	for c := range caps {
		result = append(result, c)
	}
	sort.Strings(result)
	return result
}

func mustParse(line string) ircmsg.Message {
	msg, err := ircmsg.ParseLine(line)
	if err != nil {
		panic(err)
	}
	return msg
}

func TestGetReplyTarget(t *testing.T) {
	irc := Connection{}
	assertEqual(irc.GetReplyTarget(mustParse(":shivaram!~u@vjsnqp44px9sc.irc PRIVMSG #ergo :hi")), "#ergo")
	assertEqual(irc.GetReplyTarget(mustParse(":shivaram!~u@vjsnqp44px9sc.irc PRIVMSG titlebot :hi")), "shivaram")
	irc.isupport = map[string]string{
		"CHANTYPES": "#&",
	}
	assertEqual(irc.GetReplyTarget(mustParse(":shivaram!~u@vjsnqp44px9sc.irc PRIVMSG #ergo :hi")), "#ergo")
	assertEqual(irc.GetReplyTarget(mustParse(":shivaram!~u@vjsnqp44px9sc.irc PRIVMSG &ergo :hi")), "&ergo")
	assertEqual(irc.GetReplyTarget(mustParse(":shivaram!~u@vjsnqp44px9sc.irc PRIVMSG titlebot :hi")), "shivaram")
	assertEqual(irc.GetReplyTarget(mustParse(":irc.ergo.chat NOTICE titlebot :Server is shutting down")), "")

	// no source but it's a channel message
	assertEqual(irc.GetReplyTarget(mustParse("PRIVMSG #ergo :hi")), "#ergo")
	// no source but it's a DM (no way to reply)
	assertEqual(irc.GetReplyTarget(mustParse("PRIVMSG titlebot :hi")), "")
	// invalid messages
	assertEqual(irc.GetReplyTarget(mustParse(":shivaram!~u@vjsnqp44px9sc.irc PRIVMSG")), "")
	assertEqual(irc.GetReplyTarget(mustParse("PRIVMSG")), "")
	assertEqual(irc.GetReplyTarget(mustParse("PRIVMSG :")), "")
	// not a PRIVMSG
	assertEqual(irc.GetReplyTarget(mustParse(":testnet.ergo.chat 371 shivaram :This is Ergo version 2.13.0.")), "")
}

func TestAddRawCallback(t *testing.T) {
	ircnick1 := randStr(8)
	irccon := connForTesting(ircnick1, "go-eventirc", false)
	debugTest(irccon)
	commandsSeen := make(map[string]bool)
	var linesSeen []string
	id := irccon.AddRawCallback(func(raw string, msg ircmsg.Message, err error) {
		if err == nil {
			commandsSeen[msg.Command] = true

			msg2, err2 := ircmsg.ParseLine(raw)
			assertEqual(err2, nil)
			assertEqual(msg.Command, msg2.Command)
			assertEqual(msg.Params, msg2.Params)
		} else {
			t.Errorf("bad line from IRC server: `%s`: %v", raw, err)
		}
		linesSeen = append(linesSeen, raw)
	})
	irccon.AddConnectCallback(func(e ircmsg.Message) {
		irccon.Join("#ircevent-test")
	})
	irccon.AddCallback("JOIN", func(e ircmsg.Message) {
		irccon.RemoveCallback(id)
	})
	seenRplNamreply := false
	irccon.AddCallback(RPL_NAMREPLY, func(e ircmsg.Message) {
		seenRplNamreply = true
		irccon.Quit()
	})
	err := irccon.Connect()
	if err != nil {
		t.Log(err.Error())
		t.Errorf("Can't connect to testing ircd.")
	}
	// wait for QUIT to be processed
	irccon.Loop()

	assertEqual(commandsSeen["001"], true)
	assertEqual(commandsSeen["002"], true)
	assertEqual(commandsSeen["003"], true)
	assertEqual(commandsSeen["004"], true)
	assertEqual(commandsSeen["005"], true)
	// we removed the raw callback during the JOIN handler;
	// whether we observe JOIN itself is undefined by the API,
	// but we should not observe any of the other JOIN-related commands
	assertEqual(commandsSeen[RPL_NAMREPLY], false)
	assertEqual(commandsSeen[RPL_TOPIC], false)
	assertEqual(commandsSeen[RPL_TOPICTIME], false)
	assertEqual(commandsSeen["QUIT"], false)
	// RPL_NAMREPLY should have been observed by the normal command handler:
	assertEqual(seenRplNamreply, true)

	if len(linesSeen) < 10 {
		t.Errorf("insufficient raw lines observed: %#v", linesSeen)
	}
}
