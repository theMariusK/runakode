# runakode

A sandboxed remote code execution service. Accepts source code via an HTTP API, queues it through RabbitMQ, and executes it inside isolated Docker containers using gVisor's `runsc` runtime.

## Architecture

```
Client ──POST /api──▶ API Server ──RabbitMQ──▶ Worker Pool
                          │                        │
                     Reply Queue ◀─────────────────┘
                          │                   Docker + gVisor
                     ◀────┘                   (runsc, no network)
```

The project builds three binaries:

- `runakode` — combined API server + worker pool in one process
- `runakode-api` — standalone API server
- `runakode-worker` — standalone worker pool

## Prerequisites

- Go 1.24+
- Docker with [gVisor](https://gvisor.dev/docs/user_guide/install/) (`runsc` runtime)
- RabbitMQ

## Quick Start

1. Start RabbitMQ:

```bash
docker compose up -d
```

2. Build sandbox images and binaries:

```bash
make all
```

3. Run the combined binary:

```bash
./runakode
```

Or run API and worker separately:

```bash
./runakode-api &
./runakode-worker &
```

## Configuration

Configuration is loaded from `config.yaml` (override with `--config path/to/config.yaml`).

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `address` | string | `127.0.0.1` | API listen address |
| `port` | string | `8080` | API listen port |
| `rabbitmq.url` | string | `amqp://guest:guest@localhost:5672/` | RabbitMQ connection URL |
| `rabbitmq.queue` | string | `runner_jobs` | RabbitMQ queue name |
| `supportedLanguages` | []string | `["python", "go"]` | Allowed languages |
| `apiTimeout` | int | `15` | API response timeout (seconds) |
| `maxWorkers` | int | `2` | Number of worker goroutines |
| `jobCpu` | string | `"1"` | CPU limit per sandbox container |
| `jobMemory` | string | `"1024m"` | Memory limit per sandbox container |
| `jobTimeout` | int | `10` | Sandbox execution timeout (seconds) |
| `outputMaxBytes` | int | `1048576` | Max stdout/stderr bytes captured |
| `sandboxImages` | map | `python: python-runner, go: go-runner` | Docker image per language |

CLI flags `--address` and `--port` override config file values.

## API

### `POST /api`

Execute source code in a sandboxed container.

**Request:**

```json
{
  "language": "python",
  "source_code": "print('hello world')"
}
```

**Response:**

```json
{
  "stdout": "hello world\n",
  "stderr": "",
  "exit_code": 0,
  "timeout": false,
  "exec_time_ms": 342
}
```

**Error responses:**

- `400` — bad request (invalid JSON, empty source code, unsupported language)
- `405` — wrong HTTP method
- `429` — rate limit exceeded
- `504` — execution timed out

### `GET /health`

Health check endpoint.

```json
{
  "status": "ok"
}
```

Returns `"degraded"` if the RabbitMQ connection is closed.

## Supported Languages

| Language | Sandbox Image | Entry File |
|----------|--------------|------------|
| Python | `python-runner` | `main.py` |
| Go | `go-runner` | `main.go` |

## License

Apache License 2.0
