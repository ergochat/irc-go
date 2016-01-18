// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

/*
Package ircfmt handles IRC formatting codes, escaping and unescaping.

This allows for a simpler representation of strings that contain colour codes,
bold codes, and such, without having to write and handle raw bytes when
assembling outgoing messages.

This lets you turn raw IRC messages into our escaped versions, and turn escaped
versions back into raw messages suitable for sending on IRC connections. This
is designed to be used on things like PRIVMSG / NOTICE commands, MOTD blocks,
and such.

The escape character we use in this library is the dollar sign ("$"), along
with the given escape characters:

----------------------------
 Name       | Escape | Raw
----------------------------
 Dollarsign |   $$   |  $
 Bold       |   $b   | 0x02
 Colour     |   $c   | 0x03
 Italic     |   $i   | 0x1d
 Underscore |   $u   | 0x1f
 Reset      |   $r   | 0x0f
----------------------------

This package is in alpha.
*/
package ircfmt
