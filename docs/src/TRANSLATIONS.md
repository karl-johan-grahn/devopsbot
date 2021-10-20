# Localizing the bot
The locale of the bot is set depending on the user's language preference in Slack.

[`go-i18n`](https://github.com/nicksnyder/go-i18n) manages the translations.

The web page for `go-i18n` describes the procedures for translating a new language and new messages, but they are describe here too. First get the tool: `go get -u github.com/nicksnyder/go-i18n/v2/goi18n`. The binary will be installed into `$GOPATH`.

## Translate a new language
If there is a new language to be added:
1. Create an empty message file for the new language, for example Finnish: `touch translate.fi.json`
1. Run `goi18n merge active.en.json translate.fi.json` to populate `translate.fi.json` with the messages to be translated
1. Translate `translate.fi.json` and rename it to `active.fi.json`
1. Load `active.fi.json` into the bundle

## Translate new messages
If there are new strings to be translated:
1. Run `goi18n extract -format json` to update `active.en.json` with new messages
1. Run `goi18n merge active.*.json` to generate updated  `translate.*.json` files
1. Translate all the messages in the `translate.*.json` files
1. Run `goi18n merge active.*.json translate.*.json` to merge the translated messages into the active message files
