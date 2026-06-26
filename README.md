# Windrose Operator

Kubernetes operator for [Windrose](https://playwindrose.com) dedicated game servers using the official Linux container image [`windroseserver/windroseserver`](https://hub.docker.com/r/windroseserver/windroseserver).

This project complements [windrose-server-k8s](https://github.com/DataKnifeAI/windrose-server-k8s), which packages a Wine-based server image. The operator targets the native Linux image published by the Windrose team.

## Features

- Declarative `WindroseServer` custom resource
- Manages Deployment, Service, PVC, and `ServerDescription.json` ConfigMap
- Direct IP connection mode enabled by default (required for Kubernetes)
- Persistent world saves on a PVC
- Resource defaults aligned with Windrose hardware guidance (4-player profile)

## Prerequisites

- Kubernetes 1.28+
- A StorageClass for persistent volumes (35 Gi minimum recommended)
- LoadBalancer or NodePort support for exposing game traffic (TCP + UDP on the same port)

## Quick start

Install the CRD and operator:

```shell
kubectl apply -k config/default
```

Create a server:

```shell
kubectl apply -f config/samples/windrose_v1alpha1_windroseserver.yaml
```

Watch status:

```shell
kubectl get windroseserver my-windrose-server -w
```

Connect in-game with **Play → Connect to Server** using the LoadBalancer address and port from `.status.connectionAddress` and `.status.connectionPort` when `useDirectConnection: true`.

## WindroseServer spec

| Field | Default | Description |
|-------|---------|-------------|
| `serverImage` | `windroseserver/windroseserver:latest` | Official Linux server image |
| `useDirectConnection` | `true` | Direct IP connections (recommended on Kubernetes) |
| `directConnectionServerPort` | `7777` | Game port (TCP + UDP) |
| `serverName` | | Display name in the server browser |
| `password` | | Optional server password |
| `maxPlayerCount` | `4` | Maximum simultaneous players |
| `userSelectedRegion` | | `SEA`, `CIS`, or `EU`; empty auto-selects |
| `storageSize` | `35Gi` | PVC size for world saves |
| `serviceType` | `LoadBalancer` | `LoadBalancer`, `NodePort`, or `ClusterIP` |

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

## Hardware guidance

| Players | CPU | RAM | Storage |
|---------|-----|-----|---------|
| 2 | 2 cores @ 3.2 GHz | 8 GB | 35 GB SSD |
| 4 | 2 cores @ 3.2 GHz | 12 GB | 35 GB SSD |
| 10 | 2 cores @ 3.2 GHz | 16 GB | 35 GB SSD |

Adjust `spec.resources` on the `WindroseServer` CR for larger player counts.

## Development

```shell
make generate manifests
make test
make run
```

Build the operator image:

```shell
make docker-build IMG=harbor.dataknife.net/library/windrose-operator:latest
```

## Related projects

- [DataKnifeAI/windrose-server-k8s](https://github.com/DataKnifeAI/windrose-server-k8s) — Wine-based Windrose server image and Kustomize manifests
- [Windrose Dedicated Server Guide](https://playwindrose.com/dedicated-server-guide)
- [windroseserver/windroseserver on Docker Hub](https://hub.docker.com/r/windroseserver/windroseserver)

## License

Apache License 2.0 — see [LICENSE](LICENSE).
