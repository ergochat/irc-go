package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/ergochat/irc-go/ircevent"
	"github.com/ergochat/irc-go/ircmsg"
)

func getenv(key, defaultValue string) (value string) {
	value = os.Getenv(key)
	if value == "" {
		value = defaultValue
	}
	return
}

func equalCaseInsensitive(s1, s2 string) bool {
	return s1 == s2 || strings.ToLower(s1) == strings.ToLower(s2)
}

func main() {
	nick := getenv("IRCEVENT_NICK", "robot")
	server := getenv("IRCEVENT_SERVER", "testnet.oragono.io:6697")
	channel := getenv("IRCEVENT_CHANNEL", "#ircevent-test")
	saslLogin := os.Getenv("IRCEVENT_SASL_LOGIN")
	saslPassword := os.Getenv("IRCEVENT_SASL_PASSWORD")

	irc := ircevent.Connection{
		Server:       server,
		Nick:         nick,
		Debug:        true,
		UseTLS:       true,
		TLSConfig:    &tls.Config{InsecureSkipVerify: true},
		RequestCaps:  []string{"server-time", "message-tags"},
		SASLLogin:    saslLogin, // SASL will be enabled automatically if these are set
		SASLPassword: saslPassword,
	}

	irc.AddConnectCallback(func(e ircmsg.Message) {
		// attempt to set the BOT mode on ourself:
		if botMode := irc.ISupport()["BOT"]; botMode != "" {
			irc.Send("MODE", irc.CurrentNick(), "+"+botMode)
		}
		irc.Join(channel)
	})
	irc.AddCallback("JOIN", func(e ircmsg.Message) {
		irc.Send("WHO", channel)
	})
	// implicitly synchronized by the callback handlers,
	// if this state were accessed from another goroutine we could use a mutex
	var channelNUHs []ircmsg.NUH
	var whoComplete bool = false
	irc.AddCallback(ircevent.RPL_WHOREPLY, func(e ircmsg.Message) {
		if !equalCaseInsensitive(e.Params[1], channel) {
			return // drop responses for other channels
		}
		nuh := ircmsg.NUH{
			Name: e.Params[5],
			User: e.Params[2],
			Host: e.Params[3],
		}
		channelNUHs = append(channelNUHs, nuh)
	})
	irc.AddCallback(ircevent.RPL_ENDOFWHO, func(e ircmsg.Message) {
		if !equalCaseInsensitive(e.Params[1], channel) {
			return
		}
		whoComplete = true
		if whoComplete {
			fmt.Printf("WHO response complete, got %d channel members:\n", len(channelNUHs))
			for i, nuh := range channelNUHs {
				fmt.Printf("%4d: %s!%s@%s\n", i, nuh.Name, nuh.User, nuh.Host)
			}
		}
	})
	irc.AddCallback("PRIVMSG", func(e ircmsg.Message) {
		if len(e.Params) < 2 {
			return
		}
		text := e.Params[1]
		if strings.HasPrefix(text, nick) {
			irc.Privmsg(e.Params[0], "don't @ me, fleshbag")
		} else if text == "xyzzy" {
			// this causes the server to disconnect us and the program to exit
			irc.Quit()
		} else if text == "plugh" {
			// this causes the server to disconnect us, but the client will reconnect
			irc.Send("QUIT", "I'LL BE BACK")
		} else if text == "wwssadadba" {
			// this line intentionally panics; the client will recover from it
			irc.Privmsg(e.Params[0], e.Params[2])
		}
	})
	// example client-to-client extension via message-tags:
	// have the bot maintain a running sum of integers
	var sum int64 // doesn't need synchronization as long as it's only visible from a single callback
	irc.AddCallback("TAGMSG", func(e ircmsg.Message) {
		_, tv := e.GetTag("+summand")
		if v, err := strconv.ParseInt(tv, 10, 64); err == nil {
			sum += v
			irc.SendWithTags(map[string]string{"+sum": strconv.FormatInt(sum, 10)}, "TAGMSG", e.Params[0])
		}
	})
	err := irc.Connect()
	if err != nil {
		log.Fatal(err)
	}
	irc.Loop()
}
