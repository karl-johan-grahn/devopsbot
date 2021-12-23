# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.13.4] - 2021-12-23
### Updates
- Update module github.com/rs/zerolog to v1.26.1

## [0.13.3] - 2021-12-23
### Updates
- Update errata-ai/vale-action action to v1.4.3

## [0.13.2] - 2021-12-23
### Updates
- Update docker/login-action action to v1.12.0

## [0.13.1] - 2021-12-23
### Updates
- Update docker/metadata-action action to v3.6.2

## [0.13.0] - 2021-12-22
### Updates
- When generating version file, enable matching non-annotated tags
- Slack does not yet allow users to create reminders recurring more often than once a day, so just create one that runs daily 30 min after the incident has been declared
- Include year in Slack channel name to decrease chance of having name creation conflicts and to make the name more explicit
- Describe incidents according to severity and impact

## [0.12.0] - 2021-11-29
### Adds
- Add broadcast channel as input option, add dispatch action when characters are entered, update to Slack Go API v0.10.0

## [0.11.0] - 2021-11-16
### Updates
- Update to Go 1.17.3

## [0.10.0] - 2021-11-03
### Updates
- Update UI such as update wording, add hints for responder and commander

## [0.9.0] - 2021-11-02
### Updates
- Update date handling

## [0.8.0] - 2021-10-27
### Updates
- French string updates and package updates

## [0.7.0] - 2021-10-25
### Updates
- Update strings and remove invitation of bot

## [0.6.0] - 2021-10-22
### Updates
- For security reasons a bot cannot invite itself to a channel, so only use a default broadcast channel for simplicity

## [0.5.0] - 2021-10-21
### Adds
- Add French localization

## [0.4.0] - 2021-10-21
### Adds
- Add CodeQL to find security vulnerabilities

## [0.3.0] - 2021-10-20
### Adds
- Add link to documentation and fix book path

## [0.2.2] - 2021-10-20
### Fix
- Tags pattern does not support regex fully

## [0.2.1] - 2021-10-20
### Fix
- Fix tags input

## [0.2.0] - 2021-10-20
### Adds
- Handle semantic version tagging of docker publishing

## [0.1.0] - 2021-10-18
### Adds
- Initial commit
