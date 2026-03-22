package ogstorage

import (
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"io"

	"golang.org/x/crypto/sha3"
)

const directFallbackMaxBytes = 256

type checkpointJSON struct {
	WorkflowID string                `json:"workflow_id"`
	AgentID    string                `json:"agent_id"`
	LatestStep uint64                `json:"latest_step"`
	RootHash   string                `json:"root_hash"`
	Status     string                `json:"status"`
	Events     []checkpointJSONEvent `json:"events"`
}

type checkpointJSONEvent struct {
	EventID   string `json:"event_id"`
	StepIndex uint64 `json:"step_index"`
	EventType string `json:"event_type"`
	Actor     string `json:"actor"`
	Payload   string `json:"payload"`
}

type compactCheckpointJSON struct {
	Version    int                        `json:"v"`
	WorkflowID string                     `json:"w"`
	AgentID    string                     `json:"a"`
	LatestStep uint64                     `json:"s"`
	RootHash   string                     `json:"r"`
	Status     string                     `json:"t"`
	Events     []compactCheckpointJSONEvt `json:"e"`
}

type compactCheckpointJSONEvt struct {
	EventID   string `json:"i"`
	StepIndex uint64 `json:"n"`
	EventType string `json:"y"`
	Actor     string `json:"c"`
	Payload   string `json:"p"`
}

func gzipForSingleSegment(payload []byte) ([]byte, bool) {
	out, err := gzipPayload(payload)
	if err != nil {
		return nil, false
	}
	return out, len(out) <= directFallbackMaxBytes
}

func gzipPayload(payload []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if _, err := zw.Write(payload); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func selectDirectStoredPayload(payload []byte) []byte {
	best := append([]byte(nil), payload...)

	updateBest := func(candidate []byte) {
		if len(candidate) < len(best) {
			best = append([]byte(nil), candidate...)
		}
	}

	if compact, ok := compactCheckpointPayload(payload); ok {
		updateBest(compact)
		if gzCompact, err := gzipPayload(compact); err == nil {
			updateBest(gzCompact)
		}
	}
	if gzPayload, err := gzipPayload(payload); err == nil {
		updateBest(gzPayload)
	}

	return best
}

func maybeGunzip(payload []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(payload))
	if err != nil {
		return payload, nil
	}
	defer zr.Close()

	decoded, err := io.ReadAll(zr)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

func encodeDirectFallbackPayload(payload []byte) ([]byte, bool) {
	encoded := selectDirectStoredPayload(payload)
	return encoded, len(encoded) <= directFallbackMaxBytes
}

func decodeStoredPayload(payload []byte) ([]byte, error) {
	decoded, err := maybeGunzip(payload)
	if err != nil {
		return nil, err
	}
	return maybeExpandCompactCheckpoint(decoded)
}

func computeSingleSegmentRoot(payload []byte) string {
	padded := make([]byte, directFallbackMaxBytes)
	copy(padded, payload)
	hash := sha3.NewLegacyKeccak256()
	hash.Write(padded)
	return "0x" + hex.EncodeToString(hash.Sum(nil))
}

func compactCheckpointPayload(payload []byte) ([]byte, bool) {
	var checkpoint checkpointJSON
	if err := json.Unmarshal(payload, &checkpoint); err != nil {
		return nil, false
	}

	compact := compactCheckpointJSON{
		Version:    1,
		WorkflowID: checkpoint.WorkflowID,
		AgentID:    checkpoint.AgentID,
		LatestStep: checkpoint.LatestStep,
		RootHash:   checkpoint.RootHash,
		Status:     checkpoint.Status,
		Events:     make([]compactCheckpointJSONEvt, 0, len(checkpoint.Events)),
	}
	for _, evt := range checkpoint.Events {
		compact.Events = append(compact.Events, compactCheckpointJSONEvt{
			EventID:   evt.EventID,
			StepIndex: evt.StepIndex,
			EventType: evt.EventType,
			Actor:     evt.Actor,
			Payload:   evt.Payload,
		})
	}

	out, err := json.Marshal(compact)
	if err != nil {
		return nil, false
	}
	return out, true
}

func maybeExpandCompactCheckpoint(payload []byte) ([]byte, error) {
	var compact compactCheckpointJSON
	if err := json.Unmarshal(payload, &compact); err != nil {
		return payload, nil
	}
	if compact.Version != 1 || compact.WorkflowID == "" {
		return payload, nil
	}

	checkpoint := checkpointJSON{
		WorkflowID: compact.WorkflowID,
		AgentID:    compact.AgentID,
		LatestStep: compact.LatestStep,
		RootHash:   compact.RootHash,
		Status:     compact.Status,
		Events:     make([]checkpointJSONEvent, 0, len(compact.Events)),
	}
	for _, evt := range compact.Events {
		checkpoint.Events = append(checkpoint.Events, checkpointJSONEvent{
			EventID:   evt.EventID,
			StepIndex: evt.StepIndex,
			EventType: evt.EventType,
			Actor:     evt.Actor,
			Payload:   evt.Payload,
		})
	}
	return json.Marshal(checkpoint)
}
