# Bugboy Go

Bugboy Go is a Go-based companion to the original Next.js Bugboy app. It exposes realistic failure scenarios to stress-test tools like Bugstack that capture runtime issues, gather context, propose fixes, run tests, and ship remediation automatically.

## Features

- Panic routes with stack traces:
  - `GET /bugs/panic/nil-pointer`
  - `GET /bugs/panic/index-out-of-range`
  - `GET /bugs/panic/divide-by-zero`
  - `GET /bugs/panic/nil-map-write`
- Handled application error routes:
  - `GET /bugs/error/db-timeout`
  - `GET /bugs/error/external-api`
  - `GET /bugs/error/json-parse`
- Async runtime issue route:
  - `GET /bugs/background/panic` (panic is recovered and logged in a goroutine)
- Fatal crash route:
  - `GET /bugs/fatal/unhandled-goroutine-panic` (unrecovered goroutine panic that terminates the process)
- Context-rich JSON responses with:
  - `request_id`
  - `timestamp`
  - `details` payload for root-cause context
- Browser UI at `GET /` to trigger each scenario quickly
- Health endpoint at `GET /healthz`
- Optional panic recovery switch:
  - `BUGBOY_RECOVER_PANICS=false` disables Bugboy's panic recovery middleware so request panics bubble to Go's default `net/http` panic handling

## Run

```bash
go run ./cmd/server
```

Server default:

- `http://localhost:8080`

Set custom port:

```bash
PORT=9090 go run ./cmd/server
```

Run with raw (non-Bugboy-recovered) handler panics:

```bash
BUGBOY_RECOVER_PANICS=false go run ./cmd/server
```

## Test

```bash
go test ./...
```

## BugStack SDK Integration (Go)

This app has built-in integration points for BugStack.

1. Install the SDK:

```bash
go get github.com/MasonBachmann7/bugstack-go
```

2. Set your API key:

```bash
BUGSTACK_API_KEY=bs_live_...
```

3. Run with the BugStack build tag enabled:

```bash
go run -tags bugstack ./cmd/server
```

### How it is wired

- Startup initializes BugStack from env and flushes on shutdown.
- Handled errors (`db-timeout`, `external-api`, `json-parse`) call `CaptureError`.
- Recovered panics call `CaptureError` via panic capture hooks.
- If you run without `-tags bugstack`, the integration is a no-op stub so the app still builds.

## Quick Smoke Checks

```bash
curl -i http://localhost:8080/healthz
curl -i http://localhost:8080/bugs/panic/nil-pointer
curl -i http://localhost:8080/bugs/error/db-timeout
curl -i http://localhost:8080/bugs/error/external-api
curl -i http://localhost:8080/bugs/background/panic
curl -i http://localhost:8080/bugs/fatal/unhandled-goroutine-panic
```
