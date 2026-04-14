#!/usr/bin/env bash
# End-to-end demo: start orchestrator, run agent demo, verify results.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
ORCH_DIR="${REPO_ROOT}/apps/orchestrator-go"
AGENT_DIR="${REPO_ROOT}/apps/agent-demo"
ORCH_URL="http://127.0.0.1:8080"

# ── Colors ──
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

info()  { printf "${GREEN}[INFO]${NC} %s\n" "$*"; }
warn()  { printf "${YELLOW}[WARN]${NC} %s\n" "$*"; }
fail()  { printf "${RED}[FAIL]${NC} %s\n" "$*"; exit 1; }

cleanup() {
  if [[ -n "${ORCH_PID:-}" ]]; then
    info "Stopping orchestrator (PID=${ORCH_PID})..."
    kill "${ORCH_PID}" 2>/dev/null || true
    wait "${ORCH_PID}" 2>/dev/null || true
  fi
}
trap cleanup EXIT

# ── 1. Build orchestrator ──
info "Building orchestrator..."
cd "${ORCH_DIR}"
go build -o /tmp/orch-demo . || fail "Failed to build orchestrator"

# ── 2. Start orchestrator ──
info "Starting orchestrator on ${ORCH_URL}..."
ORCH_DATA_DIR="$(mktemp -d)"
ORCH_DATA_DIR="${ORCH_DATA_DIR}" ORCH_HTTP_ADDR="127.0.0.1:8080" /tmp/orch-demo serve &
ORCH_PID=$!

# Wait for health
for i in $(seq 1 20); do
  if curl -sf "${ORCH_URL}/health" >/dev/null 2>&1; then
    info "Orchestrator healthy"
    break
  fi
  if [[ $i -eq 20 ]]; then
    fail "Orchestrator failed to start"
  fi
  sleep 0.5
done

# ── 3. Run agent demo ──
info "Running agent demo..."
cd "${AGENT_DIR}"
ORCH_URL="${ORCH_URL}" go run . || fail "Agent demo failed"

# ── 4. Verify ──
echo
info "Fetching run list..."
RUNS=$(curl -sf "${ORCH_URL}/v1/openclaw/runs")
echo "${RUNS}" | python3 -m json.tool 2>/dev/null || echo "${RUNS}"

# Get the latest run ID from the output
RUN_ID=$(echo "${RUNS}" | python3 -c "
import sys, json
data = json.load(sys.stdin)
runs = data.get('data', {}).get('runs', [])
if runs:
    print(runs[-1].get('runId', ''))
" 2>/dev/null || echo "")

if [[ -n "${RUN_ID}" ]]; then
  echo
  info "Verifying run: ${RUN_ID}"
  VERIFY=$(curl -sf "${ORCH_URL}/v1/openclaw/runs/${RUN_ID}/verify")
  echo "${VERIFY}" | python3 -m json.tool 2>/dev/null || echo "${VERIFY}"

  echo
  info "Trace:"
  TRACE=$(curl -sf "${ORCH_URL}/v1/openclaw/runs/${RUN_ID}/trace")
  echo "${TRACE}" | python3 -m json.tool 2>/dev/null || echo "${TRACE}"
fi

echo
info "Dashboard: ${ORCH_URL}/dashboard"
info "Judge console: ${ORCH_URL}/judge/verify?runId=${RUN_ID}"
info "Agent demo e2e complete!"
