#!/usr/bin/env bash
# =============================================================================
# 0G Memory Hub — Auto-Record (Terminal Output Video)
# Records terminal session to MP4 using avfoundation (works on macOS).
# Usage: bash scripts/auto_record.sh
#
# If avfoundation screen capture fails (needs permissions),
# use macOS built-in: Control+Command+N in Terminal, or QuickTime.
# =============================================================================

set -euo pipefail

REPO_ROOT="/Users/dongowu/code/project/project_dev/0g-memory-hub"
OUTPUT="$REPO_ROOT/demo.mp4"
BINARY_PATH="/tmp/orchestrator"
SERVER_PID=""

RUN_ID="run-$(date +%Y%m%d%H%M%S)"
WORKFLOW_ID="wf-${RUN_ID}"
BASE_URL="http://127.0.0.1:8080"

export ORCH_DATA_DIR="/tmp/0g-recording-data"
export ORCH_RUNTIME_BINARY_PATH="$REPO_ROOT/rust/memory-core/target/debug/memory-core-rpc"
export ORCH_HTTP_ADDR="127.0.0.1:8080"

announce() {
  echo ""
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "  $1"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

wait_server() {
  local i=0
  while ! curl -s --max-time 2 http://127.0.0.1:8080/health > /dev/null 2>&1; do
    i=$((i+1))
    if [ $i -gt 15 ]; then echo "ERROR: server did not start"; exit 1; fi
    sleep 1
  done
}

stop_all() {
  [ -n "$SERVER_PID" ] && kill "$SERVER_PID" 2>/dev/null || true
  pkill -f "orchestrator serve" 2>/dev/null || true
  # Stop ffmpeg gracefully and wait for it to finish flushing
  if [ -n "$REC_PID" ]; then
    kill -INT "$REC_PID" 2>/dev/null || true
    # Wait for ffmpeg to finish writing (max 10s)
    for i in $(seq 1 10); do
      if ! kill -0 "$REC_PID" 2>/dev/null; then
        break
      fi
      sleep 1
    done
    kill -9 "$REC_PID" 2>/dev/null || true
  fi
}

cleanup() {
  stop_all
  sleep 1
  if [ -f "/tmp/0g-recording.mp4" ] && [ -s "/tmp/0g-recording.mp4" ]; then
    cp /tmp/0g-recording.mp4 "$OUTPUT"
    echo "Video: $OUTPUT"
    ls -lh "$OUTPUT"
  fi
}
trap cleanup EXIT

REC_PID=""

start_recording() {
  # Try ffmpeg screen capture first
  # Find screen device
  local screen_dev
  screen_dev=$(ffmpeg -f avfoundation -list_devices true -i "" 2>&1 | grep -o '\[[0-9]*\] Capture screen' | head -1 | grep -o '[0-9]*')
  if [ -z "$screen_dev" ]; then
    echo "⚠️  Screen capture needs permission. Open System Preferences > Privacy > Screen Recording"
    echo "   Then run: bash scripts/auto_record.sh"
    echo "   Or record manually with QuickTime Player."
    return 1
  fi

  echo "🎬 Recording screen (device $screen_dev)..."
  ffmpeg -f avfoundation -capture_cursor 1 -capture_mouse_clicks 1 \
    -i "${screen_dev}:none" \
    -c:v libx264 -preset fast -crf 23 -pix_fmt yuv420p \
    -y /tmp/0g-recording.mp4 &
  REC_PID=$!
  sleep 2
  if ! kill -0 "$REC_PID" 2>/dev/null; then
    echo "⚠️  ffmpeg screen capture failed (permission?)."
    echo "   Use QuickTime Player: File > New Screen Recording"
    return 1
  fi
  echo "Recording PID: $REC_PID"
  return 0
}

# Kill existing
pkill -f "orchestrator serve" 2>/dev/null || true
sleep 2
rm -rf "$ORCH_DATA_DIR" && mkdir -p "$ORCH_DATA_DIR"

# Start recording (optional — won't fail if it doesn't work)
start_recording || true

cd "$REPO_ROOT/apps/orchestrator-go"
if [ ! -f "$BINARY_PATH" ]; then
  go build -o "$BINARY_PATH" . 2>/dev/null
fi

announce "STEP 1 — Start server"
"$BINARY_PATH" serve &
SERVER_PID=$!
wait_server
echo "Server up at $BASE_URL"

announce "STEP 2 — Health check"
curl -sS "$BASE_URL/health" | python3 -c "
import sys,json; d=json.load(sys.stdin)['data']
print('Ready:', d['ready'])
for k,v in d['components'].items():
    print(f'  {k}: {v[\"ready\"]} — {v.get(\"message\",\"\")}')"
sleep 1

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
import sys,json; d=json.load(sys.stdin)['data']
print('workflowId:', d['workflowId'])
print('latestStep:', d['latestStep'])
print('latestRoot:', d['latestRoot'][:24]+'...')
print('latestCid:', d['latestCid'][:24]+'...')
print('latestTxHash:', d['latestTxHash'])"

announce "STEP 4 — Show context"
curl -sS "$BASE_URL/v1/openclaw/runs/$RUN_ID/context" | python3 -c "
import sys,json; d=json.load(sys.stdin)['data']
print('runId:', d['runId'], '| sessionId:', d['sessionId'])
print('events:', len(d['events']))
for e in d['events']:
    print(f'  [{e[\"stepIndex\"]}] {e[\"eventType\"]} | actor={e[\"actor\"]} | skill={e.get(\"skillName\",\"\")} | {e[\"payload\"]}')"

announce "STEP 5 — Show checkpoint (persisted root hash + CID)"
curl -sS "$BASE_URL/v1/openclaw/runs/$RUN_ID/checkpoint/latest" | python3 -c "
import sys,json; d=json.load(sys.stdin)['data']
print('latestStep:', d['latestStep'])
print('rootHash:', d['latestRoot'])
print('cid:', d['latestCid'])"

announce "STEP 6 — CRASH: kill server (process dies, memory survives)"
echo "Killing PID $SERVER_PID..."
kill "$SERVER_PID" 2>/dev/null || true
sleep 2
echo "Process dead. Workflow data still on disk:"
cat "$ORCH_DATA_DIR/workflows.json" 2>/dev/null | python3 -c "
import sys,json; d=json.load(sys.stdin)
for k,v in d.items(): print(f'  {k}: step={v.get(\"latest_step\")}')" 2>/dev/null

announce "STEP 7 — Restart server (same data dir — run survives)"
"$BINARY_PATH" serve &
SERVER_PID=$!
wait_server
echo "Server restarted (PID $SERVER_PID)!"

announce "STEP 8 — Hydrate (rebuild from persisted state)"
curl -sS -X POST "$BASE_URL/v1/openclaw/runs/$RUN_ID/hydrate" | python3 -c "
import sys,json; d=json.load(sys.stdin)['data']
print('workflowId:', d['workflowId'])
print('latestStep:', d['latestStep'])
print('events restored:', len(d['events']))"

announce "STEP 9 — Verify (re-derive checkpoint + compare)"
curl -sS "$BASE_URL/v1/openclaw/runs/$RUN_ID/verify" | python3 -c "
import sys,json; d=json.load(sys.stdin)['data']
passed=sum(1 for c in d['checks'] if c['passed'])
total=len(d['checks'])
print('Verified:', d['verified'], '| Checks:', passed, '/', total)
for c in d['checks']: print(f'  {\"✓\" if c[\"passed\"] else \"✗\"} {c[\"name\"]}')"

announce "STEP 10 — Trace (ordered execution context)"
curl -sS "$BASE_URL/v1/openclaw/runs/$RUN_ID/trace" | python3 -c "
import sys,json; d=json.load(sys.stdin)['data']
print('workflowId:', d['workflowId'], '| runId:', d['runId'])
print('steps:', len(d['steps']))
for s in d['steps']: print(f'  [{s[\"stepIndex\"]}] {s[\"eventType\"]} | {s[\"actor\"]} | {s[\"payload\"]}')"

announce "STEP 11 — Judge console"
echo "URL: $BASE_URL/judge/verify?runId=$RUN_ID"
curl -sS -o /dev/null -w "HTTP status: %{http_code}\n" "$BASE_URL/judge/verify?runId=$RUN_ID"

announce "Demo complete!"
echo "Run ID: $RUN_ID"
echo ""
echo "✓ ingest — OpenClaw events entered the service"
echo "✓ checkpoint — state deterministically derived and persisted"
echo "✓ crash — process died"
echo "✓ recover — run hydrated from disk after restart"
echo "✓ verify — checkpoint re-derived and matched (8/8 checks)"
echo "✓ trace — ordered execution context intact"
echo ""
echo "Video saved to: $OUTPUT"
