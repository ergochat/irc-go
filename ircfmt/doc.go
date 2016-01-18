// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

/*
Package ircfmt handles IRC formatting codes, escaping and unescaping.

This allows for a simpler representation of strings that contain colour codes,
bold codes, and such, without having to write and handle raw bytes when
assembling outgoing messages.

This package is in alpha.
*/
package ircfmt
