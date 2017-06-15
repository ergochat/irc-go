# GoshuIRC-Go

This is a bunch of self-contained packages that help with IRC development in Go. The package splits themselves are fairly similar to how things are split up in the [original Python version](https://github.com/goshuirc/irc).

---

[![GoDoc](https://godoc.org/github.com/goshuirc/irc-go?status.svg)](https://godoc.org/github.com/goshuirc/irc-go)
[![Build Status](https://travis-ci.org/goshuirc/irc-go.svg?branch=master)](https://travis-ci.org/goshuirc/irc-go)
[![Coverage Status](https://coveralls.io/repos/goshuirc/irc-go/badge.svg?branch=master&service=github)](https://coveralls.io/github/goshuirc/irc-go?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/goshuirc/irc-go)](https://goreportcard.com/report/github.com/goshuirc/irc-go)

---

I'm aiming for this to become a client library all of its own. The best path to that is writing a bunch of useful, testable, self-contained packages that I'm able to wire together!

These packages are still in their early stages. Specifically, they're probably not as well-optimised as we'd like, and the interfaces exposed by them may not be final. For specific package details, view the documentation of that package.

An example bot that uses these packages can be found [here](https://gist.github.com/DanielOaks/cbbc957e8dba39f59d9e).

Packages:

* [**gircclient**](https://godoc.org/github.com/goshuirc/irc-go/client): Very work-in-progress client library.
* [**ircfmt**](https://godoc.org/github.com/goshuirc/irc-go/ircfmt): IRC format codes handling, escaping and unescaping.
* [**ircmap**](https://godoc.org/github.com/goshuirc/irc-go/ircmap): IRC string casefolding.
* [**ircmatch**](https://godoc.org/github.com/goshuirc/irc-go/ircmatch): IRC string matching (mostly just a globbing engine).
* [**ircmsg**](https://godoc.org/github.com/goshuirc/irc-go/ircmsg): IRC message handling, raw line parsing and creation.
* [**ircutils**](https://godoc.org/github.com/goshuirc/irc-go/ircutils): Useful utility functions and classes that don't fit into their own packages.

---

Also check out the eventmgr library [here](https://godoc.org/github.com/goshuirc/eventmgr), which helps with event attaching and dispatching.
