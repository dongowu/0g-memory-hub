package ogstorage

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestComputeSingleSegmentRootMatchesLiveEvidence(t *testing.T) {
	t.Parallel()

	got := computeSingleSegmentRoot([]byte("demo-checkpoint-v1"))
	want := "0xdcd9fbfdca6fc87e5d371b00623a90a0026d25103b13eb236bafe6aa6be844e8"
	if got != want {
		t.Fatalf("computeSingleSegmentRoot() = %s, want %s", got, want)
	}
}

func TestGzipForSingleSegmentCompressesCheckpointShape(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"workflow_id":"live-wf-20260322","agent_id":"openclaw-agent","latest_step":1,"root_hash":"0xroot","status":"Running","events":[{"event_id":"evt-demo","step_index":1,"event_type":"tool_result","actor":"openclaw","payload":"{\"task\":\"live_storage_probe\",\"ok\":true,\"ts\":\"2026-03-22\"}"}]}`)
	if len(raw) <= directFallbackMaxBytes {
		t.Fatalf("raw length = %d, want > %d", len(raw), directFallbackMaxBytes)
	}

	compressed, ok := gzipForSingleSegment(raw)
	if !ok {
		t.Fatalf("gzipForSingleSegment() ok = false, raw=%d", len(raw))
	}
	if len(compressed) > directFallbackMaxBytes {
		t.Fatalf("compressed length = %d, want <= %d", len(compressed), directFallbackMaxBytes)
	}

	roundTrip, err := maybeGunzip(compressed)
	if err != nil {
		t.Fatalf("maybeGunzip() error = %v", err)
	}
	if !bytes.Equal(roundTrip, raw) {
		t.Fatalf("roundTrip mismatch: got %q want %q", string(roundTrip), string(raw))
	}
}

func TestMaybeGunzipPassesThroughPlainPayload(t *testing.T) {
	t.Parallel()

	raw := []byte("plain-checkpoint")
	got, err := maybeGunzip(raw)
	if err != nil {
		t.Fatalf("maybeGunzip() error = %v", err)
	}
	if !bytes.Equal(got, raw) {
		t.Fatalf("got %q, want %q", string(got), string(raw))
	}
}

func TestEncodeDirectFallbackPayloadCompactsCheckpointJSON(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"workflow_id":"live-wf-20260322","agent_id":"openclaw-agent","latest_step":0,"root_hash":"0x82b5f1d84cbcffcc3f8fbc8d7d3f74a2050c977f70fca8f723274f793be4e5da","status":"Running","events":[{"event_id":"live-wf-20260322-step-0","step_index":0,"event_type":"tool_result","actor":"openclaw","payload":"{\"task\":\"live_storage_direct_fallback\",\"ok\":true,\"ts\":\"2026-03-22\"}"}]}`)

	encoded, ok := encodeDirectFallbackPayload(raw)
	if !ok {
		t.Fatal("encodeDirectFallbackPayload() ok = false, want true")
	}
	if len(encoded) > directFallbackMaxBytes {
		t.Fatalf("encoded length = %d, want <= %d", len(encoded), directFallbackMaxBytes)
	}

	decoded, err := decodeStoredPayload(encoded)
	if err != nil {
		t.Fatalf("decodeStoredPayload() error = %v", err)
	}

	var got any
	if err := json.Unmarshal(decoded, &got); err != nil {
		t.Fatalf("decodeStoredPayload() produced invalid json: %v", err)
	}
	var want any
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatalf("raw json invalid: %v", err)
	}
	gotJSON, _ := json.Marshal(got)
	wantJSON, _ := json.Marshal(want)
	if !bytes.Equal(gotJSON, wantJSON) {
		t.Fatalf("decoded json mismatch: got=%s want=%s", gotJSON, wantJSON)
	}
}
