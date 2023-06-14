# Changelog
All notable changes to irc-go will be documented in this file.

## [0.4.0] - 2023-06-14

irc-go v0.4.0 is a new tagged release. It incorporates enhancements to `ircmsg`, our IRC protocol handling library, and `ircfmt`, our library for handling [IRC formatting codes](https://modern.ircdocs.horse/formatting.html). There are no API breaks relative to previous tagged versions.

### Changed
* `ircmsg.ParseLineStrict` now does UTF8-aware truncation of the parsed message, using the same algorithm as `ircmsg.LineBytesStrict` (if the truncated message is invalid as UTF8, up to 3 additional bytes are removed in an attempt to make it valid)
* `TruncateUTF8Safe` was moved from `ircutils` to `ircmsg`. (An alias is provided in `ircutils` for compatibility.)

### Added
* `ircfmt.Unescape` now accepts the American spellings "gray" and "light gray", in addition to "grey" and "light grey"


## [0.3.0] - 2023-02-13

irc-go v0.3.0 is a new tagged release. It incorporates enhancements to `ircevent`, our IRC client library, and `ircfmt`, our library for handling [IRC formatting codes](https://modern.ircdocs.horse/formatting.html). There are no API breaks relative to previous tagged versions.

Thanks to [@kofany](https://github.com/kofany) for helpful discussions.

### Added
* Added `(*ircevent.Connection).DialContext`, an optional callback for customizing how ircevent creates IRC connections. Clients can create a custom `net.Dialer` instance and pass in its `DialContext` method, or use a callback that invokes a proxy, e.g. a SOCKS proxy (see `ircevent/examples/proxy.go` for an example). (#64, #91)
* Added `ircfmt.Split()`, which splits an IRC message containing formatting codes into a machine-readable representation (a slice of `ircfmt.FormattedSubstring`). (#89)
* Added `ircfmt.ParseColor()`, which parses an IRC color code string into a machine-readable representation (an `ircfmt.ColorCode`). (#89, #92)

### Fixed
* Fixed some edge cases in `ircfmt.Strip()` (#89)

## [0.2.0] - 2022-06-22

irc-go v0.2.0 is a new tagged release, incorporating enhancements to `ircevent`, our IRC client library. There are no API breaks relative to v0.1.0.

Thanks to [@ludviglundgren](https://github.com/ludviglundgren), [@Mikaela](https://github.com/Mikaela), and [@progval](https://github.com/progval) for helpful discussions, testing, and code reviews.

### Added
* Added `(*ircevent.Connection).GetLabeledReponse`, a synchronous API for getting a [labeled message response](https://ircv3.net/specs/extensions/labeled-response). (#74, thanks [@progval](https://github.com/progval)!)
* Added `(*ircevent.Connection).AddDisconnectCallback`, which allows registering callbacks that are invoked whenever ircevent detects disconnection from the server. (#78, #80, thanks [@ludviglundgren](https://github.com/ludviglundgren)!)
* Added `(ircevent.Connection).SASLOptional`; when set to true, this makes failure to SASL non-fatal, which can simplify compatibility with legacy services implementations (#78, #83, thanks [@ludviglundgren](https://github.com/ludviglundgren)!)
* `ircevent` now exposes most commonly used numerics as package constants, e.g. `ircevent.RPL_WHOISUSER` (`311`)

### Fixed
* Calling `(*ircevent.Connection).Reconnect` now takes immediate effect, even if the client is waiting for `ReconnectFreq` to expire (i.e. automatic reconnection has been throttled) (#79)
* `(*ircevent.Connection).CurrentNick()` now returns the correct value when called from a `NICK` callback (#78, #84, thanks [@ludviglundgren](https://github.com/ludviglundgren)!)

## [0.1.0] - 2022-01-19

irc-go v0.1.0 is our first tagged release. Although the project is not yet API-stable, we envision this as the first step towards full API stability. All API breaks will be documented in this changelog; we expect any such breaks to be modest in scope.

### Added
* Added `(*ircmsg.Message).Nick()` and `(*ircmsg.Message).NUH()`, which permissively interpret the source of the message as a NUH. `Nick()` returns the name component of the source (either nickname or server name) and `NUH` returns all three components (name, username, and hostname) as an `ircmsg.NUH`. (#67, #66, #58)

### Changed
* The source/prefix of the message is now parsed into `(ircmsg.Message).Source`, instead of `(ircmsg.Message).Prefix` (#68)
* `ircevent.ExtractNick()` and `ircevent.SplitNUH()` are deprecated in favor of `(*ircmsg.Message).Nick()` and `(*ircmsg.Message).NUH()` respectively
