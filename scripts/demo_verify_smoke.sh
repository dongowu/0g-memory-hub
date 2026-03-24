#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
RUN_ID="${RUN_ID:-demo-verify-$(date +%Y%m%d%H%M%S)}"
WORKFLOW_ID="${WORKFLOW_ID:-wf-${RUN_ID}}"
SESSION_ID="${SESSION_ID:-session-${RUN_ID}}"
TRACE_ID="${TRACE_ID:-trace-${RUN_ID}}"
PAUSE_FOR_RESTART="${PAUSE_FOR_RESTART:-0}"

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "${tmp_dir}"
}
trap cleanup EXIT

compact_json() {
  sed -e ':a' -e 'N' -e '$!ba' -e 's/\n/ /g' -e 's/[[:space:]]\{1,\}/ /g' "$1"
}

request_json() {
  local label="$1"
  local method="$2"
  local path="$3"
  local body="${4-}"
  local out_file="${tmp_dir}/${label}.json"
  local status

  if [[ -n "${body}" ]]; then
    status="$(curl -sS -o "${out_file}" -w '%{http_code}' -X "${method}" \
      "${BASE_URL}${path}" \
      -H 'Accept: application/json' \
      -H 'Content-Type: application/json' \
      --data "${body}")"
  else
    status="$(curl -sS -o "${out_file}" -w '%{http_code}' -X "${method}" \
      "${BASE_URL}${path}" \
      -H 'Accept: application/json')"
  fi

  if [[ "${status}" != "200" ]]; then
    echo "[FAIL] ${label} -> HTTP ${status}" >&2
    cat "${out_file}" >&2
    echo >&2
    exit 1
  fi

  echo "[OK] ${label}"
  compact_json "${out_file}"
  echo
}

echo "== demo verify smoke =="
echo "BASE_URL=${BASE_URL}"
echo "WORKFLOW_ID=${WORKFLOW_ID}"
echo "RUN_ID=${RUN_ID}"
echo

request_json "health" "GET" "/health"

ingest_tool_call="$(cat <<JSON
{
  "workflowId": "${WORKFLOW_ID}",
  "runId": "${RUN_ID}",
  "sessionId": "${SESSION_ID}",
  "traceId": "${TRACE_ID}",
  "eventId": "${RUN_ID}-evt-1",
  "eventType": "tool_call",
  "actor": "planner",
  "role": "planner",
  "toolCallId": "${RUN_ID}-tool-1",
  "skillName": "search_skill",
  "taskId": "${RUN_ID}-task-1",
  "payload": {"tool":"search","query":"0G verify demo"}
}
JSON
)"

ingest_tool_result="$(cat <<JSON
{
  "workflowId": "${WORKFLOW_ID}",
  "runId": "${RUN_ID}",
  "sessionId": "${SESSION_ID}",
  "traceId": "${TRACE_ID}",
  "eventId": "${RUN_ID}-evt-2",
  "eventType": "tool_result",
  "actor": "worker",
  "role": "executor",
  "toolCallId": "${RUN_ID}-tool-1",
  "skillName": "search_skill",
  "taskId": "${RUN_ID}-task-1",
  "payload": {"ok":true,"items":1}
}
JSON
)"

request_json "ingest_1" "POST" "/v1/openclaw/ingest" "${ingest_tool_call}"
request_json "ingest_2" "POST" "/v1/openclaw/ingest" "${ingest_tool_result}"
request_json "context" "GET" "/v1/openclaw/runs/${RUN_ID}/context"
request_json "checkpoint_latest" "GET" "/v1/openclaw/runs/${RUN_ID}/checkpoint/latest"

if [[ "${PAUSE_FOR_RESTART}" == "1" ]]; then
  echo
  echo "[operator] Restart the service now in the other terminal, then press Enter to continue with hydrate/verify."
  read -r _
else
  echo
  echo "[operator] Optional restart point reached. If you want the full crash/recover story, restart the service now and re-run from hydrate."
fi
echo

request_json "hydrate" "POST" "/v1/openclaw/runs/${RUN_ID}/hydrate"
request_json "verify" "GET" "/v1/openclaw/runs/${RUN_ID}/verify"
request_json "trace" "GET" "/v1/openclaw/runs/${RUN_ID}/trace"

echo
echo "Judge console:"
echo "${BASE_URL}/judge/verify?runId=${RUN_ID}"
