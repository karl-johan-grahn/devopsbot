FROM golang:1.17.1 AS builder

WORKDIR /go/src/github.com/karl-johan-grahn/devopsbot

ARG TOKEN

ENV GIT_URL=https://"${TOKEN}":@github.com/

RUN git config --global url.${GIT_URL}.insteadOf https://github.com/

COPY . ./

ENV GOPRIVATE github.com/karl-johan-grahn/devopsbot

RUN make build

FROM alpine:3.14.2

RUN apk add --no-cache ca-certificates=20191127-r5

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
