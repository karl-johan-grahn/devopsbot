FROM docker.io/library/golang:1.18.4 AS builder

WORKDIR /go/src/github.com/karl-johan-grahn/devopsbot

COPY . ./

RUN make build

FROM docker.io/library/alpine:3.16.1

ARG VERSION
ARG REVISION

LABEL org.opencontainers.image.url="ghcr.io/karl-johan-grahn/devopsbot"
LABEL org.opencontainers.image.source="https://github.com/karl-johan-grahn/devopsbot"
LABEL org.opencontainers.image.version=$VERSION
LABEL org.opencontainers.image.revision=$REVISION

COPY --from=builder /go/src/github.com/karl-johan-grahn/devopsbot/bin/devopsbot /devopsbot
# Copy over string translations
COPY --from=builder /go/src/github.com/karl-johan-grahn/devopsbot/bot/active.*.json /

USER 1001:1001

CMD [ "/devopsbot" ]
