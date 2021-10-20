# Slack app setup

This page contains the Slack app manifest which can be used to configure the application.
Make sure to set the correct scopes.
Make sure to specify the address `bot/command` for the slash command.

```yaml
_metadata:
  major_version: 1
  minor_version: 1
display_information:
  name: devopsbot
  description: DevOpsBot
  background_color: "#004492"
features:
  bot_user:
    display_name: devopsbot
    always_online: false
  slash_commands:
    - command: /devopsbot
      url: https://<domain>/bot/command
      description: DevOpsBot
      usage_hint: "[help, incident, resolve]"
      should_escape: false
oauth_config:
  scopes:
    user:
      - channels:read
      - channels:write
      - groups:read
      - groups:write
      - im:read
      - im:write
      - mpim:read
      - mpim:write
      - reminders:write
      - users:read
    bot:
      - channels:manage
      - channels:read
      - chat:write
      - chat:write.customize
      - commands
      - groups:read
      - groups:write
      - im:read
      - im:write
      - mpim:read
      - mpim:write
      - users:read
settings:
  interactivity:
    is_enabled: true
    request_url: https://<domain>/bot/interactive
  org_deploy_enabled: false
  socket_mode_enabled: false
  token_rotation_enabled: false
```
