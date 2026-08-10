# Changelog
All notable changes to irc-go will be documented in this file.

## [0.7.0] - 2026-08-05

irc-go v0.7.0 is a new tagged release. It includes enhancements to `ircevent`, our IRC client library. There are no API breaks relative to previous tagged versions.

### Added
* Added `(*ircevent.Connection).MaxMessageLength()` and `FetchUserHost`. `MaxMessageLength` can be used to compute the maximum length of a `PRIVMSG` or `NOTICE` message that can be sent to a target without truncation. By default, the initial invocation of `MaxMessageLength` returns a safe default value until a more accurate measurement can be obtained; `FetchUserHost` can be enabled to perform this measurement eagerly on each connection, at the cost of sending an additional command. (#114, #123, thanks [@lollipopman](https://github.com/lollipopman)!)
* Added `(*ircevent.Connection).SendBatch`, `SendBatchWithLabel`, and `GetLabeledResponseForBatch`. These are APIs for sending a compliant [IRCv3 client-initiated batch](https://ircv3.net/specs/extensions/client-batch), for example with the [multiline extension](https://ircv3.net/specs/extensions/multiline). (#106, #119)
* Added `(*ircevent.Connection).AddRawCallback`, an API for adding callbacks that take in the unparsed IRC line, without filtering on the command. This can be used to implement programmatic debug logs, or catch-all handling of otherwise unhandled commands. (#76, #116, thanks [@jcjordyn130](https://github.com/jcjordyn130)!)

### Security
* `ircevent` is now more resilient to malicious `BATCH` responses from servers, including excessively large or deep batches. `(*ircevent.Connection).MaxTotalBatchSize` was added to limit the total size of buffered batch data; it defaults to 8 MiB when unset. (#118, #122)

### Changed
* Shortened the timeout for `GetLabeledResponse` (now approximately 1 minute by default) (#124)

## [0.6.0] - 2026-03-15

irc-go v0.6.0 is a new tagged release. It includes a bug fix to `ircevent`, our IRC client library. There are no API breaks relative to previous tagged versions.

### Fixed
* `ircevent` now sends `NICK` and `USER` without waiting for `CAP` responses. This improves compliance with the [specification](https://ircv3.net/specs/extensions/capability-negotiation.html), speeds up connecting to legacy servers, and may improve compatibility with some servers. (#111, #112, thanks [@Yahweasel](https://github.com/Yahweasel)!)

## [0.5.0] - 2026-01-15

irc-go v0.5.0 is a new tagged release. It incorporates enhancements to `ircevent`, our IRC client library, and `ircutils`, our collection of miscellaneous IRC utilities. There are no API breaks relative to previous tagged versions.

### Added
* Added support for TLS client certificate authentication with `SASL EXTERNAL` (set `(*ircevent.Connection).SASLMech` to `EXTERNAL` and add the client certificate to `(*ircevent.Connection).TLSConfig`) (#102)
* Added `(*ircevent.Connection).GetReplyTarget`, which determines whether a message was sent to a channel or as a DM and returns the correct target to reply to (#97, #105)
* Added `ircutils.SASLBuffer`, which handles base64 decoding and concatenation when receiving arbitrarily large SASL responses, up to a configurable limit (#102, #104)
* Added `ircutils.EncodeSASLResponse`, which handles base64 encoding and chunking when emitting arbitrarily large SASL responses (#102)
* Exposed `ircevent.ClientHasQuit` error, which is returned when attempting to reconnect after `Quit()` was already called (#99, thanks [@frrad](https://github.com/frrad)!)

### Fixed
* `ircevent` is now capable of correctly emitting SASL PLAIN responses that exceed 400 bytes of base64 (#102)

### Changed
* `ircmsg` now validates that the IRC command must be ASCII (#108, #109)

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
