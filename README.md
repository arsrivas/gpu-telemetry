# GPU Telemetry Pipeline (Elastic, Message-Queue Based)

## Overview

This project implements an **end-to-end elastic GPU telemetry pipeline** for an AI cluster.  
It simulates continuous GPU telemetry ingestion, transport, persistence, and querying using a **custom-built message queue**

The system is designed to demonstrate:

- Scalable ingestion and consumption
- Clean system boundaries
- Correct message-queue semantics
- Kubernetes-native deployment
- Idiomatic Go practices
- Thoughtful and transparent use of AI assistance

---

## High-Level Architecture (HLD)

![High-Level Architecture](docs/diagram/sequnce.svg)

### Architectural Notes

- **Streamers** and **Collectors** are stateless → scale horizontally
- **Message Queue** provides at-least-once delivery semantics
- **PostgreSQL** is the system of record
- **TelemetryRetention** cron job cleans up old entries periodically
- **API Gateway** reads directly from DB (not from collectors)
- **Helm + Kubernetes** manage lifecycle and scaling
- **Kind** is used for local cluster simulation

---

## System Components

### 1. Telemetry Streamer

- Reads telemetry from a CSV file
- Each CSV row is an independent telemetry datapoint
- CSV timestamp is replaced with **current ingestion time**
- Streams telemetry continuously in a loop
- Publishes messages to the custom MQ
- Horizontally scalable


### 2. Custom Message Queue (MQ)

- HTTP-based message queue
- Supports:
  - `Enqueue`
  - `Poll(batchSize)`
  - `Ack(messageID)`
- Backpressure handled via polling
- Deduplication handled at storage layer

#### Delivery Semantics
- At-least-once delivery
- Messages are acknowledged only after successful persistence
- Unacknowledged messages are eligible for redelivery

#### Failure Handling
- Collector crash before Ack → message is retried
- Collector crash after Ack → safe due to idempotent DB inserts
- MQ restart → in-memory state resets

#### Scale Limits
- Designed and tested up to 10 Streamers and 10 Collectors
- Horizontal scaling via Kubernetes replicas


### 3. Telemetry Collector

- Polls messages from MQ
- Persists telemetry into PostgreSQL
- Acknowledges messages **only after successful persistence**
- Idempotent inserts → safe to retry
- Horizontally scalable

#### Why Polling Instead of Push / PubSub

Collectors use a polling model instead of push-based delivery.

Reasons:
- Collectors control backpressure
- Avoids overloading slow consumers
- Simpler failure recovery
- Easier horizontal scaling

Polling ensures the consumer dictates throughput, not the queue.

#### Collector Autoscaling Strategy (Future Scope)

The collector architecture is intentionally designed to support queue-driven autoscaling in the future.

If queue depth grows faster than collector polling capacity, an autoscaling mechanism can be introduced where:

- Queue depth or processing lag would be surfaced as a metric
- Collectors could be horizontally scaled using an HPA or event-driven autoscaler
- Scaling decisions would be based on backlog growth rather than CPU usage

This approach preserves backpressure control while allowing the system to adapt to sustained increases in load without redesigning the collector.

### 4. Storage Layer

- PostgreSQL-backed persistent store
- Abstracted via `storage.Store` interface
- Shared by Collector and API
- Deduplication via unique telemetry IDs
- Clean separation between interface and implementation


### 5. API Gateway

- Stateless REST API
- Reads telemetry directly from PostgreSQL
- Structured logging using `zap`
- Correct HTTP semantics and error handling
- Readiness probe via `/healthz`
- Swagger/OpenAPI auto-generated

### 6. Telemetry Retention
- CronJob to periodically clean old entries from DB

---
## Sample User Workflow

1. Deploy the applications

2. Streamers begin publishing telemetry from CSV files

3. Collectors consume and persist telemetry into PostgreSQL

4. User queries available GPUs:
   ```GET /api/v1/gpus```

5. User queries telemetry for a GPU:
   ```GET /api/v1/gpus/{id}/telemetry?start_time=...&end_time=...```

6. Results are returned ordered by timestamp

---

## API Endpoints

| Endpoint | Description |
|--------|-------------|
| GET /api/v1/gpus | List all GPUs |
| GET /api/v1/gpus/{id}/telemetry | Query telemetry by GPU |
| GET /healthz | Readiness probe |

### Telemetry Query Parameters

- `start_time` – RFC3339 timestamp (UTC), optional
- `end_time` – RFC3339 timestamp (UTC), optional

Telemetry is returned ordered by timestamp.

---

## OpenAPI (Swagger)

The OpenAPI specification is **auto-generated** using `swaggo/swag`.

### Generate OpenAPI Spec

```bash
make swagger
```
### Once the API server is running:

```
http://<IP>:8081/swagger/index.html
```

## Project Structure
```
.
├── build/              Dockerfiles
│   ├── Dockerfile.api
│   ├── Dockerfile.collector
│   ├── Dockerfile.mq
│   └── Dockerfile.streamer
├── cmd/
│   ├── api/            API server entrypoint
│   ├── collector/      Collector entrypoint
│   ├── mq/             Message queue entrypoint
│   └── streamer/       Streamer entrypoint
├── config/             Config loading
├── deployment/
│   └── helm/           Helm chart
├── docs/               Swagger output
├── model/              Data model struct
├── pkg/
│   ├── api/            HTTP handlers & routing
│   ├── client/         MQ client
|   ├── collector/      Telemetry collector
|   ├── mq/        Message queue
│   ├── storage/        DB
|   ├── streamer/       Telemetry streamer
|   ├── util/           Zap logger wrapper 
├── Makefile
└── README.md
```

