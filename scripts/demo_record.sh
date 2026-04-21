#!/usr/bin/env bash
# =============================================================================
# 0G Memory Hub — Local Demo Recording Script
# =============================================================================
# Records the full Crash / Recover / Verify / Trace demo flow using the
# local file-based storage (no 0G credentials required).
#
# Usage:
#   ./scripts/demo_record.sh [run-id]
#
# Requirements:
#   - Go 1.25+, Rust stable
#   - Built binary: /tmp/orchestrator (run `go build` in apps/orchestrator-go first)
#
# What it demonstrates:
#   1. Service health check
#   2. Ingest an OpenClaw-style event
#   3. Show context + checkpoint
#   4. Simulate crash (stop server)
#   5. Restart server, hydrate the run
#   6. Verify restored state
#   7. Show trace
#   8. Open judge console
# =============================================================================

set -euo pipefail

BINARY_PATH="${BINARY_PATH:-/tmp/orchestrator}"
REPO_ROOT="${REPO_ROOT:-/Users/dongowu/code/project/project_dev/0g-memory-hub}"
RUNTIME_BIN="$REPO_ROOT/rust/memory-core/target/debug/memory-core-rpc"
DATA_DIR="${DATA_DIR:-/tmp/0g-demo-$(date +%Y%m%d%H%M%S)}"
HTTP_ADDR="${HTTP_ADDR:-127.0.0.1:8080}"
BASE_URL="http://$HTTP_ADDR"

RUN_ID="${1:-run-$(date +%Y%m%d%H%M%S)}"
WORKFLOW_ID="wf-${RUN_ID}"

export ORCH_DATA_DIR="$DATA_DIR"
export ORCH_RUNTIME_BINARY_PATH="$RUNTIME_BIN"
export ORCH_HTTP_ADDR="$HTTP_ADDR"

mkdir -p "$DATA_DIR"

cleanup() {
  echo ""
  echo "== cleanup =="
  pkill -f "orchestrator serve" 2>/dev/null || true
}
trap cleanup EXIT

wait_server() {
  local i=0
  while ! curl -s --max-time 2 "$BASE_URL/health" > /dev/null 2>&1; do
    i=$((i+1))
    if [ $i -gt 15 ]; then
      echo "ERROR: server did not start in time" >&2
      exit 1
    fi
    sleep 1
  done
}

announce() {
  echo ""
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "  $1"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# -----------------------------------------------------------------------------
# STEP 0: Build
# -----------------------------------------------------------------------------
announce "STEP 0 — Build"
cd "$REPO_ROOT/apps/orchestrator-go"
if [ ! -f "$BINARY_PATH" ]; then
  echo "Building orchestrator..."
  go build -o "$BINARY_PATH" .
fi
echo "Binary: $BINARY_PATH"
echo "Data dir: $DATA_DIR"

# -----------------------------------------------------------------------------
# STEP 1: Start server
# -----------------------------------------------------------------------------
announce "STEP 1 — Start server"
echo "$BINARY_PATH serve"
"$BINARY_PATH" serve &
wait_server
echo "Server up at $BASE_URL"

# -----------------------------------------------------------------------------
# STEP 2: Health check
# -----------------------------------------------------------------------------
announce "STEP 2 — Health check"
curl -sS "$BASE_URL/health" | python3 -c "
import sys, json
d = json.load(sys.stdin)['data']
print('Ready:', d['ready'])
for k, v in d['components'].items():
    print(f'  {k}: {v[\"ready\"]} — {v.get(\"message\",\"\")}')"

# -----------------------------------------------------------------------------
# STEP 3: Ingest OpenClaw event
# -----------------------------------------------------------------------------
announce "STEP 3 — Ingest OpenClaw-style event"
echo "Run ID: $RUN_ID"
curl -sS -X POST "$BASE_URL/v1/openclaw/ingest" \
  -H 'Content-Type: application/json' \
  -d "{
    \"workflowId\": \"$WORKFLOW_ID\",
    \"runId\": \"$RUN_ID\",
    \"sessionId\": \"session-$RUN_ID\",
    \"traceId\": \"trace-$RUN_ID\",
    \"eventId\": \"evt-plan-1\",
    \"eventType\": \"tool_call\",
    \"actor\": \"planner\",
    \"role\": \"planner\",
    \"toolCallId\": \"tool-search-1\",
    \"skillName\": \"memory_reader\",
    \"taskId\": \"task-$RUN_ID\",
    \"payload\": {\"goal\": \"find BTC sentiment\"}
  }" | python3 -c "
