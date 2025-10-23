# WebRTC TURN Connectivity Probe

This container spins up two in-process WebRTC peers that are forced to relay
traffic through a TURN server. It is useful for validating that a TURN service
is reachable and functioning for data channels (the same TURN credentials can
be reused for media flows).

## Build

```bash
docker build -t local/webrtc-turn-check webrtc-test
```

## Run

Provide the TURN endpoint (domain and ports) via environment variables. If your
TURN service requires credentials, set `TURN_USERNAME` and `TURN_PASSWORD`.

```bash
docker run --rm \
  -e TURN_DOMAIN=turn.server.ghifari.dev \
  -e TURN_PORT=3478 \
  -e TURN_TLS_PORT=5349 \
  -e TURN_AUTH_SECRET=super-secret-ice \
  local/webrtc-turn-check
```

On success the container prints the selected relay candidate pair and exits
with status 0. Connection failures or authentication errors cause a non-zero
exit code.

### Environment Variables

- `TURN_DOMAIN` (required): TURN server hostname.
- `TURN_PORT` (default `3478`): Plain TURN port for both UDP and TCP transports.
- `TURN_TLS_PORT` (default `5349`): TURN over TLS port (set to `0` to disable).
- `TURN_USERNAME`, `TURN_PASSWORD`: Optional long-term credential pair.
- `TURN_AUTH_SECRET`: When set, the tool derives a short-lived username/password
  using Coturn's REST authentication scheme (`static-auth-secret`). Overrides
  `TURN_USERNAME`/`TURN_PASSWORD`.

## Example Output

```
offer peer selected pair: local=172.17.0.2:54824 (relay) <-> remote=10.42.0.78:3478 (srflx)
answer peer selected pair: local=172.17.0.2:48756 (relay) <-> remote=10.42.0.78:3478 (srflx)
offer received message: hello-from-answer (remaining confirmations: 1)
answer received message: hello-from-offer (remaining confirmations: 0)
WebRTC relay over TURN succeeded via turn.server.ghifari.dev
```
