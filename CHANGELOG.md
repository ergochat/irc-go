# Changelog
All notable changes to irc-go will be documented in this file.

## [0.1.0] - 2022-01-19

irc-go v0.1.0 is our first tagged release. Although the project is not yet API-stable, we envision this as the first step towards full API stability. All API breaks will be documented in this changelog; we expect any such breaks to be modest in scope.

### Added
* Added `(*ircmsg.Message).Nick()` and `(*ircmsg.Message).NUH()`, which permissively interpret the source of the message as a NUH. `Nick()` returns the name component of the source (either nickname or server name) and `NUH` returns all three components (name, username, and hostname) as an `ircmsg.NUH`. (#67, #66, #58)

### Changed
* The source/prefix of the message is now parsed into `(ircmsg.Message).Source`, instead of `(ircmsg.Message).Prefix` (#68)
* `ircevent.ExtractNick()` and `ircevent.SplitNUH()` are deprecated in favor of `(*ircmsg.Message).Nick()` and `(*ircmsg.Message).NUH()` respectively
