package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// Client 0G Storage 客户端
type Client struct {
	rpcURL     string
	indexerURL string
	logger     *logrus.Logger
	httpClient *http.Client
}

// Config 存储配置
type Config struct {
	RPCURL     string
	IndexerURL string
}

// NewClient 创建新的存储客户端
func NewClient(config *Config) *Client {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	return &Client{
		rpcURL:     config.RPCURL,
		indexerURL: config.IndexerURL,
		logger:     logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// UploadResult 上传结果
type UploadResult struct {
	CID    string `json:"cid"`
	TxHash string `json:"tx_hash"`
	Size   int    `json:"size"`
}

// Upload 上传数据到 0G Storage
func (c *Client) Upload(ctx context.Context, data []byte) (*UploadResult, error) {
	c.logger.WithField("size", len(data)).Info("Uploading to 0G Storage")

	// 计算内容哈希
	hash := sha256.Sum256(data)
	contentHash := hex.EncodeToString(hash[:])

	// 生成 CID (格式: 0g_{hash[:16]})
	cid := fmt.Sprintf("0g_%s", contentHash[:16])

	// 模拟上传（实际需要调用 0G SDK 或 API）
	// 在生产环境中，这里应该调用 0G Storage SDK
	txHash := fmt.Sprintf("0x%s", contentHash[:32])

	c.logger.WithFields(logrus.Fields{
		"cid":    cid,
		"txHash": txHash,
	}).Info("Upload successful")

	return &UploadResult{
		CID:    cid,
		TxHash: txHash,
		Size:   len(data),
	}, nil
}

// UploadFile 上传文件
func (c *Client) UploadFile(ctx context.Context, filePath string) (*UploadResult, error) {
	c.logger.WithField("file", filePath).Info("Uploading file to 0G Storage")

	// 读取文件
	data, err := readFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return c.Upload(ctx, data)
}

// Download 从 0G Storage 下载数据
func (c *Client) Download(ctx context.Context, cid string) ([]byte, error) {
	c.logger.WithField("cid", cid).Info("Downloading from 0G Storage")

	// 模拟下载（实际需要调用 0G SDK 或 API）
	// 在生产环境中，这里应该调用 0G Storage SDK
	mockData := []byte(fmt.Sprintf("Mock content for CID: %s", cid))

	c.logger.Info("Download successful")
	return mockData, nil
}

// DownloadFile 下载文件
func (c *Client) DownloadFile(ctx context.Context, cid, outputPath string) error {
	c.logger.WithFields(logrus.Fields{
		"cid":   cid,
		"to":    outputPath,
	}).Info("Downloading file from 0G Storage")

	data, err := c.Download(ctx, cid)
	if err != nil {
		return err
	}

	return writeFile(outputPath, data)
}

// VerifyCID 验证 CID 是否存在
func (c *Client) VerifyCID(ctx context.Context, cid string) (bool, error) {
	c.logger.WithField("cid", cid).Info("Verifying CID")

	// 尝试下载
	_, err := c.Download(ctx, cid)
	if err != nil {
		return false, nil
	}

	return true, nil
}

// GetFileInfo 获取文件信息
type FileInfo struct {
	CID       string `json:"cid"`
	Size      int    `json:"size"`
	Timestamp int64  `json:"timestamp"`
}

// GetFileInfo 获取文件信息
func (c *Client) GetFileInfo(ctx context.Context, cid string) (*FileInfo, error) {
	// 模拟获取文件信息
	return &FileInfo{
		CID:       cid,
		Size:      1024, // mock
		Timestamp: time.Now().Unix(),
	}, nil
}

// IndexerAPI Indexer API 客户端

// FileInfoResponse Indexer API 响应
type FileInfoResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    *FileData   `json:"data"`
}

type FileData struct {
	CID       string `json:"cid"`
	Size      int    `json:"size"`
	Timestamp int64  `json:"timestamp"`
	TxHash    string `json:"tx_hash"`
}

// GetFileInfoFromIndexer 从 Indexer 获取文件信息
func (c *Client) GetFileInfoFromIndexer(ctx context.Context, cid string) (*FileInfo, error) {
	url := fmt.Sprintf("%s/file/info/%s", c.indexerURL, cid)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result FileInfoResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Code != 200 {
		return nil, fmt.Errorf("api error: %s", result.Message)
	}

	return &FileInfo{
		CID:       result.Data.CID,
		Size:      result.Data.Size,
		Timestamp: result.Data.Timestamp,
	}, nil
}

// Helper functions

// Helper functions

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

// BatchUploadResult 批量上传结果
type BatchUploadResult struct {
	Success int
	Failed  int
	Results []*UploadResult
}

// UploadBatch 批量上传
func (c *Client) UploadBatch(ctx context.Context, dataList [][]byte) (*BatchUploadResult, error) {
	result := &BatchUploadResult{
		Results: make([]*UploadResult, 0, len(dataList)),
	}

	for _, data := range dataList {
		uploadResult, err := c.Upload(ctx, data)
		if err != nil {
			result.Failed++
			continue
		}
		result.Success++
		result.Results = append(result.Results, uploadResult)
	}

	return result, nil
}

// StreamUpload 流式上传
type StreamUploader struct {
	client *Client
	ctx    context.Context
	data   *bytes.Buffer
}

// NewStreamUploader 创建流式上传器
func (c *Client) NewStreamUploader(ctx context.Context) *StreamUploader {
	return &StreamUploader{
		client: c,
		ctx:    ctx,
		data:   bytes.NewBuffer(nil),
	}
}

// Write 写入数据
func (s *StreamUploader) Write(p []byte) (n int, err error) {
	return s.data.Write(p)
}

// Close 完成上传
func (s *StreamUploader) Close() (*UploadResult, error) {
	return s.client.Upload(s.ctx, s.data.Bytes())
}
