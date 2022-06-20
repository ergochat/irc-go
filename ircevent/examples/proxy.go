package main

import (
	"crypto/tls"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

	"golang.org/x/net/proxy"

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

func main() {
	nick := getenv("IRCEVENT_NICK", "robot")
	server := getenv("IRCEVENT_SERVER", "testnet.oragono.io:6697")
	channel := getenv("IRCEVENT_CHANNEL", "#ircevent-test")
	saslLogin := os.Getenv("IRCEVENT_SASL_LOGIN")
	saslPassword := os.Getenv("IRCEVENT_SASL_PASSWORD")

	proxyAddr := getenv("IRCEVENT_PROXY_URL", "socks5://192.168.1.100:8080")
	proxyUrl, err := url.Parse(proxyAddr)
	if err != nil {
		log.Fatalf("invalid proxy URL: %v\n", err)
	}
	proxyDialer, err := proxy.FromURL(proxyUrl, proxy.Direct)
	if err != nil {
		log.Fatalf("couldn't connect to proxy server: %v\n", err)
	}
	proxyContextDialer, ok := proxyDialer.(proxy.ContextDialer)
	if !ok {
		log.Fatalf("proxy dialer does not expose DialContext(): %v\n", proxyDialer)
	}

	irc := ircevent.Connection{
		Server:       server,
		DialContext:  proxyContextDialer.DialContext,
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
	irc.AddCallback("JOIN", func(e ircmsg.Message) {}) // TODO try to rejoin if we *don't* get this
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
	err = irc.Connect()
	if err != nil {
		log.Fatal(err)
	}
	irc.Loop()
}
