# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY . .
RUN go build -o lookout-connect ./cmd

FROM eclipse-mosquitto:2
RUN apk add --no-cache bash coreutils

COPY --from=builder /src/lookout-connect /usr/local/bin/lookout-connect
COPY mosquitto.conf.template /etc/mosquitto/mosquitto.conf.template
COPY entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh

EXPOSE 1883

ENTRYPOINT ["/entrypoint.sh"]