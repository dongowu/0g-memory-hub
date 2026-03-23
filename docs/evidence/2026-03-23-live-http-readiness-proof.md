# Live HTTP Readiness Proof — March 23, 2026

## Goal

Verify that the Go orchestrator can run as a long-lived HTTP service and expose a structured readiness report for the OpenClaw-facing API.

This proof covers:

1. the `serve` command booting successfully,
2. the HTTP server binding locally,
3. the `/health` endpoint returning structured readiness JSON,
4. the Rust runtime being probed successfully through the readiness path.

This proof does **not** claim a live 0G upload or live chain anchor write. The current readiness endpoint verifies:

- runtime via a real stdio probe to `memory-core-rpc`,
- storage via upload-capable config validation,
- anchor via config validation and optional-component reporting.

## Environment

- Date: **March 23, 2026**
- Host bind: `127.0.0.1:18080`
- Runtime binary:
  `/Users/dongowu/code/project/project_dev/0g-memory-hub/rust/memory-core/target/debug/memory-core-rpc`

## Commands

Build the Rust runtime:

```bash
cd rust/memory-core
cargo build --bin memory-core-rpc
```

Build the Go orchestrator binary:

```bash
cd apps/orchestrator-go
GOENV=off GOCACHE=/tmp/go-build-0g-memory-hub-serve2 \
  GOPROXY=https://goproxy.cn,direct GOSUMDB=off \
  /Users/dongowu/.local/share/mise/installs/go/1.26.0/bin/go build -o /tmp/orchestrator-go
```

Run the service locally:

```bash
ORCH_RUNTIME_BINARY_PATH=/Users/dongowu/code/project/project_dev/0g-memory-hub/rust/memory-core/target/debug/memory-core-rpc \
ORCH_CHAIN_PRIVATE_KEY=1111111111111111111111111111111111111111111111111111111111111111 \
ORCH_HTTP_ADDR=127.0.0.1:18080 \
ORCH_DATA_DIR=.orchestrator-demo \
/tmp/orchestrator-go serve
```

Probe readiness:

```bash
curl -i http://127.0.0.1:18080/health
```

## Observed Response

Server boot log:

```text
http_addr=127.0.0.1:18080
```

Observed readiness response:

```http
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "data": {
    "ready": true,
    "components": {
      "anchor": {
        "ready": true,
        "required": false,
        "message": "ok"
      },
      "runtime": {
        "ready": true,
        "required": true,
        "message": "ok"
      },
      "storage": {
        "ready": true,
        "required": true,
        "message": "ok"
      }
    }
  }
}
```

Timestamp observed in the live response:

```text
2026-03-23T04:26:24Z
```

## Conclusion

The orchestrator successfully ran as a local HTTP service and returned a structured readiness report. The runtime probe passed through the real Rust stdio bridge, which confirms the long-lived service path can see and use the runtime dependency before serving requests.
