# gIRC-go

This is a bunch of self-contained packages that help with IRC development in Go. The package splits themselves are fairly similar to how things are split up in the [original Python version](https://github.com/DanielOaks/girc).

---

[![Build Status](https://travis-ci.org/DanielOaks/girc-go.svg?branch=master)](https://travis-ci.org/DanielOaks/girc-go) [![Coverage Status](https://coveralls.io/repos/DanielOaks/girc-go/badge.svg?branch=master&service=github)](https://coveralls.io/github/DanielOaks/girc-go?branch=master)

---

I'm aiming for this to become a client library all of its own. The best path to that is writing a bunch of useful, testable, self-contained packages that I'm able to wire together!

These packages are still in their early stages. Specifically, they're probably not as well-optimised as we'd like.

An example bot that uses these packages can be found [here](https://gist.github.com/DanielOaks/cbbc957e8dba39f59d9e).

Packages:

* [**eventmgr**](https://godoc.org/github.com/DanielOaks/girc-go/eventmgr): Event attaching and dispatching.
* [**ircfmt**](https://godoc.org/github.com/DanielOaks/girc-go/ircfmt): IRC format codes handling, escaping and unescaping.
* [**ircmsg**](https://godoc.org/github.com/DanielOaks/girc-go/ircmsg): IRC message handling, raw line parsing and creation.
