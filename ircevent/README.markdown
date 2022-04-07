Description
-----------

This is an event-based IRC client library. It is a fork of [thoj/go-ircevent](https://github.com/thoj-ircevent).

Features
--------
* Event-based: register callbacks for IRC commands
* Handles reconnections
* Supports SASL
* Supports requesting [IRCv3 capabilities](https://ircv3.net/specs/core/capability-negotiation)
* Advanced IRCv3 support, including [batch](https://ircv3.net/specs/extensions/batch) and [labeled-response](https://ircv3.net/specs/extensions/labeled-response)

Example
-------
See [examples/simple.go](examples/simple.go) for a working example, but this illustrates the API:

```go
irc := ircevent.Connection{
	Server:      "testnet.ergo.chat:6697",
	UseTLS:      true,
	Nick:        "ircevent-test",
	Debug:       true,
	RequestCaps: []string{"server-time", "message-tags"},
}

irc.AddConnectCallback(func(e ircmsg.Message) { irc.Join("#ircevent-test") })

irc.AddCallback("PRIVMSG", func(event ircmsg.Message) {
	// event.Source is the source;
	// event.Params[0] is the target (the channel or nickname the message was sent to)
	// and event.Params[1] is the message itself
});

err := irc.Connect()
if err != nil {
	log.Fatal(err)
}
irc.Loop()
```

The read loop executes all callbacks in serial on a single goroutine, respecting
the order in which messages are received from the server. All callbacks must
complete before the next message can be processed; if your callback needs to
trigger a long-running task, you should spin off a new goroutine for it.

Commands
--------
These methods can be used from inside callbacks, or externally:

	irc.Send(command, params...)
	irc.SendWithTags(tags, command, params...)
	irc.Join(channel)
	irc.Privmsg(target, message)
	irc.Privmsgf(target, formatString, params...)

The `ircevent.Connection` object is synchronized internally, so these methods
can be run from any goroutine without external locking.
