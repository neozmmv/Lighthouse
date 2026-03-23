package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// ── Config ────────────────────────────────────────────────────────────────────

type appConfig struct {
	// S3 endpoint for internal bucket operations
	MinioEndpoint string
	// S3 endpoint used when generating upload presigned URLs (visible to Tor clients)
	MinioPublicEndpoint string
	// S3 endpoint used when generating download presigned URLs (localhost always)
	MinioLocalEndpoint string
	AccessKey          string
	SecretKey          string
	Bucket             string
	Debug              bool
}

func loadConfig() appConfig {
	getenv := func(key, def string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return def
	}
	withScheme := func(u string) string {
		if u != "" && !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
			return "http://" + u
		}
		return u
	}
	return appConfig{
		MinioEndpoint:       withScheme(getenv("MINIO_ENDPOINT", "http://minio:9000")),
		MinioPublicEndpoint: withScheme(os.Getenv("MINIO_PUBLIC_ENDPOINT")),
		MinioLocalEndpoint:  withScheme(getenv("MINIO_LOCAL_ENDPOINT", "http://127.0.0.1:9000")),
		// Support both naming conventions: MINIO_ACCESS_KEY (native) or MINIO_ROOT_USER (docker)
		AccessKey: func() string {
			if v := os.Getenv("MINIO_ACCESS_KEY"); v != "" {
				return v
			}
			return getenv("MINIO_ROOT_USER", "lighthouse")
		}(),
		SecretKey: func() string {
			if v := os.Getenv("MINIO_SECRET_KEY"); v != "" {
				return v
			}
			return getenv("MINIO_ROOT_PASSWORD", "lighthouse_secret")
		}(),
		Bucket: getenv("MINIO_BUCKET", "lighthouse"),
		Debug:  strings.ToLower(getenv("DEBUG", "false")) == "true",
	}
}

// ── S3 clients ────────────────────────────────────────────────────────────────

func newS3Client(endpoint, accessKey, secretKey string) *s3.Client {
	creds := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(creds),
	)
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}
	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})
}

// ── Server ────────────────────────────────────────────────────────────────────

type server struct {
	s3       *s3.Client // internal: all bucket operations
	s3Public *s3.Client // upload presigned URLs (sent to Tor clients)
	s3Local  *s3.Client // download presigned URLs (localhost only)
	bucket   string
	debug    bool
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// CORS — mirror the Python FastAPI CORSMiddleware config
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	p := r.URL.Path

	switch {
	case r.Method == http.MethodGet && p == "/api/health":
		s.handleHealth(w, r)

	case r.Method == http.MethodPost && p == "/api/upload/init":
		s.handleUploadInit(w, r)

	case r.Method == http.MethodPost && p == "/api/upload/finish":
		s.handleUploadFinish(w, r)

	case r.Method == http.MethodPost && p == "/api/upload/abort":
		s.handleUploadAbort(w, r)

	case r.Method == http.MethodGet && p == "/api/files":
		s.handleListFiles(w, r)

	// GET /api/files/{file_id}/download — file_id may contain slashes
	case r.Method == http.MethodGet &&
		strings.HasPrefix(p, "/api/files/") &&
		strings.HasSuffix(p, "/download"):
		fileID := p[len("/api/files/") : len(p)-len("/download")]
		s.handleDownloadFile(w, r, fileID)

	// DELETE /api/files/{file_id}
	case r.Method == http.MethodDelete && strings.HasPrefix(p, "/api/files/"):
		fileID := p[len("/api/files/"):]
		s.handleDeleteFile(w, r, fileID)

	default:
		http.NotFound(w, r)
	}
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func (s *server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type initUploadRequest struct {
	Filename    string `json:"filename"`
	TotalChunks int    `json:"total_chunks"`
}

func (s *server) handleUploadInit(w http.ResponseWriter, r *http.Request) {
	var body initUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Filename == "" || body.TotalChunks < 1 {
		writeError(w, http.StatusBadRequest, "filename and total_chunks are required")
		return
	}

	fileID := newUUID() + "-" + body.Filename

	resp, err := s.s3.CreateMultipartUpload(r.Context(), &s3.CreateMultipartUploadInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(fileID),
		Metadata: map[string]string{"filename": body.Filename},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	uploadID := aws.ToString(resp.UploadId)

	presigner := s3.NewPresignClient(s.s3Public)
	urls := make([]map[string]any, 0, body.TotalChunks)
	for partNum := 1; partNum <= body.TotalChunks; partNum++ {
		req, err := presigner.PresignUploadPart(r.Context(), &s3.UploadPartInput{
			Bucket:     aws.String(s.bucket),
			Key:        aws.String(fileID),
			UploadId:   aws.String(uploadID),
			PartNumber: aws.Int32(int32(partNum)),
		}, func(o *s3.PresignOptions) { o.Expires = 2 * time.Hour })
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to presign part: "+err.Error())
			return
		}
		urls = append(urls, map[string]any{"part_number": partNum, "url": req.URL})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"file_id":   fileID,
		"upload_id": uploadID,
		"urls":      urls,
	})
}

type part struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
}

