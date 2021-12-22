# Getting started
## Invite bot to broadcast channel
For security reasons, the Slack API prohibits bots to invite themselves to channels.
The bot therefore needs to be manually invited to the chosen broadcast channel before it can communicate there.
Invite the bot to the broadcast channel by using this command in the channel:

```console
/invite devopsbot
```

## Build
These instructions assume you are on a Mac.

Make sure you have the latest version of Go installed: `brew update && brew upgrade`

Install `golangci-lint` to be able to run `make lint`: `brew install golangci/tap/golangci-lint`

To build the binary locally:

```console
$ make build
```

To build the Docker image locally:

```console
$ make image.iid
```

## Deployment
The bot can be deployed any preferred way.

### Kubernetes
A certificate need to be issued to expose the application over HTTPS, for example via ZeroSSL or Let's Encrypt.

The application need to be made publicly available.

The Kubernetes resources could look like this for example:

```yaml
---
# Source: helmchart/templates/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: devopsbot-settings
data:
  addr: :3333
  incident.environments: |-
    [
      "Staging",
      "Production"
    ]
  incident.regions: |-
    [
      "eu-west-1",
      "us-east-1"
    ]
  incident.severityLevels: |-
    [
      "high",
      "medium",
      "low"
    ]
  incident.impactLevels: |-
    [
      "high",
      "medium",
      "low"
    ]
  server.prometheusNamespace: devopsbot
  tls.addr: :3443
  tls.cert: /var/devopsbot/tls.crt
  tls.key: /var/devopsbot/tls.key
  trace: "false"
  verbose: "false"
---
# Source: helmchart/templates/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: devopsbot
  labels:
    app: devopsbot
spec:
  ports:
      - port: 3333
        targetPort: 3333
        protocol: TCP
        name: "http"
  selector:
    app: devopsbot
---
# Source: helmchart/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: devopsbot
  labels:
    app: devopsbot

spec:
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  replicas: 1
  selector:
    matchLabels:
      app: devopsbot
  template:
    metadata:
      labels:
        app: devopsbot
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/path: "/metrics"
        prometheus.io/port: "3333"
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 25
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app
                  operator: In
                  values:
                  - devopsbot
              topologyKey: "kubernetes.io/hostname"
      containers:
        - name: devopsbot
          image: <path_to_image>
          imagePullPolicy: Always
          ports:
            - name: http
              containerPort: 3333
          resources:
            limits:
              cpu: 900m
              memory: 256Mi
            requests:
              cpu: 600m
              memory: 20Mi
          readinessProbe:
            httpGet:
              path: /ready
              port: 3333
            initialDelaySeconds: 1
            timeoutSeconds: 1
            periodSeconds: 2
            failureThreshold: 300
          livenessProbe:
            httpGet:
              path: /live
              port: 3333
            initialDelaySeconds: 300
            timeoutSeconds: 2
            periodSeconds: 3
            failureThreshold: 2
          env:
            - name: slack.botAccessToken
              valueFrom:
                secretKeyRef:
                  key: slack.botAccessToken
                  name: app-secrets
            - name: slack.userAccessToken
              valueFrom:
                secretKeyRef:
                  key: slack.userAccessToken
                  name: app-secrets
            - name: slack.adminGroupID
              valueFrom:
                secretKeyRef:
                  key: slack.adminGroupID
                  name: app-secrets
            - name: slack.broadcastChannelID
              valueFrom:
                secretKeyRef:
                  key: slack.broadcastChannelID
                  name: app-secrets
            - name: slack.signingSecret
              valueFrom:
                secretKeyRef:
                  key: slack.signingSecret
                  name: app-secrets
            - name: incidentDocTemplateURL
              valueFrom:
                secretKeyRef:
                  key: incidentDocTemplateURL
                  name: app-secrets
            - name: server.prometheusNamespace
              valueFrom:
                configMapKeyRef:
                  key: server.prometheusNamespace
                  name: devopsbot-settings
            - name: incident.environments
              valueFrom:
                configMapKeyRef:
                  key: incident.environments
                  name: devopsbot-settings
            - name: incident.regions
              valueFrom:
                configMapKeyRef:
                  key: incident.regions
                  name: devopsbot-settings
            - name: incident.severityLevels
              valueFrom:
                configMapKeyRef:
                  key: incident.severityLevels
                  name: devopsbot-settings
            - name: incident.impactLevels
              valueFrom:
                configMapKeyRef:
                  key: incident.impactLevels
                  name: devopsbot-settings
            - name: addr
              valueFrom:
                configMapKeyRef:
                  key: addr
                  name: devopsbot-settings
            - name: tls.addr
              valueFrom:
                configMapKeyRef:
                  key: tls.addr
                  name: devopsbot-settings
            - name: tls.cert
              valueFrom:
                configMapKeyRef:
                  key: tls.cert
                  name: devopsbot-settings
            - name: tls.key
              valueFrom:
                configMapKeyRef:
                  key: tls.key
                  name: devopsbot-settings
            - name: verbose
              valueFrom:
                configMapKeyRef:
                  key: verbose
                  name: devopsbot-settings
            - name: trace
              valueFrom:
                configMapKeyRef:
                  key: trace
                  name: devopsbot-settings
          volumeMounts:
            - name: tls-cert
              mountPath: /var/devopsbot
              readOnly: true
      volumes:
        - name: tls-cert
          secret:
            secretName: <secret_with_tls_cert>
---
```

## Local development
To run the bot locally a valid certificates is needed by using for example `mkcert`, and `devopsbot` need to resolve to `127.0.0.1`:

```console
$ mkcert devopsbot
$ sudo echo "127.0.0.1 devopsbot" >> /etc/hosts
```

Then, run a local copy after having provided values for parameters
that are empty by default:

```console
$ bin/devopsbot \
  --slack.botAccessToken=xoxb-.... \
  --slack.signingSecret=...
```

And access it at <https://devopsbot:3443> or <http://devopsbot:3333>.

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

## Troubleshooting

The bot assumes basic infrastructure to be working.
A mitigation plan should be made if any of these systems fail:
- The app runs via Slack which can become unavailable, check [Slack System Status](https://status.slack.com/)
- The bot need to be deployed successfully to be available in Slack, check the deployment
- The app Docker image is hosted on GitHub container registry which can become unavailable, check [GitHub Status](https://www.githubstatus.com/)
- No internet is available, check with your internet provider
- No electricity is available, check with your electricity provider
- Input devices work as expected, check keyboard and mouse
