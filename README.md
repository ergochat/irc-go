# ergochat/irc-go

These are libraries to help in writing IRC clients and servers in Go, prioritizing correctness, safety, and [IRCv3 support](https://ircv3.net/). They are not fully API-stable, but we expect any API breaks to be modest in scope.

---

[![GoDoc](https://godoc.org/github.com/goshuirc/irc-go?status.svg)](https://godoc.org/github.com/goshuirc/irc-go)
[![Build Status](https://travis-ci.org/goshuirc/irc-go.svg?branch=master)](https://travis-ci.org/goshuirc/irc-go)
[![Coverage Status](https://coveralls.io/repos/goshuirc/irc-go/badge.svg?branch=master&service=github)](https://coveralls.io/github/goshuirc/irc-go?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/goshuirc/irc-go)](https://goreportcard.com/report/github.com/goshuirc/irc-go)

---

Packages:

* [**ircmsg**](https://godoc.org/github.com/goshuirc/irc-go/ircmsg): IRC message handling, raw line parsing and creation.
* [**ircreader**](https://godoc.org/github.com/goshuirc/irc-go/ircreader): Optimized reader for \n-terminated lines, with an expanding but bounded buffer.
* [**ircevent**](https://godoc.org/github.com/goshuirc/irc-go/ircevent): IRC client library (fork of [thoj/go-ircevent](https://github.com/thoj/go-ircevent)).
* [**ircfmt**](https://godoc.org/github.com/goshuirc/irc-go/ircfmt): IRC format codes handling, escaping and unescaping.
* [**ircutils**](https://godoc.org/github.com/goshuirc/irc-go/ircutils): Useful utility functions and classes that don't fit into their own packages.

For a relatively complete example of the library's use, see [slingamn/titlebot](https://github.com/slingamn/titlebot).
