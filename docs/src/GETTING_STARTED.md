# Getting started
## Build
These instructions assume you are on a Mac.

Make sure you have the latest version of Go installed: `brew update && brew upgrade`

Install `golangci-lint` to be able to run `make lint`: `brew install golangci/tap/golangci-lint`

To build a binary locally:

```console
$ make build
```

To build a Docker image, a [GitHub personal access token](https://help.github.com/en/articles/creating-a-personal-access-token-for-the-command-line)
is required so that some private dependencies can be pulled down at build-time:

```console
$ GITHUB_TOKEN=<personal access token> make image.iid
$ docker run -it --rm $(< image.iid)
```

## Helm chart
TODO

## Local development
To run locally you need valid certificates (use `mkcert`), and you need `devopsbot` to resolve to `127.0.0.1`:

```console
$ mkcert devopsbot
$ sudo echo "127.0.0.1 devopsbot" >> /etc/hosts
```

Then, you can run a local copy after having provided values for parameters
that are empty by default:

```console
$ bin/devopsbot \
  --slack.accessToken=xoxb-.... \
  --slack.signingSecret=...
```

And access it at https://devopsbot:3443 or http://devopsbot:3333

See the `--help` output for more flags.

To test `devopsbot` functionality, it must be accessible by Slack. Optionally
use [inlets](https://github.com/inlets/inlets) to expose the locally running
`devopsbot` to the Internet. The `inlets` server can run on a free tier EC2
instance. Make sure it is accessible from the whole Internet and port range is
wide enough. Note its publicly accessible IPv4 IP. Run the `inlets` server from
the EC2 instance. Run the `inlets` client from laptop where `devopsbot` is running.

### Verify routes

One way of verifying routes is via manual `POST` requests:
1. Start the bot
1. Generate a Slack signature for the request body being investigated, in 
   this case `command=/devopsbot`, for example via `go`:
    ```go
    package main

    import (
      "crypto/hmac"
      "crypto/sha256"
      "encoding/hex"
      "fmt"
    )

    func main() {
      h := hmac.New(sha256.New, []byte("<signing secret, empty if not specified>"))
      _, _ = h.Write([]byte("v0:<UNIX time stamp>:command=%2Fdevopsbot"))
      computed := h.Sum(nil)
      fmt.Println(hex.EncodeToString(computed))
    }
    ```
1. Post the request:
    ```sh
    curl -X POST -H 'Content-type: application/x-www-form-urlencoded' -H 'X-Slack-Request-Timestamp: <UNIX time stamp>' -H 'X-Slack-Signature: v0=<signature from previous step>' --data 'command=%2Fdevopsbot' localhost:3333/bot/command
    ```
