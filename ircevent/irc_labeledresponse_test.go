package ircevent

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"testing"
)

const (
	multilineName   = "draft/multiline"
	chathistoryName = "draft/chathistory"
	concatTag       = "draft/multiline-concat"
	playbackCap     = "draft/event-playback"
)

func TestLabeledResponse(t *testing.T) {
	irccon := connForTesting("go-eventirc", "go-eventirc", false)
	irccon.Debug = true
	irccon.RequestCaps = []string{"message-tags", "batch", "labeled-response"}
	irccon.RealName = "ecf61da38b58"
	results := make(map[string]string)
	irccon.AddConnectCallback(func(e Event) {
		irccon.SendWithLabel(func(batch *Batch) {
			if batch == nil {
				return
			}
			for _, line := range batch.Items {
				results[line.Command] = line.Params[len(line.Params)-1]
			}
			irccon.Quit()
		}, nil, "WHOIS", irccon.CurrentNick())
	})
	err := irccon.Connect()
	if err != nil {
		t.Fatalf("labeled response connection failed: %s", err)
	}
	irccon.Loop()

	// RPL_WHOISUSER, last param is the realname
	assertEqual(results["311"], "ecf61da38b58")
	if _, ok := results["379"]; !ok {
		t.Errorf("Expected 379 RPL_WHOISMODES in response, but not received")
	}
	assertEqual(len(irccon.batches), 0)
}

func TestLabeledResponseNoCaps(t *testing.T) {
	irccon := connForTesting("go-eventirc", "go-eventirc", false)
	irccon.Debug = true
	irccon.RequestCaps = []string{"message-tags"}
	irccon.RealName = "ecf61da38b58"

	err := irccon.Connect()
	if err != nil {
		t.Fatalf("labeled response connection failed: %s", err)
	}
	go irccon.Loop()

	results := make(map[string]string)
	err = irccon.SendWithLabel(func(batch *Batch) {
		if batch == nil {
			return
		}
		for _, line := range batch.Items {
			results[line.Command] = line.Params[len(line.Params)-1]
		}
		irccon.Quit()
	}, nil, "WHOIS", irccon.CurrentNick())
	if err != CapabilityNotNegotiated {
		t.Errorf("expected capability negotiation error, got %v", err)
	}
	assertEqual(len(irccon.batches), 0)
	irccon.Quit()
}

// test labeled single-line response, and labeled ACK
func TestLabeledResponseSingleResponse(t *testing.T) {
	irc := connForTesting("go-eventirc", "go-eventirc", false)
	irc.Debug = true
	irc.RequestCaps = []string{"message-tags", "batch", "labeled-response"}

	err := irc.Connect()
	if err != nil {
		t.Fatalf("labeled response connection failed: %s", err)
	}
	go irc.Loop()

	channel := fmt.Sprintf("#%s", randomString())
	irc.Join(channel)
	event := make(chan empty)
	err = irc.SendWithLabel(func(batch *Batch) {
		if !(batch != nil && batch.Command == "PONG" && batch.Params[len(batch.Params)-1] == "asdf") {
			t.Errorf("expected labeled PONG, got %#v", batch)
		}
		close(event)
	}, nil, "PING", "asdf")
	<-event

	// no-op JOIN will send labeled ACK
	event = make(chan empty)
	err = irc.SendWithLabel(func(batch *Batch) {
		if !(batch != nil && batch.Command == "ACK") {
			t.Errorf("expected labeled ACK, got %#v", batch)
		}
		close(event)
	}, nil, "JOIN", channel)
	<-event

	assertEqual(len(irc.batches), 0)
	irc.Quit()
}

func randomString() string {
	buf := make([]byte, 8)
	rand.Read(buf)
	return hex.EncodeToString(buf)
}

