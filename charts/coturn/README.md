# Coturn Helm Chart

This chart deploys the [Coturn](https://github.com/coturn/coturn) TURN/STUN server on Kubernetes. It packages the official `coturn/coturn` container image and exposes key configuration options such as TURN ports, authentication settings, and persistence.

## Features

- Host networking enabled by default for optimal TURN/UDP performance.
- Configurable `turnserver.conf` shipped via ConfigMap; override with your own.
- Optional persistence for `/var/lib/coturn`.
- Flexible exposure strategy: use host networking or enable a Service/LoadBalancer.
- Hooks for Prometheus `ServiceMonitor`.

## Installing

```bash
helm install my-turn charts/coturn \
  --set turnserver.config="$(< my-turnserver.conf)" \
  --set turnserver.extraEnv[0].name=STATIC_AUTH_SECRET \
  --set turnserver.extraEnv[0].value="very-secret" \
  --set turnserver.extraArgs[0]='--use-auth-secret' \
  --set turnserver.extraArgs[1]='--static-auth-secret=$(STATIC_AUTH_SECRET)'
```

> **Important:** Set `realm`, authentication options, and relay networking parameters in the config or extra args before exposing the service.

## Values Overview

- `hostNetwork`: Default `true`. Disable to rely on a Kubernetes `Service`.
- `turnserver.config`: Inline Coturn configuration (templated with Helm values). Set `existingConfigMap` to reuse an external ConfigMap instead.
- `turnserver.args`: Defaults to `["-c", "/etc/coturn/turnserver.conf"]`.
- `turnserver.extraEnv` / `extraEnvFrom`: Supply static secrets or ConfigMaps (for example REST API shared secret).
- `turnserver.persistence`: Enable to back `/var/lib/coturn` with a PVC.
- `service.enabled`: Default `false`. Set to `true` when `hostNetwork` is disabled and you want a Kubernetes Service with custom ports.
- `serviceMonitor.enabled`: Creates a Prometheus `ServiceMonitor` targeting a `metrics` service port.
- `tls.*`: When `tls.enabled` is true the chart mounts the referenced secret (for example, one managed by cert-manager) and adds the proper `--cert/--pkey` flags automatically.

Consult `values.yaml` for the full list of options and default values.

## Example: Expose on turn.server.ghifari.dev:31192

If your cluster does not provide an external load-balancer, you can publish Coturn through a NodePort and map your DNS record (e.g. `turn.server.ghifari.dev`) to the node IP. The chart ships with ready-made values in `charts/coturn/examples/nodeport-values.yaml`:

```bash
helm install turn charts/coturn -f charts/coturn/examples/nodeport-values.yaml
```

Key points:

- `hostNetwork` is disabled so the pod binds only to the Service.
- UDP/TCP 3478 traffic is exposed on NodePort `31192`. Point `turn.server.ghifari.dev` to any node running the TURN pod, then use `turn.server.ghifari.dev:31192` in clients.
- The example also demonstrates wiring a static auth secret from `coturn-rest-secret`. Create that Kubernetes Secret first:

```bash
kubectl create secret generic coturn-rest-secret \
  --from-literal=sharedSecret='super-secret-key'
```

## TLS certificates via cert-manager

Issue a certificate for your TURN domain with cert-manager and point the chart to the generated secret by enabling `tls.enabled` and setting `tls.secretName`. The container mounts the secret at `tls.mountPath` (default `/coturn/tls`) and automatically adds `--cert`, `--pkey`, and optional `--ca-file` CLI flags.

Example `Certificate` resource:

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: turn-server-ghifari-dev
  namespace: default
spec:
  secretName: turn-server-ghifari-dev-tls
  dnsNames:
    - turn.server.ghifari.dev
  issuerRef:
    name: letsencrypt-production
    kind: ClusterIssuer
```

Deploy the chart with TLS enabled:

```bash
helm install turn charts/coturn \
  -f charts/coturn/examples/nodeport-values.yaml \
  --set tls.secretName=turn-server-ghifari-dev-tls
```
