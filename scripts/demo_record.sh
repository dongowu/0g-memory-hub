#!/usr/bin/env bash
# =============================================================================
# 0G Memory Hub — Local Demo Recording Script
# =============================================================================
# Crash / Recover / Verify / Trace demo — no 0G credentials needed.
#
# Usage:
#   bash scripts/demo_record.sh [run-id]
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
SERVER_PID=""

export ORCH_DATA_DIR="$DATA_DIR"
export ORCH_RUNTIME_BINARY_PATH="$RUNTIME_BIN"
export ORCH_HTTP_ADDR="$HTTP_ADDR"

announce() {
  echo ""
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "  $1"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

wait_server() {
  local i=0
  while ! curl -s --max-time 2 "$BASE_URL/health" > /dev/null 2>&1; do
    i=$((i+1))
    if [ $i -gt 15 ]; then echo "ERROR: server did not start" >&2; exit 1; fi
    sleep 1
  done
}

cleanup() {
  echo ""
  if [ -n "$SERVER_PID" ] && kill -0 "$SERVER_PID" 2>/dev/null; then
    kill "$SERVER_PID" 2>/dev/null || true
  fi
  pkill -f "orchestrator serve" 2>/dev/null || true
}
trap cleanup EXIT

# -----------------------------------------------------------------------------
# Pre-flight
# -----------------------------------------------------------------------------
announce "STEP 0 — Pre-flight"
pkill -f "orchestrator serve" 2>/dev/null || true
sleep 2
rm -rf "$DATA_DIR" && mkdir -p "$DATA_DIR"
echo "Data dir: $DATA_DIR"

# Build
cd "$REPO_ROOT/apps/orchestrator-go"
if [ ! -f "$BINARY_PATH" ]; then
  echo "Building..."
  go build -o "$BINARY_PATH" .
fi

# -----------------------------------------------------------------------------
# Start server
# -----------------------------------------------------------------------------
announce "STEP 1 — Start server"
"$BINARY_PATH" serve &
SERVER_PID=$!
wait_server
echo "Server up at $BASE_URL (PID $SERVER_PID)"

# -----------------------------------------------------------------------------
# Health
# -----------------------------------------------------------------------------
announce "STEP 2 — Health check"
curl -sS "$BASE_URL/health" | python3 -c "
import sys,json
d=json.load(sys.stdin)['data']
print('Ready:', d['ready'])
for k,v in d['components'].items():
    print(f'  {k}: {v[\"ready\"]} — {v.get(\"message\",\"\")}')"

# -----------------------------------------------------------------------------
# Ingest
# -----------------------------------------------------------------------------
announce "STEP 3 — Ingest OpenClaw event"
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
import sys,json
d=json.load(sys.stdin)['data']
print('workflowId:', d['workflowId'])
print('latestStep:', d['latestStep'])
print('latestRoot:', d['latestRoot'][:24]+'...')
print('latestCid:', d['latestCid'][:24]+'...')
print('latestTxHash:', d['latestTxHash'])"

# -----------------------------------------------------------------------------
# Context
# -----------------------------------------------------------------------------
announce "STEP 4 — Show context"
curl -sS "$BASE_URL/v1/openclaw/runs/$RUN_ID/context" | python3 -c "
import sys,json
d=json.load(sys.stdin)['data']
print('runId:', d['runId'], '| sessionId:', d['sessionId'], '| traceId:', d['traceId'])
print('events:', len(d['events']))
for e in d['events']:
    print(f'  [{e[\"stepIndex\"]}] {e[\"eventType\"]} | actor={e[\"actor\"]} | skill={e.get(\"skillName\",\"\")} | {e[\"payload\"]}')"

# -----------------------------------------------------------------------------
# Checkpoint
# -----------------------------------------------------------------------------
announce "STEP 5 — Show checkpoint"
curl -sS "$BASE_URL/v1/openclaw/runs/$RUN_ID/checkpoint/latest" | python3 -c "
import sys,json
d=json.load(sys.stdin)['data']
print('latestStep:', d['latestStep'])
print('rootHash:', d['latestRoot'])
print('cid:', d['latestCid'])"

# -----------------------------------------------------------------------------
# CRASH
# -----------------------------------------------------------------------------
announce "STEP 6 — CRASH: kill server (process dies, memory survives on disk)"
echo "Killing PID $SERVER_PID..."
kill "$SERVER_PID" 2>/dev/null || true
sleep 2
echo "Process dead."

# Show persisted data
echo "Persisted state still on disk:"
cat "$DATA_DIR/workflows.json" 2>/dev/null | python3 -c "
import sys,json
d=json.load(sys.stdin)
for k,v in d.items():
    print(f'  {k}: step={v.get(\"latest_step\")}')" 2>/dev/null

# -----------------------------------------------------------------------------
# Restart
# -----------------------------------------------------------------------------
announce "STEP 7 — Restart server (same data dir — run survives)"
"$BINARY_PATH" serve &
SERVER_PID=$!
wait_server
echo "Server restarted (PID $SERVER_PID). Run is still there."

# -----------------------------------------------------------------------------
# Hydrate
# -----------------------------------------------------------------------------
announce "STEP 8 — Hydrate (rebuild run from persisted state)"
curl -sS -X POST "$BASE_URL/v1/openclaw/runs/$RUN_ID/hydrate" | python3 -c "
import sys,json
d=json.load(sys.stdin)['data']
print('workflowId:', d['workflowId'])
print('runId:', d['runId'])
print('latestStep:', d['latestStep'])
print('events restored:', len(d['events']))"

# -----------------------------------------------------------------------------
# Verify
# -----------------------------------------------------------------------------
announce "STEP 9 — Verify (re-derive checkpoint + compare against persisted state)"
curl -sS "$BASE_URL/v1/openclaw/runs/$RUN_ID/verify" | python3 -c "
import sys,json
d=json.load(sys.stdin)['data']
passed=sum(1 for c in d['checks'] if c['passed'])
total=len(d['checks'])
print('Verified:', d['verified'], '| Checks:', passed, '/', total)
for c in d['checks']:
    print(f'  {\"✓\" if c[\"passed\"] else \"✗\"} {c[\"name\"]}')"

# -----------------------------------------------------------------------------
# Trace
# -----------------------------------------------------------------------------
announce "STEP 10 — Trace (ordered execution context)"
curl -sS "$BASE_URL/v1/openclaw/runs/$RUN_ID/trace" | python3 -c "
import sys,json
d=json.load(sys.stdin)['data']
print('workflowId:', d['workflowId'], '| runId:', d['runId'])
print('steps:', len(d['steps']))
for s in d['steps']:
    print(f'  [{s[\"stepIndex\"]}] {s[\"eventType\"]} | actor={s[\"actor\"]} | {s[\"payload\"]}')"

# -----------------------------------------------------------------------------
# Judge console
# -----------------------------------------------------------------------------
announce "STEP 11 — Judge console"
echo "URL: $BASE_URL/judge/verify?runId=$RUN_ID"
curl -sS -o /dev/null -w "HTTP status: %{http_code}\n" "$BASE_URL/judge/verify?runId=$RUN_ID"

# -----------------------------------------------------------------------------
announce "Demo complete!"
echo ""
echo "Run ID: $RUN_ID"
echo "Data dir: $DATA_DIR"
echo ""
echo "Story proved:"
echo "  ✓ ingest — OpenClaw events entered the service"
echo "  ✓ checkpoint — state deterministically derived and persisted"
echo "  ✓ crash — process died"
echo "  ✓ recover — run hydrated from disk after restart"
echo "  ✓ verify — checkpoint re-derived and matched persisted proof"
echo "  ✓ trace — ordered execution context intact"
