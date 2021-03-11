Description
-----------

This is an event-based IRC client library. It is a fork of [thoj/go-ircevent](https://github.com/thoj-ircevent).

Features
--------
* Event-based: register callbacks for IRC commands
* Handles reconnections
* Supports SASL
* Supports requesting [IRCv3 capabilities](https://ircv3.net/specs/core/capability-negotiation)

Example
-------
See [examples/simple.go](examples/simple.go) for a working example, but this illustrates the API:

```go
irc := ircevent.Connection{
	Server:      "testnet.oragono.io:6697",
	UseTLS:      true,
	Nick:        "ircevent-test",
	Debug:       true,
	RequestCaps: []string{"server-time", "message-tags"},
}

irc.AddCallback("001", func(e ircmsg.Message) { irc.Join("#ircevent-test") })

irc.AddCallback("PRIVMSG", func(event ircmsg.Message) {
	// event.Prefix is the source;
	// event.Params[0] is the target (the channel or nickname the message was sent to)
	// and event.Params[1] is the message itself
});

err := irc.Connect()
if err != nil {
	log.Fatal(err)
}
irc.Loop()
```

The read loop will wait for all callbacks to complete before moving on
to the next message. If your callback needs to trigger a long-running task,
you should spin off a new goroutine for it.

Commands
--------
These commands can be used from inside callbacks, or externally:

	irc.Connect("irc.someserver.com:6667") //Connect to server
	irc.Send(command, params...)
	irc.SendWithTags(tags, command, params...)
	irc.Join(channel)
	irc.Privmsg(target, message)
	irc.Privmsgf(target, formatString, params...)
