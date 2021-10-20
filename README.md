# Devopsbot - Development Operations Bot
Devopsbot is a Slack bot written in Go using the [Slack API in Go](https://github.com/slack-go/slack).
It improves development efficiency by automating tasks such as:
- Declaring incidents
- Resolving incidents

The bot essentially automates the Incident Command System (ICS).

## Design Principles
The application is built around three key principles:
1. Everything the bot can do, any person is able to do - if it goes down and is unavailable, it will not block anyone
1. Secrets are maintained externally, for example via Hashicorp Vault
1. Configuration is maintained externally, for example via Kubernetes config maps
