package ogstorage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/0gfoundation/0g-storage-client/transfer"
)

type fakeUploadSession struct {
	gotPath string
	root    string
	txHash  string
	err     error
}

func (s *fakeUploadSession) UploadFile(_ context.Context, filePath string) (string, string, error) {
	s.gotPath = filePath
	payload, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", err
	}
	if string(payload) != "checkpoint-blob" {
		return "", "", unexpectedPayloadError(payload)
	}
	return s.root, s.txHash, s.err
}

type fakeDownloadSession struct {
	gotRoot string
	gotPath string
	data    []byte
	err     error
}

func (s *fakeDownloadSession) Download(_ context.Context, root, outputPath string, _ bool) error {
	s.gotRoot = root
	s.gotPath = outputPath
	if s.err != nil {
		return s.err
	}
	return os.WriteFile(outputPath, s.data, 0o600)
}

type fakeSessionFactory struct {
	root        string
	txHash      string
	download    []byte
	uploadErr   error
	downloadErr error
}

func (f fakeSessionFactory) UploadBytes(_ context.Context, _ SDKConfig, payload []byte) (string, string, error) {
	if f.uploadErr != nil {
		return "", "", f.uploadErr
	}
	if string(payload) != "checkpoint-blob" {
		return "", "", unexpectedPayloadError(payload)
	}
	return f.root, f.txHash, nil
}

func (f fakeSessionFactory) DownloadBytes(_ context.Context, _ SDKConfig, key string) ([]byte, error) {
	if f.downloadErr != nil {
		return nil, f.downloadErr
	}
	if key != "0xroot" {
		return nil, unexpectedKeyError(key)
	}
	return append([]byte(nil), f.download...), nil
}

type unexpectedPayloadError []byte

func (e unexpectedPayloadError) Error() string { return "unexpected payload: " + string(e) }

type unexpectedKeyError string

func (e unexpectedKeyError) Error() string { return "unexpected key: " + string(e) }

type stubReadinessHTTPClient struct {
	do func(req *http.Request) (*http.Response, error)
}

func (c stubReadinessHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.do(req)
}

func makeHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestUploadCheckpoint(t *testing.T) {
	t.Parallel()

	client := &SDKClient{
		config: SDKConfig{
			IndexerRPCURL:    "https://indexer-storage-testnet-standard.0g.ai",
			BlockchainRPCURL: "https://evmrpc-testnet.0g.ai",
			PrivateKey:       "0xabc123",
		},
		adapter: fakeSessionFactory{
			root:   "0xroot",
			txHash: "0xtx",
		},
	}

	result, err := client.UploadCheckpoint(context.Background(), []byte("checkpoint-blob"))
	if err != nil {
		t.Fatalf("UploadCheckpoint() error = %v", err)
	}
	if result.Key != "0xroot" {
		t.Fatalf("result.Key = %s, want 0xroot", result.Key)
	}
	if result.TxHash != "0xtx" {
		t.Fatalf("result.TxHash = %s, want 0xtx", result.TxHash)
	}
}

func TestUploadCheckpointRequiresChainCredentials(t *testing.T) {
	t.Parallel()

	client := &SDKClient{
		config: SDKConfig{
			IndexerRPCURL: "https://indexer-storage-testnet-standard.0g.ai",
		},
	}

	_, err := client.UploadCheckpoint(context.Background(), []byte("checkpoint-blob"))
	if err == nil {
		t.Fatal("UploadCheckpoint() error = nil, want credential/config error")
	}
}

func TestDownloadCheckpoint(t *testing.T) {
	t.Parallel()

	client := &SDKClient{
		config: SDKConfig{
			IndexerRPCURL: "https://indexer-storage-testnet-standard.0g.ai",
		},
		adapter: fakeSessionFactory{
			download: []byte("checkpoint-blob"),
		},
	}

	payload, err := client.DownloadCheckpoint(context.Background(), "0xroot")
	if err != nil {
		t.Fatalf("DownloadCheckpoint() error = %v", err)
	}
	if string(payload) != "checkpoint-blob" {
		t.Fatalf("payload = %q, want checkpoint-blob", string(payload))
	}
}