func TestNestedBatch(t *testing.T) {
	irc := connForTesting("go-eventirc", "go-eventirc", false)
	irc.Debug = true
	irc.RequestCaps = []string{"message-tags", "batch", "labeled-response", "server-time", multilineName, chathistoryName, playbackCap}
	channel := fmt.Sprintf("#%s", randomString())

	irc.AddConnectCallback(func(e Event) {
		irc.Join(channel)
		irc.Privmsg(channel, "hi")
		irc.Send("BATCH", "+123", "draft/multiline", channel)
		irc.SendWithTags(map[string]string{"batch": "123"}, "PRIVMSG", channel, "hello")
		irc.SendWithTags(map[string]string{"batch": "123"}, "PRIVMSG", channel, "")
		irc.SendWithTags(map[string]string{"batch": "123", concatTag: ""}, "PRIVMSG", channel, "how is ")
		irc.SendWithTags(map[string]string{"batch": "123"}, "PRIVMSG", channel, "everyone?")
		irc.Send("BATCH", "-123")
	})

	err := irc.Connect()
	if err != nil {
		t.Fatalf("labeled response connection failed: %s", err)
	}
	go irc.Loop()

	var historyBatch *Batch
	event := make(chan empty)
	irc.SendWithLabel(func(batch *Batch) {
		historyBatch = batch
		close(event)
	}, nil, "CHATHISTORY", "LATEST", channel, "*", "10")

	<-event
	assertEqual(len(irc.labelCallbacks), 0)

	if historyBatch == nil {
		t.Errorf("received nil history batch")
	}

	// history should contain the JOIN, the PRIVMSG, and the multiline batch as a single item
	if !(historyBatch.Command == "BATCH" && len(historyBatch.Items) == 3) {
		t.Errorf("chathistory must send a real batch, got %#v", historyBatch)
	}
	var privmsg, multiline *Batch
	for _, item := range historyBatch.Items {
		switch item.Command {
		case "PRIVMSG":
			privmsg = item
		case "BATCH":
			multiline = item
		}
	}
	if !(privmsg.Command == "PRIVMSG" && privmsg.Params[0] == channel && privmsg.Params[1] == "hi") {
		t.Errorf("expected echo of individual privmsg, got %#v", privmsg)
	}
	if !(multiline.Command == "BATCH" && len(multiline.Items) == 4 && multiline.Items[3].Command == "PRIVMSG" && multiline.Items[3].Params[1] == "everyone?") {
		t.Errorf("expected multiline in history, got %#v\n", multiline)
	}

	assertEqual(len(irc.batches), 0)
	irc.Quit()
}

func TestBatchHandlers(t *testing.T) {
	alice := connForTesting("alice", "go-eventirc", false)
	alice.Debug = true
	alice.RequestCaps = []string{"message-tags", "batch", "labeled-response", "server-time", "echo-message", multilineName, chathistoryName, playbackCap}
	channel := fmt.Sprintf("#%s", randomString())

	aliceUnderstandsBatches := true
	var aliceBatchCount, alicePrivmsgCount int
	alice.AddBatchCallback(func(batch *Batch) bool {
		if aliceUnderstandsBatches {
			aliceBatchCount++
			return true
		}
		return false
	})
	alice.AddCallback("PRIVMSG", func(e Event) {
		alicePrivmsgCount++
	})

	err := alice.Connect()
	if err != nil {
		t.Fatalf("labeled response connection failed: %s", err)
	}
	go alice.Loop()
	alice.Join(channel)
	synchronize(alice)

	bob := connForTesting("bob", "go-eventirc", false)
	bob.Debug = true
	bob.RequestCaps = []string{"message-tags", "batch", "labeled-response", "server-time", "echo-message", multilineName, chathistoryName, playbackCap}
	var buf bytes.Buffer
	bob.AddBatchCallback(func(b *Batch) bool {
		if !(len(b.Params) >= 3 && b.Params[1] == multilineName) {
			return false
		}
		for i, item := range b.Items {
			if item.Command == "PRIVMSG" {
				buf.WriteString(item.Params[1])
				if !(item.HasTag(concatTag) || i == len(b.Items)-1) {
					buf.WriteByte('\n')
				}
			}
		}
		return true
	})

	err = bob.Connect()
	if err != nil {
		t.Fatalf("labeled response connection failed: %s", err)
	}
	go bob.Loop()
	bob.Join(channel)
	synchronize(bob)

	sendMultiline := func() {
		alice.Send("BATCH", "+123", "draft/multiline", channel)
		alice.SendWithTags(map[string]string{"batch": "123"}, "PRIVMSG", channel, "hello")
		alice.SendWithTags(map[string]string{"batch": "123"}, "PRIVMSG", channel, "")
		alice.SendWithTags(map[string]string{"batch": "123", concatTag: ""}, "PRIVMSG", channel, "how is ")
		alice.SendWithTags(map[string]string{"batch": "123"}, "PRIVMSG", channel, "everyone?")
		alice.Send("BATCH", "-123")
		synchronize(alice)
	}
	multilineMessageValue := "hello\n\nhow is everyone?"

	sendMultiline()
	synchronize(alice)
	synchronize(bob)

	assertEqual(alicePrivmsgCount, 0)
	alicePrivmsgCount = 0
	assertEqual(aliceBatchCount, 1)
	aliceBatchCount = 0

	assertEqual(buf.String(), multilineMessageValue)
	buf.Reset()

	aliceUnderstandsBatches = false
	sendMultiline()
	synchronize(alice)
	synchronize(bob)

	// disabled alice's batch handler, she should see a flattened batch
	assertEqual(alicePrivmsgCount, 4)
	assertEqual(aliceBatchCount, 0)

	assertEqual(buf.String(), multilineMessageValue)

	assertEqual(len(alice.batches), 0)
	assertEqual(len(bob.batches), 0)
	alice.Quit()
	bob.Quit()
}

func synchronize(irc *Connection) {
	event := make(chan empty)
	irc.SendWithLabel(func(b *Batch) {
		close(event)
	}, nil, "PING", "!")
	<-event
}