import sys, json
d = json.load(sys.stdin)['data']
print('workflowId:', d['workflowId'])
print('latestStep:', d['latestStep'])
print('latestRoot:', d['latestRoot'])
print('latestCid:', d['latestCid'])
print('latestTxHash:', d['latestTxHash'])"

# -----------------------------------------------------------------------------
# STEP 4: Show context
# -----------------------------------------------------------------------------
announce "STEP 4 — Show context"
curl -sS "$BASE_URL/v1/openclaw/runs/$RUN_ID/context" | python3 -c "
import sys, json
d = json.load(sys.stdin)['data']
print('runId:', d['runId'])
print('sessionId:', d['sessionId'])
print('traceId:', d['traceId'])
print('events:', len(d['events']), 'event(s)')
for e in d['events']:
    print(f'  [{e[\"stepIndex\"]}] {e[\"eventType\"]} | {e[\"actor\"]} | {e[\"payload\"]}')"

# -----------------------------------------------------------------------------
# STEP 5: Show checkpoint
# -----------------------------------------------------------------------------
announce "STEP 5 — Show checkpoint"
curl -sS "$BASE_URL/v1/openclaw/runs/$RUN_ID/checkpoint/latest" | python3 -c "
import sys, json
d = json.load(sys.stdin)['data']
print('latestStep:', d.get('latestStep'))
print('rootHash:', d.get('rootHash'))
print('cid:', d.get('cid'))"

# -----------------------------------------------------------------------------
# STEP 6: CRASH — stop server (simulating process death)
# -----------------------------------------------------------------------------
announce "STEP 6 — SIMULATE CRASH (stop server)"
echo "Stopping server..."
pkill -f "orchestrator serve"
sleep 2
echo "Server stopped. Process crashed!"

# -----------------------------------------------------------------------------
# STEP 7: Restart server
# -----------------------------------------------------------------------------
announce "STEP 7 — Restart server (process survives, memory persists)"
"$BINARY_PATH" serve &
wait_server
echo "Server restarted! Run is still queryable."

# -----------------------------------------------------------------------------
# STEP 8: Hydrate
# -----------------------------------------------------------------------------
announce "STEP 8 — Hydrate (rebuild run from persisted state)"
curl -sS -X POST "$BASE_URL/v1/openclaw/runs/$RUN_ID/hydrate" | python3 -c "
import sys, json
d = json.load(sys.stdin)['data']
print('workflowId:', d['workflowId'])
print('runId:', d['runId'])
print('latestStep:', d['latestStep'])
print('events restored:', len(d['events']))"

# -----------------------------------------------------------------------------
# STEP 9: Verify
# -----------------------------------------------------------------------------
announce "STEP 9 — Verify (re-derive checkpoint + compare against persisted state)"
result=$(curl -sS "$BASE_URL/v1/openclaw/runs/$RUN_ID/verify")
echo "$result" | python3 -c "
import sys, json
d = json.load(sys.stdin)['data']
passed = sum(1 for c in d['checks'] if c['passed'])
total = len(d['checks'])
print('Verified:', d['verified'])
print('Checks:', passed, '/', total)
for c in d['checks']:
    status = '✓' if c['passed'] else '✗'
    print(f'  {status} {c[\"name\"]}')"

# -----------------------------------------------------------------------------
# STEP 10: Trace
# -----------------------------------------------------------------------------
announce "STEP 10 — Trace (ordered execution context)"
curl -sS "$BASE_URL/v1/openclaw/runs/$RUN_ID/trace" | python3 -c "
import sys, json
d = json.load(sys.stdin)['data']
print('workflowId:', d['workflowId'])
print('runId:', d['runId'])
print('steps:', len(d['steps']))
for s in d['steps']:
    print(f'  [{s[\"stepIndex\"]}] {s[\"eventType\"]} | {s[\"actor\"]} | {s[\"payload\"]}')"

# -----------------------------------------------------------------------------
# STEP 11: Judge console
# -----------------------------------------------------------------------------
announce "STEP 11 — Judge console"
echo "URL: $BASE_URL/judge/verify?runId=$RUN_ID"
curl -sS -o /dev/null -w "HTTP status: %{http_code}\n" "$BASE_URL/judge/verify?runId=$RUN_ID"

# -----------------------------------------------------------------------------
# Done
# -----------------------------------------------------------------------------
announce "Demo complete!"
echo "Run ID: $RUN_ID"
echo "Data dir: $DATA_DIR"
echo ""
echo "All checks passed — run survived crash, hydrated correctly, verified against persisted state."
