<div align="center">

```
  ██████╗  ██████╗ ██╗      █████╗ ██╗   ██╗███╗   ██╗ ██████╗██╗  ██╗
 ██╔════╝ ██╔═══██╗██║     ██╔══██╗██║   ██║████╗  ██║██╔════╝██║  ██║
 ██║  ███╗██║   ██║██║     ███████║██║   ██║██╔██╗ ██║██║     ███████║
 ██║   ██║██║   ██║██║     ██╔══██║██║   ██║██║╚██╗██║██║     ██╔══██║
 ╚██████╔╝╚██████╔╝███████╗██║  ██║╚██████╔╝██║ ╚████║╚██████╗██║  ██║
  ╚═════╝  ╚═════╝ ╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═══╝ ╚═════╝╚═╝  ╚═╝
```

**Drop a zip. Get a live URL.**

A self-hosted deployment platform for Node.js and Next.js projects — built from scratch in Go.

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![Next.js](https://img.shields.io/badge/Frontend-Next.js_15-black?style=flat-square&logo=next.js)](https://nextjs.org)
[![PostgreSQL](https://img.shields.io/badge/Database-PostgreSQL-336791?style=flat-square&logo=postgresql)](https://postgresql.org)
[![License](https://img.shields.io/badge/License-MIT-purple?style=flat-square)](LICENSE)

[Features](#features) · [Architecture](#architecture) · [Getting Started](#getting-started) · [API](#api-reference) · [Roadmap](#roadmap)

</div>

---

## What is this?

**golaunch** is a mini Vercel/Railway clone I built to learn Go systems programming. You upload a `.zip` of your Node.js or Next.js project through a slick UI, and the platform:

1. Extracts and stores your project
2. Detects the framework automatically
3. Runs `npm install` + `next build` (if needed)
4. Spawns the process and streams build logs live over SSE
5. Hands you a live URL

The entire backend is Go — no third-party queues, no cloud services, no Docker (yet). Just raw Go concurrency doing real work.

---

## Features

- **Zero-config deploys** — upload a zip, get a URL. That's it.
- **Live log streaming** — watch every line of output in real time via Server-Sent Events
- **Framework detection** — automatically detects Next.js vs plain Node.js and runs the right build pipeline
- **Built-in worker queue** — hand-rolled goroutine worker pool with retry logic and backoff. No RabbitMQ, no Redis.
- **Clean Architecture / DDD** — domain, application, and infrastructure layers properly separated
- **Session persistence** — refresh mid-deploy? The frontend reconnects and re-runs automatically
- **Atomic port allocation** — concurrent deploys get unique ports via atomic counter. No races.
- **Postgres-backed** — full project lifecycle tracked in the DB (pending → building → running → stopped/failed)

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Next.js Frontend                        │
│   Upload → Configure → Deploy  (SSE log stream, session persist)│
└─────────────────────────┬───────────────────────────────────────┘
                          │ HTTP / SSE
┌─────────────────────────▼───────────────────────────────────────┐
│                      Go Backend                                  │
│                                                                  │
│  ┌─────────────┐   ┌──────────────┐   ┌──────────────────────┐ │
│  │   Handlers  │──▶│  Use Cases   │──▶│   Worker Pool        │ │
│  │  (HTTP/SSE) │   │  (App Layer) │   │  (4 goroutines)      │ │
│  └─────────────┘   └──────┬───────┘   └──────────┬───────────┘ │
│                            │                       │             │
│                    ┌───────▼───────┐    ┌──────────▼──────────┐ │
│                    │  LogRegistry  │    │   ProjectRunner     │ │
│                    │  (sync.Map)   │    │  npm install/build  │ │
│                    └───────────────┘    │  process streaming  │ │
│                                        └─────────────────────┘ │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    PostgreSQL                             │   │
│  │   projects: id, unique_key, status, port, source_location│   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Layer breakdown

```
internal/
├── domain/                  # Pure business logic — no imports from infra
│   ├── entities/            # Project entity, status types, validation
│   └── repository/          # Repository interfaces (contracts)
│
├── application/             # Use cases — orchestrate domain + infra
│   ├── upload_project.go    # Upload, extract, persist
│   ├── run_project.go       # Submit to queue, return log channel
│   ├── project_runner.go    # npm install / build / start logic
│   └── log_registry.go      # Thread-safe projectID → chan LogLine map
│
├── infrastructure/          # Concrete implementations
│   ├── database/postgres/   # pgx repository implementations
│   ├── http/handlers/       # Upload + Run HTTP handlers
│   ├── storage/             # Zip save + extract
│   └── config/              # Config loading
│
└── queue/                   # Hand-rolled worker pool
    ├── types.go             # Job struct
    └── workerpool.go        # Goroutine pool, retry with exponential backoff
```

---

## The Queue

Instead of pulling in RabbitMQ or Redis, I wrote a goroutine-based worker pool from scratch.

```go
pool := queue.NewWorkerPool(4, processor)
pool.Start()
defer pool.ShutDown()
```

**How it works:**

- `NewWorkerPool(n, processor)` creates a buffered job channel (cap 200) and `n` goroutines
- Each worker loops on `select { case job := <-wp.jobs }` — blocking until work arrives
- On failure: exponential backoff retry (up to 3 attempts: 2s, 4s, 6s)
- On shutdown: context cancel signals all workers to drain and exit cleanly
- The processor is a closure injected at startup — captures the DB repo, runner, and log registry

**Why not just `go func()`?** Unbounded goroutine spawning under load would fork hundreds of `npm install` processes simultaneously. The pool gates that to exactly N concurrent deploys.

---

## Log Streaming

The SSE pipeline is the interesting part. Here's the flow:

```
POST /run/{projectID}
       │
       ▼
RunProjectUseCase.Execute()
  1. Creates chan LogLine (buffered 64)
  2. Registers channel in LogRegistry (sync.Map keyed by projectID)
  3. Submits Job to WorkerPool
  4. Returns channel immediately ←── handler starts SSE loop here
       │
       ▼  (concurrently, in a worker goroutine)
processor(ctx, job)
  1. Looks up channel from registry
  2. defer close(logCh)        ←── this is how the SSE handler knows to stop
  3. runner.Run(ctx, path, port, send)
       ├── npm install  (buffered — only streams on failure)
       ├── next build   (buffered)
       └── npm start    (streamed live, line by line)
```

The `send` function inside the worker uses `select` with `ctx.Done()` — so if the client closes the browser tab, the context cancels, `exec.CommandContext` kills the child process, and the goroutine exits cleanly.

---

## Getting Started

### Prerequisites

- Go 1.22+
- Node.js 18+ (on the server — it runs your uploaded projects)
- PostgreSQL
- `golang-migrate` CLI

### 1. Clone and configure

```bash
git clone https://github.com/yourhandle/golaunch.git
cd golaunch
cp cmd/configuration.json.example cmd/configuration.json
# edit configuration.json with your DB credentials and port
```

### 2. Run migrations

```bash
migrate -path ./migrations -database "postgres://user:pass@localhost:5432/golaunch?sslmode=disable" up
```

### 3. Start the backend

```bash
go run ./cmd/main.go
```

### 4. Start the frontend

```bash
cd frontend
npm install
npm run dev
```

Open `http://localhost:3000`, upload a zip, watch it deploy.

---

## API Reference

### `POST /upload`

Upload a `.zip` of your project.

**Request:** `multipart/form-data` with field `file`

**Response:**
```json
{
  "ProjectID": "7a2cad8b-aa2f-49cc-8e7f-13ca7b7fa058",
  "UniqueKey": "64ed80f9-c8ba-49b5-bbd8-98a5575fd5c4"
}
```

---

### `GET /run/{projectID}`

Start the project and stream logs. Returns an SSE stream.

**Events:**

| Event | Description |
|-------|-------------|
| `stdout` | Standard output from the process |
| `stderr` | Standard error (warnings, errors) |
| `error` | Runner-level error (project not found, queue full, etc.) |
| `done` | Process finished |

**Example with curl:**
```bash
curl -N http://localhost:5000/run/7a2cad8b-aa2f-49cc-8e7f-13ca7b7fa058
```

```
event: stdout
data: [runner] running npm install...

event: stdout
data: [runner] Next.js detected — running npm run build...

event: stdout
data: [runner] process started on port 3001 (pid 9842)

event: stdout
data:  ✓ Ready in 892ms

event: done
data: process finished
```

---

## Tech Stack

| Layer | Tech |
|-------|------|
| Backend language | Go 1.22 |
| HTTP | `net/http` stdlib (no framework) |
| Database | PostgreSQL via `pgx/v5` |
| Migrations | `golang-migrate` |
| Queue | Hand-rolled goroutine worker pool |
| Log streaming | Server-Sent Events (SSE) |
| Frontend | Next.js 15 App Router |
| Styling | Tailwind CSS v4 |
| Animations | Framer Motion |
| Process management | `os/exec` with `exec.CommandContext` |

---

## What I Learned Building This

This was a deliberate learning project. Things that clicked while building it:

**Go concurrency** — goroutines, channels, `select`, `sync.Map`, atomic counters. The log streaming pipeline specifically — a channel as the bridge between a worker goroutine and an HTTP handler, with `defer close(ch)` as the done signal.

**Clean Architecture in Go** — keeping domain logic free of infrastructure imports. The `repository.ProjectRepository` interface means the use cases don't know or care whether the DB is Postgres, SQLite, or an in-memory map.

**SSE vs WebSocket** — SSE is one-way and works over plain HTTP/1.1. Perfect for log streaming where the client only needs to receive. WebSocket is bidirectional but heavier. SSE was the right call here.

**Process lifecycle** — `exec.CommandContext` ties a child process to a Go context. When the context cancels (client disconnects), the OS kills the child. Clean.

**Why buffer size matters** — the log channel has a buffer of 64. Without it, if the SSE writer is slightly slow, the worker goroutine blocks on every line. The buffer decouples them.

---

## Roadmap

- [ ] GitHub repo URL support (clone instead of upload)
- [ ] Docker container isolation per project
- [ ] Subdomain routing (`{project}.golaunch.dev`)
- [ ] Process management (stop/restart running projects)
- [ ] Environment variable injection at runtime
- [ ] Multi-user support with auth
- [ ] `GET /projects/{id}/status` polling endpoint
- [ ] Metrics endpoint (`/stats` — queue depth, active workers, deploy counts)

---

## Project Structure

```
golaunch/
├── cmd/
│   ├── main.go                    # Entry point — wires everything, starts pool
│   └── configuration.json         # Server + DB config
├── internal/
│   ├── application/
│   │   ├── upload_project.go      # Upload use case
│   │   ├── run_project.go         # Run use case (queue submit + channel return)
│   │   ├── project_runner.go      # npm/node process management
│   │   └── log_registry.go        # projectID → log channel registry
│   ├── domain/
│   │   ├── entities/
│   │   │   ├── project.go         # Project entity + status types
│   │   │   └── error.go           # Domain errors
│   │   └── repository/
│   │       ├── project_repo.go    # Repository interface
│   │       └── storage_repo.go    # Storage interface
│   ├── infrastructure/
│   │   ├── config/config.go       # Config loader
│   │   ├── database/postgres/     # pgx implementations
│   │   ├── http/
│   │   │   ├── server.go          # Route wiring + CORS
│   │   │   └── handlers/          # Upload + Run handlers
│   │   ├── storage/               # Zip save + extract
│   │   └── utils/                 # ID generation
│   └── queue/
│       ├── types.go               # Job struct
│       └── workerpool.go          # Worker pool implementation
├── migrations/                    # SQL migration files
└── frontend/                      # Next.js app
    ├── app/
    ├── components/
    │   ├── deploy-wizard.tsx      # 3-step wizard orchestrator
    │   └── steps/
    │       ├── upload-step.tsx    # Drag & drop zip upload
    │       ├── configure-step.tsx # Name + env vars
    │       └── deploy-step.tsx    # Live log stream + pipeline viz
    └── lib/
        └── session.ts             # localStorage deploy session persistence
```

---

<div align="center">

Built by Parham Mohebbi · learning Go by shipping things

</div>