func TestDefaultUploadOptionUsesOfficialWorkingDefaults(t *testing.T) {
	t.Parallel()

	opt := defaultUploadOption(SDKConfig{ExpectedReplica: 1})

	if opt.FinalityRequired != transfer.FileFinalized {
		t.Fatalf("FinalityRequired = %v, want %v", opt.FinalityRequired, transfer.FileFinalized)
	}
	if opt.Method != "min" {
		t.Fatalf("Method = %q, want min", opt.Method)
	}
	if !opt.FullTrusted {
		t.Fatal("FullTrusted = false, want true")
	}
	if opt.ExpectedReplica != 1 {
		t.Fatalf("ExpectedReplica = %d, want 1", opt.ExpectedReplica)
	}
}

func TestCheckReadinessLiveProbePrimaryIndexer(t *testing.T) {
	t.Parallel()

	var requestedPath string
	client := &SDKClient{
		config: SDKConfig{
			IndexerRPCURL:    "https://indexer-storage-testnet-standard.0g.ai",
			BlockchainRPCURL: "https://evmrpc-testnet.0g.ai",
			PrivateKey:       "0xabc123",
		},
		prober: stubReadinessHTTPClient{
			do: func(req *http.Request) (*http.Response, error) {
				requestedPath = req.URL.Path
				return makeHTTPResponse(http.StatusOK, `{"code":0,"message":"ok","data":{"networkIdentity":{"flowAddress":"0x00000000000000000000000000000000000000aa"}}}`), nil
			},
		},
	}

	if err := client.CheckReadiness(context.Background()); err != nil {
		t.Fatalf("CheckReadiness() error = %v", err)
	}
	if requestedPath != "/node/status" {
		t.Fatalf("requested path = %q, want /node/status", requestedPath)
	}
}

func TestCheckReadinessFallsBackToTurboIndexer(t *testing.T) {
	t.Parallel()

	var hosts []string
	client := &SDKClient{
		config: SDKConfig{
			IndexerRPCURL:    "https://broken-indexer.local",
			BlockchainRPCURL: "https://evmrpc-testnet.0g.ai",
			PrivateKey:       "0xabc123",
		},
		prober: stubReadinessHTTPClient{
			do: func(req *http.Request) (*http.Response, error) {
				hosts = append(hosts, req.URL.Host)
				if req.URL.Host == "broken-indexer.local" {
					return nil, errors.New("dial failed")
				}
				if req.URL.Host == "indexer-storage-testnet-turbo.0g.ai" {
					return makeHTTPResponse(http.StatusOK, `{"code":0,"message":"ok","data":{"networkIdentity":{"flowAddress":"0x00000000000000000000000000000000000000bb"}}}`), nil
				}
				return nil, errors.New("unexpected host")
			},
		},
	}

	if err := client.CheckReadiness(context.Background()); err != nil {
		t.Fatalf("CheckReadiness() error = %v", err)
	}
	if len(hosts) != 2 {
		t.Fatalf("probe attempts = %d, want 2", len(hosts))
	}
}

func TestCheckReadinessLiveProbeFailure(t *testing.T) {
	t.Parallel()

	client := &SDKClient{
		config: SDKConfig{
			IndexerRPCURL:    "https://indexer-storage-testnet-standard.0g.ai",
			BlockchainRPCURL: "https://evmrpc-testnet.0g.ai",
			PrivateKey:       "0xabc123",
		},
		prober: stubReadinessHTTPClient{
			do: func(req *http.Request) (*http.Response, error) {
				return makeHTTPResponse(http.StatusOK, `{"code":0,"message":"ok","data":{"networkIdentity":{"flowAddress":""}}}`), nil
			},
		},
	}

	err := client.CheckReadiness(context.Background())
	if err == nil {
		t.Fatal("CheckReadiness() error = nil, want live probe error")
	}
	if !strings.Contains(err.Error(), "empty flowAddress") {
		t.Fatalf("error = %v, want empty flowAddress", err)
	}
}