## Build & Run (Local) - QuickStart User install flow

### Prerequisites

The project has been tested with the following tool versions.  
Other compatible versions may work as well.

- **Go** ≥ 1.24  
- **Docker** ≥ 25.x  
- **kubectl** ≥ 1.34  
- **Helm** ≥ 3.14  
- **Kind** ≥ 0.31.0  

Please ensure the tools are installed and available in your `PATH`.

Installation guides:
- https://go.dev/doc/install
- https://docs.docker.com/get-docker/
- https://kubernetes.io/docs/tasks/tools/
- https://helm.sh/docs/intro/install/
- https://kind.sigs.k8s.io/docs/user/quick-start/

### Environment validation

```
make preflight
```
### Create kind cluster (One-time setup)
Create cluster
```
make kind-create
```
### One-command Quick Start

Assuming a local Kind cluster already exists, the entire build and deployment flow can be executed
using a single command

```bash
make all
```

### Unit testing
Run unit tests
```
make test
```
Run unit tests with coverage
```
make coverage
```

### Build and load docker image
Builds Docker images:

- API
- Collector
- Streamer
- MQ
```
make docker-build
```
Load docker images into Kind
```
make kind-load
```

### Deploy using Helm
Deploy chart to `telemetry` namespace
```
make deploy
```
### Accessing the API locally

The API service can be accessed via port-forwarding:

```bash
kubectl port-forward svc/telemetry-api 8081:8081 -n telemetry
```
Swagger
```
http://localhost:8081/swagger/index.html
```
---
## Operations
### Kind cluster management
Create cluster (cluster config can be found in `deployment` folder)
```
make kind-create
```
Delete cluster
```
make kind-delete
```
### Docker Image management
Remove all project Docker images:
```
make docker-clean
```
### Helm app lifecycle
Uninstall Helm release
```
make undeploy
```
Upgrade
```
make deploy
``` 
---
## Helm Configuration
Application configuration is defined in `values.yaml` and injected into pods
as environment variables via the Helm chart.

For local development, a `.env.example` file is provided for convenience.
### Scaling
Collectors and Streamers are stateless and can be scaled by adjusting
the replica count in `values.yaml`:
```
collector:
  replicas: 2

streamer:
  replicas: 1
```
### Telemetry retention
Periodic cleanup of old telemetry data via `CronJob`
```
retention:
  enabled: true
  retentionDays: 1
```
---
## Observability

- Structured logging using `zap`
- Component-level log separation
- Errors include contextual fields (GPU ID, message ID, etc.)
- Kubernetes readiness probe exposed via /healthz

---
## Troubleshooting

### Resetting PostgreSQL data (development only)

If you need to reset the local PostgreSQL state during development,
the persistent volume claim can be deleted:

```bash
kubectl delete pvc data-telemetry-postgres-0 -n telemetry
```
---
## Design Tradeoffs

| Design Area | Decision (Chosen) | Alternative | Why This Choice | What We Deliberately Give Up |
|------------|------------------|-------------|-----------------|------------------------------|
| **Queue Protocol** | HTTP-based Queue | gRPC / Binary Protocol | • Rapid prototyping<br>• Apt for assignment use case | • No streaming support<br>• No long-lived connections |
| **MQ Topology** | Single-instance MQ | Distributed MQ | • Avoids leader election complexity<br>• Avoids coordination and partitioning | • Single point of failure<br>• Limited horizontal scalability |
| **Queue Durability** | In-memory Queue | Durable Queue | • Simple implementation<br>• Faster message access | • Messages lost on MQ restart<br>• No replay capability |
| **Delivery Semantics** | At-least-once Delivery | Exactly-once Delivery | • Prevents data loss under failure<br>• Simpler delivery semantics | • Duplicate messages possible |
| **Deduplication Strategy** | Storage-level Deduplication | Queue-level Deduplication | • Centralizes system of record<br>• Simplifies MQ design | • Database overhead |
| **Consumption Model** | Pull-based Consumption | Push-based Delivery | • Collectors control backpressure<br>• Predictable load handling | • Inefficient polling when queue is idle |
| **Processing Mode** | Batch Polling | Single-message Processing | • Higher throughput<br>• Fewer database transactions | • Larger retry scope on partial failures |
| **PostgreSQL Deployment** | Custom PostgreSQL | Bitnami / Managed Chart | • Rapid prototyping<br>• Explicit schema ownership | • No built-in HA, backups, or upgrades<br>• More operational responsibility |
| **Ingestion Control** | No Rate Limiting | Rate-limited Ingestion | • Simple streamer implementation<br>• Maximizes ingestion speed | • Risk of downstream pressure<br>• Relies on MQ and collectors to absorb bursts |
| **Replay Capability** | No Replay / Backfill | Replay-enabled Pipeline | • Reduced storage and complexity<br>• Clear data lifecycle | • Historical reprocessing not possible<br>• Harder recovery from logical bugs |
| **Autoscaling** | No HPA | HPA-based Autoscaling | • Predictable and deterministic behavior<br>• Easy to reason about scaling | • Manual intervention required<br>• No dynamic response to load |
| **Observability** | Logs-first Observability | Metrics & Tracing | • Simple setup<br>• Effective debugging | • Limited visibility into latency and throughput |
| **Telemetry Source** | CSV per Streamer Replica | Real-time Telemetry Source | • Each replica acts as independent data source<br>• No coordination required | • Not representative of real-time telemetry |

---