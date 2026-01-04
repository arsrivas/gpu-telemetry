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

![High-Level Architecture](docs/diagram/hld.svg)

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

- `start_time` – Unix timestamp (seconds), optional
- `end_time` – Unix timestamp (seconds), optional

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
http://localhost:8081/swagger/index.html
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

## Build & Run (Local)

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

### Unit testing
Run unit tests
```
make test
```
Run unit tests with coverage
```
make coverage
```
### Kind cluster
Create cluster
```
make kind-create
```
Delete cluster
```
make kind-delete
```

### Docker image build/clean
Builds Docker images:

- API
- Collector
- Streamer
- MQ
```
make docker-build
```
Remove Docker images:
```
make docker-clean
```
### Load Docker image into Kind
Load docker images into Kind
```
make kind-load
```

### Helm Deployment
Deploy chart to `telemetry` namespace
```
make deploy
```
Uninstall chart
```
make undeploy
```
### Accessing the API locally

The API service can be accessed via port-forwarding:

```bash
kubectl port-forward svc/telemetry-api 8081:8081 -n telemetry
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

### HTTP-based Queue vs gRPC / Binary Protocol
**Pros**
- Rapid prototyping
- Apt for assignment usecase

**Cons**
- No streaming or long-lived connections


### Single-instance MQ vs Distributed MQ
**Pros**
- Avoids complexity of leader election, coordination and partitioning

**Cons**
- Single point of failure
- Limited horizontal scalability


### In-memory Queue vs Durable Queue
**Pros**
- Simple implementation
- Faster message access

**Cons**
- Messages lost on MQ microservice restart
- No replay capability


### At-least-once vs Exactly-once Delivery
**Pros**
- Prevents data loss under failure
- Simpler and more reliable semantics

**Cons**
- Duplicate messages possible

### Deduplication at Storage vs Queue
**Pros**
- Centralizes system of record
- Simplifies MQ design

**Cons**
- Additional database overhead


### Pull-based Consumption vs Push-based Delivery
**Pros**
- Collectors control backpressure
- Predictable load handling

**Cons**
- Inefficient when queue is idle


### Batch Polling vs Single-message Processing
**Pros**
- Higher throughput
- Fewer database transactions

**Cons**
- Larger retry scope on partial failures

### Custom PostgreSQL vs Bitnami / Managed Chart
**Pros**
- Rapid prototyping and avoids legacy bitnami charts
- Explicit schema ownership

**Cons**
- No built-in HA, backups, or upgrades
- More operational responsibility


### No Rate Limiting at Ingestion
**Pros**
- Simple streamer implementation
- Maximizes ingestion speed

**Cons**
- Risk of downstream pressure
- Relies on MQ and collectors to absorb bursts

### No Replay / Backfill Support
**Pros**
- Reduced storage and complexity
- Clear data lifecycle

**Cons**
- Historical reprocessing not possible
- Harder to recover from logical bugs

### No Autoscaling (HPA)
**Pros**
- Predictable and deterministic behavior
- Easy to reason about scaling

**Cons**
- Manual intervention required
- No dynamic response to load

### Logs-first Observability vs Metrics & Tracing
**Pros**
- Simple setup, effective debugging

**Cons**
- Limited visibility into latency and throughput

### CSV telemetry data ingestion
- Each streamer replica reads own CSV and acts as source of data

---
## Use of AI Assistance

AI assistance was used extensively during development.

### Project Bootstrapping
- Used AI to generate initial project structure
- Prompts focused on idiomatic Go layout and clean boundaries

### Code Development
- AI used to scaffold handlers, MQ interfaces, and storage abstractions
- Manual intervention required for:
  - Correct acknowledgment semantics
  - Idempotent DB design
  - Concurrency edge cases

### Unit Tests
- AI assisted in generating table-driven tests
- Manual refinement needed for mock behavior and edge cases

### Build & Deployment
- AI used to draft Makefile and Helm scaffolding
- Manual tuning required for Windows/Linux compatibility and Kind

### Where AI Fell Short
- Initial MQ designs lacked proper failure semantics
- Some generated tests were not working and required manual fixes