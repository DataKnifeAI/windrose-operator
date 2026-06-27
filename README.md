# Windrose Operator

Kubernetes operator for [Windrose](https://playwindrose.com) dedicated game servers using the official Linux image [`windroseserver/windroseserver`](https://hub.docker.com/r/windroseserver/windroseserver).

This project complements [windrose-server-k8s](https://github.com/DataKnifeAI/windrose-server-k8s), which packages a Wine-based server image and static Kustomize manifests. The operator:

- Runs the **native Linux** image from the Windrose team (not the Wine fork)
- Manages servers declaratively via a `WindroseServer` custom resource
- Matches the **Envoy Gateway** exposure pattern used in DataKnife `prd-apps` (`game-servers` namespace)

## Comparison

| | windrose-operator | windrose-server-k8s |
|--|---------------------|---------------------|
| Image | `windroseserver/windroseserver` | Harbor Wine image (`indifferentbroccoli` fork) |
| Deployment | Operator-managed CR | Kustomize manifests |
| External access | Envoy Gateway (TCPRoute/UDPRoute) | Envoy Gateway (manual manifests today) |
| Config | `ServerDescription.json` ConfigMap | Env vars + generated JSON on disk |
| Mount path | `/home/ue_user/app/R5/...` | `/home/steam/server-files/...` |

## Features

- `WindroseServer` CRD with status (phase, ready, connection address/port)
- Envoy Gateway exposure: Gateway, EnvoyProxy, TCPRoute, UDPRoute
- ClusterIP backend services (`{name}` and `{name}-envoy`)
- PVC for world saves, ConfigMap for `ServerDescription.json`
- Pod resources auto-selected from `maxPlayerCount` (override supported)

## Architecture

```
Clients → spec.gateway.address (Kube-VIP / MetalLB)
              ↓
      {base}-gateway  (GatewayClass: envoy)
              ↓
   TCPRoute / UDPRoute  →  {name}-envoy  (ClusterIP)
              ↓
      {name} Deployment  (windroseserver/windroseserver)
              ↓
      PVC  +  ServerDescription.json (ConfigMap)
```

Each `WindroseServer` reconciles these Kubernetes resources:

| Kind | Default name (CR: `windrose-server`) |
|------|--------------------------------------|
| Deployment | `windrose-server` |
| PersistentVolumeClaim | `windrose-server-files` |
| ConfigMap | `windrose-server-config` |
| Service (ClusterIP) | `windrose-server` |
| Service (Envoy backend) | `windrose-server-envoy` |
| Gateway | `windrose-gateway` |
| EnvoyProxy | `game-windrose-kubevip` |
| TCPRoute | `windrose-game-tcp` |
| UDPRoute | `windrose-game-udp` |

Gateway-related names strip a trailing `-server` suffix from the CR name before applying the patterns above. Override with `spec.gateway.gatewayName` or `spec.gateway.envoyProxyName`.

## Prerequisites

- Kubernetes 1.28+
- [Envoy Gateway](https://gateway.envoyproxy.io/) installed with a `GatewayClass` named `envoy`
- A StorageClass for persistent volumes (35 Gi minimum per Windrose guidance)
- One dedicated external IP per server (`spec.gateway.address`), typically from Kube-VIP or MetalLB

## Quick start

Install the CRD and operator:

```shell
kubectl apply -k config/default
```

Create a server (adjust IP, namespace, and storage class for your cluster):

```shell
kubectl apply -f config/samples/windrose_v1alpha1_windroseserver.yaml
```

Check status:

```shell
kubectl get windroseserver -n game-servers
kubectl describe windroseserver windrose-server -n game-servers
```

Connect in-game via **Play → Connect to Server** using:

- Address: `.status.connectionAddress` (falls back to `spec.gateway.address`)
- Port: `.status.connectionPort` (defaults to `7777`)

## Example

```yaml
apiVersion: windrose.dataknife.ai/v1alpha1
kind: WindroseServer
metadata:
  name: windrose-server
  namespace: game-servers
spec:
  serverImage: windroseserver/windroseserver:latest
  gateway:
    address: 192.168.14.186
    className: envoy
  useDirectConnection: true
  directConnectionServerPort: 7777
  directConnectionProxyAddress: 0.0.0.0
  serverName: My Windrose Server
  maxPlayerCount: 4
  userSelectedRegion: EU
  storageSize: 35Gi
  storageClassName: truenas-csi-nfs
  nodeSelector:
    kubernetes.io/os: linux
    kubernetes.io/arch: amd64
```

## Spec reference

### Gateway (required)

| Field | Default | Description |
|-------|---------|-------------|
| `gateway.address` | *(required)* | External IP for this server (e.g. `192.168.14.186`) |
| `gateway.className` | `envoy` | GatewayClass name |
| `gateway.gatewayName` | `{base}-gateway` | Override Gateway resource name |
| `gateway.envoyProxyName` | `game-{base}-kubevip` | Override EnvoyProxy resource name |
| `gateway.externalTrafficPolicy` | `Cluster` | Envoy LoadBalancer traffic policy |

### Server image

| Field | Default | Description |
|-------|---------|-------------|
| `serverImage` | `windroseserver/windroseserver:latest` | Official Linux server image |
| `imagePullPolicy` | `IfNotPresent` | Container pull policy |
| `imagePullSecrets` | | Pull secrets for private registries |
| `nodeSelector` | | Pin the pod to specific nodes |

### Game settings

These fields are written to `ServerDescription.json`. See the [dedicated server guide](https://playwindrose.com/dedicated-server-guide) for semantics.

| Field | Default | Description |
|-------|---------|-------------|
| `useDirectConnection` | `true` | Direct IP connections (required for Kubernetes) |
| `directConnectionServerPort` | `7777` | Game port exposed via TCP and UDP |
| `directConnectionProxyAddress` | `0.0.0.0` | Bind address inside the container |
| `serverName` | | Display name in the server browser |
| `password` | | Server password (empty = public) |
| `maxPlayerCount` | `4` | Max simultaneous players (1–32) |
| `userSelectedRegion` | | `SEA`, `CIS`, or `EU`; empty = auto-detect |
| `inviteCode` | | Invite code when `useDirectConnection` is `false` |
| `p2pProxyAddress` | `127.0.0.1` | P2P proxy address for invite-code mode |
| `autoLoadLatestBackupIfHasBroken` | `true` | Restore broken saves from backups on launch |

Direct IP and invite-code modes are mutually exclusive in Windrose. Kubernetes deployments should use direct connection.

### Storage

| Field | Default | Description |
|-------|---------|-------------|
| `storageSize` | `35Gi` | PVC capacity for world saves |
| `storageClassName` | | StorageClass for the saves PVC |

Saves are mounted at `/home/ue_user/app/R5/Saved`. `WorldDescription.json` is created by the server on first boot and stored on the PVC.

### Pod resources

When `spec.resources` is **omitted**, CPU and memory are derived from `maxPlayerCount` using [Windrose hardware guidance](https://playwindrose.com/dedicated-server-guide):

| Players | CPU request | Memory request | CPU limit | Memory limit |
|---------|-------------|----------------|-----------|--------------|
| 1–2 | 2 | 8Gi | 4 | 10Gi |
| 3–4 | 2 | 12Gi | 4 | 16Gi |
| 5–32 | 2 | 16Gi | 4 | 16Gi |

When `spec.resources` is **set** (requests or limits), it fully overrides auto-selection:

```yaml
spec:
  maxPlayerCount: 10
  resources:
    requests:
      cpu: 250m
      memory: 2Gi
    limits:
      cpu: "4"
      memory: 10Gi
```

## Status

| Field | Description |
|-------|-------------|
| `phase` | `Pending`, `Running`, or `Failed` |
| `ready` | `true` when the game server pod is ready |
| `connectionAddress` | IP clients connect to |
| `connectionPort` | Port clients connect to |
| `message` | Human-readable detail |

Short name: `kubectl get ws`

## Official Docker image layout

The operator mirrors the upstream Docker run invocation:

```shell
docker run --user ue_user --name WindroseServer \
  -p 7777:7777/tcp -p 7777:7777/udp -d \
  -v {path}/Saved:/home/ue_user/app/R5/Saved \
  -v {path}/ServerDescription.json:/home/ue_user/app/R5/ServerDescription.json \
  windroseserver/windroseserver:latest
```

## Development

Requires Go 1.25+ and [golangci-lint](https://golangci-lint.run/) for local linting.

```shell
make generate manifests   # CRD, RBAC, deepcopy
make test                 # unit tests with race detector
make lint                 # golangci-lint
make ci                   # generate, vet, lint, and test
make build                # bin/manager
make run                  # local controller
make docker-build IMG=harbor.dataknife.net/library/windrose-operator:latest
```

CI runs lint and tests on every push and pull request.

## Related projects

- [DataKnifeAI/windrose-server-k8s](https://github.com/DataKnifeAI/windrose-server-k8s) — Wine-based image and Kustomize manifests
- [GitLab mirror](docs/GITLAB_MIRROR.md) — CI builds `harbor.dataknife.net/library/windrose-operator`
- [Windrose Dedicated Server Guide](https://playwindrose.com/dedicated-server-guide)
- [windroseserver/windroseserver on Docker Hub](https://hub.docker.com/r/windroseserver/windroseserver)

## License

Apache License 2.0 — see [LICENSE](LICENSE).
