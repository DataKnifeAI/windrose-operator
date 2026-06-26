# Windrose Operator

Kubernetes operator for [Windrose](https://playwindrose.com) dedicated game servers using the official Linux container image [`windroseserver/windroseserver`](https://hub.docker.com/r/windroseserver/windroseserver).

This project complements [windrose-server-k8s](https://github.com/DataKnifeAI/windrose-server-k8s), which packages a Wine-based server image. The operator targets the native Linux image published by the Windrose team and matches the **Envoy Gateway** exposure pattern used on DataKnife `prd-apps` clusters.

## Features

- Declarative `WindroseServer` custom resource
- **Envoy Gateway** exposure via Gateway + EnvoyProxy + TCPRoute/UDPRoute (matches production `game-servers` namespace)
- Official `windroseserver/windroseserver` image with correct mount paths
- ClusterIP backend services (`{name}` and `{name}-envoy`)
- Persistent world saves on a PVC
- `ServerDescription.json` managed via ConfigMap

## Architecture

```
Clients → Kube-VIP IP (spec.gateway.address)
              ↓
      {base}-gateway (Envoy Gateway)
              ↓
   TCPRoute / UDPRoute → {name}-envoy (ClusterIP)
              ↓
      {name} Deployment (windroseserver/windroseserver)
```

For a CR named `windrose-server`, resource names match the existing production layout:

| Resource | Name |
|----------|------|
| Gateway | `windrose-gateway` |
| EnvoyProxy | `game-windrose-kubevip` |
| TCPRoute | `windrose-game-tcp` |
| UDPRoute | `windrose-game-udp` |
| Backend Service | `windrose-server-envoy` |

## Prerequisites

- Kubernetes 1.28+
- [Envoy Gateway](https://gateway.envoyproxy.io/) with `GatewayClass` `envoy`
- A StorageClass for persistent volumes (35 Gi minimum recommended)
- A dedicated Kube-VIP / MetalLB IP per server (`spec.gateway.address`)

## Quick start

Install the CRD and operator:

```shell
kubectl apply -k config/default
```

Create a server (adjust the gateway IP and storage class for your cluster):

```shell
kubectl apply -f config/samples/windrose_v1alpha1_windroseserver.yaml
```

Watch status:

```shell
kubectl get windroseserver windrose-server -n game-servers -w
```

Connect in-game with **Play → Connect to Server** using `.status.connectionAddress` and `.status.connectionPort`.

## WindroseServer spec

| Field | Default | Description |
|-------|---------|-------------|
| `serverImage` | `windroseserver/windroseserver:latest` | Official Linux server image |
| `gateway.address` | *(required)* | External IP (e.g. `192.168.14.186`) |
| `gateway.className` | `envoy` | GatewayClass name |
| `useDirectConnection` | `true` | Direct IP connections |
| `directConnectionServerPort` | `7777` | Game port (TCP + UDP) |
| `storageSize` | `35Gi` | PVC size for world saves |
| `storageClassName` | | StorageClass for saves PVC |

Settings map to [`ServerDescription.json`](https://playwindrose.com/dedicated-server-guide) fields. World configuration (`WorldDescription.json`) is created by the server on first boot and stored on the PVC.

## Official Docker image layout

The operator mounts resources the same way as the upstream Docker instructions:

```shell
docker run --user ue_user --name WindroseServer \
  -p 7777:7777/tcp -p 7777:7777/udp -d \
  -v {path}/Saved:/home/ue_user/app/R5/Saved \
  -v {path}/ServerDescription.json:/home/ue_user/app/R5/ServerDescription.json \
  windroseserver/windroseserver:latest
```

## Development

```shell
make generate manifests
make test
make run
```

## Related projects

- [DataKnifeAI/windrose-server-k8s](https://github.com/DataKnifeAI/windrose-server-k8s) — Wine-based Windrose server image and Kustomize manifests
- [Windrose Dedicated Server Guide](https://playwindrose.com/dedicated-server-guide)
- [windroseserver/windroseserver on Docker Hub](https://hub.docker.com/r/windroseserver/windroseserver)

## License

Apache License 2.0 — see [LICENSE](LICENSE).
