package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const bucket = "lighthouse"

type server struct {
	s3      *minio.Client // internal — all S3 operations
	s3Local *minio.Client // local — presigned download URLs for the host
}

func newServer() (*server, error) {
	minioEndpoint := getEnv("MINIO_ENDPOINT", "127.0.0.1:9000")
	minioLocalEndpoint := getEnv("MINIO_LOCAL_ENDPOINT", "127.0.0.1:9000")
	minioUser := getEnv("MINIO_ROOT_USER", "lighthouse")
	minioPass := getEnv("MINIO_ROOT_PASSWORD", "lighthouse")

	s3, err := minio.New(minioEndpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(minioUser, minioPass, ""),
		Secure:       false,
		Region:       "us-east-1",
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	s3Local, err := minio.New(minioLocalEndpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(minioUser, minioPass, ""),
		Secure:       false,
		Region:       "us-east-1",
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 local client: %w", err)
	}

	return &server{s3: s3, s3Local: s3Local}, nil
}

// --- request/response types ---

type initUploadRequest struct {
	Filename    string `json:"filename"`
	TotalChunks int    `json:"total_chunks"`
}

type presignedURL struct {
	PartNumber int    `json:"part_number"`
	URL        string `json:"url"`
}

type finishUploadRequest struct {
	FileID   string `json:"file_id"`
	UploadID string `json:"upload_id"`
	Parts    []struct {
		PartNumber int    `json:"part_number"`
		ETag       string `json:"etag"`
	} `json:"parts"`
}

type abortUploadRequest struct {
	FileID   string `json:"file_id"`
	UploadID string `json:"upload_id"`
}

// --- handlers ---

func (s *server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *server) uploadInit(c *gin.Context) {
	var body initUploadRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	fileID := fmt.Sprintf("%s-%s", uuid.New().String(), body.Filename)

	core := minio.Core{Client: s.s3}
	uploadID, err := core.NewMultipartUpload(c.Request.Context(), bucket, fileID, minio.PutObjectOptions{
		UserMetadata: map[string]string{"filename": body.Filename},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to start multipart upload: %s", err)})
		return
	}

	// create a temporary client pointing to the public .onion endpoint for
	// presigning only — Presign() computes the signature locally, no connection is made
	publicEndpoint := getEnv("MINIO_PUBLIC_ENDPOINT", "127.0.0.1:9000")
	minioUser := getEnv("MINIO_ROOT_USER", "lighthouse")
	minioPass := getEnv("MINIO_ROOT_PASSWORD", "lighthouse")

	s3Public, err := minio.New(publicEndpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(minioUser, minioPass, ""),
		Secure:       false,
		Region:       "us-east-1",
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create public client: %s", err)})
		return
	}

	var urls []presignedURL
	for i := 1; i <= body.TotalChunks; i++ {
		url, err := s3Public.Presign(c.Request.Context(), http.MethodPut, bucket, fileID, 2*time.Hour, map[string][]string{
			"uploadId":   {uploadID},
			"partNumber": {fmt.Sprintf("%d", i)},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to generate presigned URL: %s", err)})
			return
		}
		urls = append(urls, presignedURL{PartNumber: i, URL: url.String()})
	}

	c.JSON(http.StatusOK, gin.H{
		"file_id":   fileID,
		"upload_id": uploadID,
		"urls":      urls,
	})
}

func (s *server) uploadFinish(c *gin.Context) {
	var body finishUploadRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	var parts []minio.CompletePart
	for _, p := range body.Parts {
		parts = append(parts, minio.CompletePart{
			PartNumber: p.PartNumber,
			ETag:       p.ETag,
		})
	}

	core := minio.Core{Client: s.s3}
	if _, err := core.CompleteMultipartUpload(c.Request.Context(), bucket, body.FileID, body.UploadID, parts, minio.PutObjectOptions{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to complete multipart upload: %s", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"file_id": body.FileID, "status": "ok"})
}

func (s *server) uploadAbort(c *gin.Context) {
	var body abortUploadRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	core := minio.Core{Client: s.s3}
	if err := core.AbortMultipartUpload(c.Request.Context(), bucket, body.FileID, body.UploadID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to abort multipart upload: %s", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "aborted"})
}

func (s *server) listFiles(c *gin.Context) {
	type fileInfo struct {
		FileID     string `json:"file_id"`
		Filename   string `json:"filename"`
		Size       int64  `json:"size"`
		UploadedAt string `json:"uploaded_at"`
	}

	var files []fileInfo

	for obj := range s.s3.ListObjects(c.Request.Context(), bucket, minio.ListObjectsOptions{}) {
		if obj.Err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to list objects: %s", obj.Err)})
			return
		}

		stat, err := s.s3.StatObject(c.Request.Context(), bucket, obj.Key, minio.StatObjectOptions{})
		if err != nil {
			continue
		}

		filename := stat.UserMetadata["Filename"]
		if filename == "" {
			filename = obj.Key
		}

		files = append(files, fileInfo{
			FileID:     obj.Key,
			Filename:   filename,
			Size:       obj.Size,
			UploadedAt: obj.LastModified.Format(time.RFC3339),
		})
	}

	if files == nil {
		files = []fileInfo{}
	}

	c.JSON(http.StatusOK, files)
}

func (s *server) deleteFile(c *gin.Context) {
	fileID := c.Param("file_id")
	fileID = strings.TrimPrefix(fileID, "/")

	if err := s.s3.RemoveObject(c.Request.Context(), bucket, fileID, minio.RemoveObjectOptions{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to delete file: %s", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (s *server) downloadFile(c *gin.Context) {
	fileID := c.Param("file_id")
	fileID = strings.TrimPrefix(fileID, "/")

	stat, err := s.s3.StatObject(c.Request.Context(), bucket, fileID, minio.StatObjectOptions{})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	filename := stat.UserMetadata["Filename"]
	if filename == "" {
		filename = fileID
	}

	url, err := s.s3Local.PresignedGetObject(c.Request.Context(), bucket, fileID, time.Hour, map[string][]string{
		"response-content-disposition": {fmt.Sprintf(`attachment; filename="%s"`, filename)},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to generate download URL: %s", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url.String(), "filename": filename})
}

// --- helpers ---

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	srv, err := newServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %s", err)
	}

	if os.Getenv("DEBUG") != "true" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "*")
		c.Header("Access-Control-Allow-Headers", "*")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	r.GET("/api/health", srv.health)
	r.POST("/api/upload/init", srv.uploadInit)
	r.POST("/api/upload/finish", srv.uploadFinish)
	r.POST("/api/upload/abort", srv.uploadAbort)
	r.GET("/api/files", srv.listFiles)
	r.DELETE("/api/files/*file_id", srv.deleteFile)
	r.GET("/api/files/*file_id", srv.downloadFile)

	addr := fmt.Sprintf(":%s", getEnv("PORT", "8000"))
	log.Printf("listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %s", err)
	}
}
