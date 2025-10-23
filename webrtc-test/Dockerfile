# syntax=docker/dockerfile:1

FROM golang:1.22 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/webrtc-turn-check ./...

FROM gcr.io/distroless/base-debian12

COPY --from=build /out/webrtc-turn-check /webrtc-turn-check

ENV TURN_DOMAIN=turn.server.ghifari.dev \
    TURN_PORT=3478 \
    TURN_TLS_PORT=5349

ENTRYPOINT ["/webrtc-turn-check"]