type finishUploadRequest struct {
	FileID   string `json:"file_id"`
	UploadID string `json:"upload_id"`
	Parts    []part `json:"parts"`
}

func (s *server) handleUploadFinish(w http.ResponseWriter, r *http.Request) {
	var body finishUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	completed := make([]types.CompletedPart, len(body.Parts))
	for i, p := range body.Parts {
		completed[i] = types.CompletedPart{
			PartNumber: aws.Int32(int32(p.PartNumber)),
			ETag:       aws.String(p.ETag),
		}
	}
	sort.Slice(completed, func(i, j int) bool {
		return aws.ToInt32(completed[i].PartNumber) < aws.ToInt32(completed[j].PartNumber)
	})

	_, err := s.s3.CompleteMultipartUpload(r.Context(), &s3.CompleteMultipartUploadInput{
		Bucket:          aws.String(s.bucket),
		Key:             aws.String(body.FileID),
		UploadId:        aws.String(body.UploadID),
		MultipartUpload: &types.CompletedMultipartUpload{Parts: completed},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"file_id": body.FileID, "status": "ok"})
}

type abortUploadRequest struct {
	FileID   string `json:"file_id"`
	UploadID string `json:"upload_id"`
}

func (s *server) handleUploadAbort(w http.ResponseWriter, r *http.Request) {
	var body abortUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	_, err := s.s3.AbortMultipartUpload(r.Context(), &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(body.FileID),
		UploadId: aws.String(body.UploadID),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "aborted"})
}

type fileInfo struct {
	FileID     string `json:"file_id"`
	Filename   string `json:"filename"`
	Size       int64  `json:"size"`
	UploadedAt string `json:"uploaded_at"`
}

func (s *server) handleListFiles(w http.ResponseWriter, r *http.Request) {
	listResp, err := s.s3.ListObjectsV2(r.Context(), &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]fileInfo, 0, len(listResp.Contents))
	for _, obj := range listResp.Contents {
		headResp, err := s.s3.HeadObject(r.Context(), &s3.HeadObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    obj.Key,
		})
		filename := aws.ToString(obj.Key)
		if err == nil {
			if v := headResp.Metadata["filename"]; v != "" {
				filename = v
			}
		}
		uploadedAt := ""
		if obj.LastModified != nil {
			uploadedAt = obj.LastModified.UTC().Format(time.RFC3339)
		}
		result = append(result, fileInfo{
			FileID:     aws.ToString(obj.Key),
			Filename:   filename,
			Size:       aws.ToInt64(obj.Size),
			UploadedAt: uploadedAt,
		})
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *server) handleDeleteFile(w http.ResponseWriter, r *http.Request, fileID string) {
	_, err := s.s3.DeleteObject(r.Context(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fileID),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *server) handleDownloadFile(w http.ResponseWriter, r *http.Request, fileID string) {
	headResp, err := s.s3.HeadObject(r.Context(), &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fileID),
	})
	if err != nil {
		if isHTTP404(err) {
			writeError(w, http.StatusNotFound, "File not found")
		} else {
			writeError(w, http.StatusInternalServerError, "Error generating download URL: "+err.Error())
		}
		return
	}

	filename := headResp.Metadata["filename"]
	if filename == "" {
		filename = fileID
	}

	presigner := s3.NewPresignClient(s.s3Local)
	req, err := presigner.PresignGetObject(r.Context(), &s3.GetObjectInput{
		Bucket:                     aws.String(s.bucket),
		Key:                        aws.String(fileID),
		ResponseContentDisposition: aws.String(`attachment; filename="` + filename + `"`),
	}, func(o *s3.PresignOptions) { o.Expires = 1 * time.Hour })
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error generating download URL: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"url": req.URL, "filename": filename})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, detail string) {
	writeJSON(w, status, map[string]string{"detail": detail})
}

