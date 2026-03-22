package ogstorage

import (
	"fmt"

	"github.com/0gfoundation/0g-storage-client/contract"
	"github.com/0gfoundation/0g-storage-client/core"
	"github.com/0gfoundation/0g-storage-client/core/merkle"
	"github.com/0gfoundation/0g-storage-client/node"
	"github.com/ethereum/go-ethereum/common"
)

type directUploadPlan struct {
	storedPayload []byte
	data          core.IterableData
	tree          *merkle.Tree
}

func buildDirectUploadPlan(payload []byte) (*directUploadPlan, error) {
	storedPayload := selectDirectStoredPayload(payload)

	data, err := core.NewDataInMemory(storedPayload)
	if err != nil {
		return nil, fmt.Errorf("build direct upload data: %w", err)
	}

	tree, err := core.MerkleTree(data)
	if err != nil {
		return nil, fmt.Errorf("build direct upload merkle tree: %w", err)
	}

	return &directUploadPlan{
		storedPayload: storedPayload,
		data:          data,
		tree:          tree,
	}, nil
}

func (p *directUploadPlan) root() common.Hash {
	return p.tree.Root()
}

func (p *directUploadPlan) submission(submitter common.Address) (*contract.Submission, error) {
	submission, err := core.NewFlow(p.data, nil).CreateSubmission(submitter)
	if err != nil {
		return nil, fmt.Errorf("build direct upload submission: %w", err)
	}
	return submission, nil
}

func (p *directUploadPlan) segments() ([]node.SegmentWithProof, error) {
	numChunks := p.data.NumChunks()
	segments := make([]node.SegmentWithProof, 0, p.data.NumSegments())

	for segIndex := uint64(0); segIndex < p.data.NumSegments(); segIndex++ {
		startIndex := segIndex * core.DefaultSegmentMaxChunks
		if startIndex >= numChunks {
			break
		}

		segment, err := core.ReadAt(p.data, core.DefaultSegmentSize, int64(segIndex*core.DefaultSegmentSize), p.data.PaddedSize())
		if err != nil {
			return nil, fmt.Errorf("read direct upload segment %d: %w", segIndex, err)
		}

		if startIndex+uint64(len(segment))/core.DefaultChunkSize >= numChunks {
			expectedLen := core.DefaultChunkSize * int(numChunks-startIndex)
			segment = segment[:expectedLen]
		}

		segments = append(segments, node.SegmentWithProof{
			Root:     p.tree.Root(),
			Data:     segment,
			Index:    segIndex,
			Proof:    p.tree.ProofAt(int(segIndex)),
			FileSize: uint64(p.data.Size()),
		})
	}

	return segments, nil
}
