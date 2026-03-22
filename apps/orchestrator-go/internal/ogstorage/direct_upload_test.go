package ogstorage

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/0gfoundation/0g-storage-client/core"
)

func TestBuildDirectUploadPlanSingleSegmentRootMatchesLegacyComputation(t *testing.T) {
	t.Parallel()

	payload := []byte("demo-checkpoint-v1")

	plan, err := buildDirectUploadPlan(payload)
	if err != nil {
		t.Fatalf("buildDirectUploadPlan() error = %v", err)
	}

	if got, want := plan.tree.Root().Hex(), computeSingleSegmentRoot(payload); got != want {
		t.Fatalf("plan.tree.Root() = %s, want %s", got, want)
	}

	segments, err := plan.segments()
	if err != nil {
		t.Fatalf("plan.segments() error = %v", err)
	}
	if len(segments) != 1 {
		t.Fatalf("len(plan.segments()) = %d, want 1", len(segments))
	}
	if got, want := len(segments[0].Data), core.DefaultChunkSize; got != want {
		t.Fatalf("len(segments[0].Data) = %d, want %d", got, want)
	}
}

func TestBuildDirectUploadPlanMultiSegmentProofsValidate(t *testing.T) {
	t.Parallel()

	payload := make([]byte, core.DefaultSegmentSize+321)
	rng := rand.New(rand.NewSource(42))
	for i := range payload {
		payload[i] = byte(rng.Intn(256))
	}

	plan, err := buildDirectUploadPlan(payload)
	if err != nil {
		t.Fatalf("buildDirectUploadPlan() error = %v", err)
	}
	if plan.data.NumSegments() < 2 {
		t.Fatalf("plan.data.NumSegments() = %d, want >= 2", plan.data.NumSegments())
	}
	if !bytes.Equal(plan.storedPayload, payload) {
		t.Fatal("plan.storedPayload changed incompressible payload, want raw payload")
	}

	segments, err := plan.segments()
	if err != nil {
		t.Fatalf("plan.segments() error = %v", err)
	}
	if got, want := len(segments), int(plan.data.NumSegments()); got != want {
		t.Fatalf("len(plan.segments()) = %d, want %d", got, want)
	}

	for _, segment := range segments {
		segmentRoot, numSegmentsFlowPadded := core.PaddedSegmentRoot(segment.Index, segment.Data, plan.data.Size())
		if err := segment.Proof.ValidateHash(plan.tree.Root(), segmentRoot, segment.Index, numSegmentsFlowPadded); err != nil {
			t.Fatalf("segment %d proof validation error = %v", segment.Index, err)
		}
	}
}