func isHTTP404(err error) bool {
	var respErr *smithyhttp.ResponseError
	return errors.As(err, &respErr) && respErr.HTTPStatusCode() == 404
}

func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("rand.Read failed: %v", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // RFC 4122 variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// ensureBucket creates the S3 bucket if it does not already exist.
func ensureBucket(ctx context.Context, client *s3.Client, bucket string) error {
	_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})
	if err == nil {
		return nil // already exists
	}
	if !isHTTP404(err) {
		return fmt.Errorf("checking bucket: %w", err)
	}
	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)})
	return err
}

// waitForTorHostname polls the Tor hidden service hostname file, returning the
// onion address when it appears. Returns "" on timeout.
func waitForTorHostname(path string, timeout time.Duration) string {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			if addr := strings.TrimSpace(string(data)); addr != "" {
				return addr
			}
		}
		time.Sleep(1 * time.Second)
	}
	return ""
}

// ── main ──────────────────────────────────────────────────────────────────────

// version is injected at build time via -ldflags "-X main.version=v1.2.3"
var version = "dev"

func main() {
	cfg := loadConfig()

	// Resolve the public endpoint for presigned upload URLs.
	// If not already set, wait for the Tor hidden service hostname (like entrypoint.sh).
	publicEndpoint := cfg.MinioPublicEndpoint
	if publicEndpoint == "" {
		hostnameFile := os.Getenv("TOR_HOSTNAME_FILE")
		if hostnameFile == "" {
			home, _ := os.UserHomeDir()
			hostnameFile = filepath.Join(home, ".lighthouse", "hidden_service", "hostname")
			// Also check the Docker path as fallback
			if _, err := os.Stat(hostnameFile); errors.Is(err, os.ErrNotExist) {
				if _, err2 := os.Stat("/var/lib/tor/hidden_service/hostname"); err2 == nil {
					hostnameFile = "/var/lib/tor/hidden_service/hostname"
				}
			}
		}
		log.Printf("Waiting for Tor hidden service hostname at %s...", hostnameFile)
		onion := waitForTorHostname(hostnameFile, 120*time.Second)
		if onion != "" {
			publicEndpoint = "http://" + onion
			log.Printf("Tor onion address detected: %s", publicEndpoint)
		} else {
			log.Println("Timeout waiting for Tor hostname, using MINIO_ENDPOINT as fallback")
			publicEndpoint = cfg.MinioEndpoint
		}
	}

	internalClient := newS3Client(cfg.MinioEndpoint, cfg.AccessKey, cfg.SecretKey)

	// Create the bucket if it doesn't exist yet (replaces the minio-init container).
	if err := ensureBucket(context.Background(), internalClient, cfg.Bucket); err != nil {
		log.Fatalf("failed to ensure bucket %q exists: %v", cfg.Bucket, err)
	}
	log.Printf("Bucket %q ready", cfg.Bucket)

	srv := &server{
		s3:       internalClient,
		s3Public: newS3Client(publicEndpoint, cfg.AccessKey, cfg.SecretKey),
		s3Local:  newS3Client(cfg.MinioLocalEndpoint, cfg.AccessKey, cfg.SecretKey),
		bucket:   cfg.Bucket,
		debug:    cfg.Debug,
	}

	addr := ":8000"
	log.Printf("Lighthouse backend %s listening on %s (debug=%v)", version, addr, cfg.Debug)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
