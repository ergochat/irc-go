package main

import (
	"log"
	"os"
	"strconv"

	"net/http"
	_ "net/http/pprof"

	"github.com/goshuirc/irc-go/ircevent"
	"github.com/goshuirc/irc-go/ircmsg"
)

/*
Flooding stress test (responds to its own echo messages in a loop);
don't run this against a real IRC server!
*/

func getenv(key, defaultValue string) (value string) {
	value = os.Getenv(key)
	if value == "" {
		value = defaultValue
	}
	return
}

func main() {
	ps := http.Server{
		Addr: getenv("IRCEVENT_PPROF_LISTENER", "localhost:6077"),
	}
	go func() {
		if err := ps.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	nick := getenv("IRCEVENT_NICK", "chatterbox")
	server := getenv("IRCEVENT_SERVER", "localhost:6667")
	channel := getenv("IRCEVENT_CHANNEL", "#ircevent-test")
	limit := 0
	if envLimit, err := strconv.Atoi(os.Getenv("IRCEVENT_LIMIT")); err == nil {
		limit = envLimit
	}

	irc := ircevent.Connection{
		Server:      server,
		Nick:        nick,
		RequestCaps: []string{"server-time", "echo-message"},
	}

	irc.AddCallback("001", func(e ircmsg.Message) { irc.Join(channel) })
	irc.AddCallback("JOIN", func(e ircmsg.Message) { irc.Privmsg(channel, "hi there friend!") })
	// echo whatever we get back
	count := 0
	irc.AddCallback("PRIVMSG", func(e ircmsg.Message) {
		if limit != 0 && count >= limit {
			irc.Quit()
		} else {
			irc.Privmsg(e.Params[0], e.Params[1])
			count++
		}
	})
	err := irc.Connect()
	if err != nil {
		log.Fatal(err)
	}
	irc.Loop()
}
