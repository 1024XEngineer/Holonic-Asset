package upload

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	qiniuclient "github.com/qiniu/go-sdk/v7/client"
	qiniustorage "github.com/qiniu/go-sdk/v7/storage"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
)

type bucketManagerStub struct {
	info       qiniustorage.FileInfo
	statErr    error
	statBucket string
	statKey    string
	batchOps   []string
	batchRet   []qiniustorage.BatchOpRet
	batchErr   error
	deleteErr  error
	deleted    []string
}

func (s *bucketManagerStub) BatchWithContext(
	_ context.Context,
	_ string,
	operations []string,
) ([]qiniustorage.BatchOpRet, error) {
	s.batchOps = append([]string(nil), operations...)
	return s.batchRet, s.batchErr
}

type formUploaderStub struct {
	key     string
	data    []byte
	extra   *qiniustorage.PutExtra
	uptoken string
	err     error
}

func (s *formUploaderStub) Put(
	_ context.Context,
	_ any,
	uptoken string,
	key string,
	data io.Reader,
	_ int64,
	extra *qiniustorage.PutExtra,
) error {
	s.key = key
	s.uptoken = uptoken
	s.extra = extra
	s.data, _ = io.ReadAll(data)
	return s.err
}

func (s *bucketManagerStub) Stat(bucket, key string) (qiniustorage.FileInfo, error) {
	s.statBucket, s.statKey = bucket, key
	return s.info, s.statErr
}

func (s *bucketManagerStub) Delete(bucket, key string) error {
	s.deleted = append(s.deleted, bucket+":"+key)
	return s.deleteErr
}

func validQiniuConfig() config.QiniuConfig {
	return config.QiniuConfig{
		AccessKey: "test-access-key",
		SecretKey: "test-secret-key",
		Bucket:    "asset-bucket",
		Domain:    "cdn.example.com",
	}
}

func TestNewQiniuStorageValidatesConfig(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*config.QiniuConfig)
		field  string
	}{
		{name: "access key", mutate: func(cfg *config.QiniuConfig) { cfg.AccessKey = "" }, field: "accessKey"},
		{name: "secret key", mutate: func(cfg *config.QiniuConfig) { cfg.SecretKey = "" }, field: "secretKey"},
		{name: "bucket", mutate: func(cfg *config.QiniuConfig) { cfg.Bucket = "" }, field: "bucket"},
		{name: "domain", mutate: func(cfg *config.QiniuConfig) { cfg.Domain = "" }, field: "domain"},
		{name: "invalid domain", mutate: func(cfg *config.QiniuConfig) { cfg.Domain = "://bad" }, field: "domain"},
		{name: "invalid upload URL", mutate: func(cfg *config.QiniuConfig) { cfg.UploadURL = "ftp://upload.example.com" }, field: "uploadURL"},
		{name: "invalid expiry", mutate: func(cfg *config.QiniuConfig) { cfg.UploadTokenExpiry = -time.Second }, field: "uploadTokenExpiry"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := validQiniuConfig()
			test.mutate(&cfg)
			store, err := NewQiniuStorage(cfg)
			if !errors.Is(err, ErrInvalidStorageConfig) {
				t.Fatalf("expected invalid storage config, got %v", err)
			}
			if !strings.Contains(err.Error(), test.field) {
				t.Fatalf("expected error to mention %q, got %v", test.field, err)
			}
			if store != nil {
				t.Fatalf("expected nil store, got %+v", store)
			}
		})
	}
}

func TestCreateUploadTargetSignsRestrictedToken(t *testing.T) {
	store, err := NewQiniuStorage(validQiniuConfig())
	if err != nil {
		t.Fatalf("create object store: %v", err)
	}
	store.random = strings.NewReader("0123456789abcdef")

	target, err := store.CreateUploadTarget(context.Background(), UploadRequest{
		ContentType:   "image/png; charset=binary",
		ContentLength: 128,
	})
	if err != nil {
		t.Fatalf("create upload target: %v", err)
	}

	wantKey := "uploads/30313233343536373839616263646566"
	if target.ObjectKey != wantKey {
		t.Fatalf("expected object key %q, got %q", wantKey, target.ObjectKey)
	}
	assertQiniuPrivateURL(t, target.ObjectURL, "https://cdn.example.com/"+wantKey)
	if target.UploadURL != defaultQiniuUploadURL {
		t.Fatalf("unexpected upload URL: %q", target.UploadURL)
	}

	policy := decodeUploadPolicy(t, target.UploadToken)
	if policy.Scope != "asset-bucket:"+wantKey || policy.SaveKey != wantKey || !policy.ForceSaveKey || policy.InsertOnly != 1 {
		t.Fatalf("token does not restrict the destination: %+v", policy)
	}
	if policy.FsizeMin != 128 || policy.FsizeLimit != 128 {
		t.Fatalf("token does not restrict file size: %+v", policy)
	}
	if policy.MimeLimit != "image/png" || policy.DetectMime != 1 {
		t.Fatalf("token does not restrict MIME type: %+v", policy)
	}
	wantDeadline := time.Now().Add(defaultUploadTokenTTL).Unix()
	if delta := int64(policy.Expires) - wantDeadline; delta < -1 || delta > 1 { //nolint:gosec // Qiniu deadlines use positive Unix timestamps.
		t.Fatalf("unexpected token deadline: got %d want about %d", policy.Expires, wantDeadline)
	}
}

func TestCreateUploadTargetUsesProvidedObjectKey(t *testing.T) {
	cfg := validQiniuConfig()
	cfg.UploadURL = "https://upload.example.com/"
	cfg.UploadTokenExpiry = 10 * time.Minute
	store, err := NewQiniuStorage(cfg)
	if err != nil {
		t.Fatalf("create object store: %v", err)
	}

	target, err := store.CreateUploadTarget(context.Background(), UploadRequest{
		ObjectKey:     "projects/7/reference image.png",
		ContentType:   "image/png",
		ContentLength: 8,
	})
	if err != nil {
		t.Fatalf("create upload target: %v", err)
	}
	if target.ObjectKey != "projects/7/reference image.png" {
		t.Fatalf("unexpected object key: %q", target.ObjectKey)
	}
	assertQiniuPrivateURL(t, target.ObjectURL, "https://cdn.example.com/projects/7/reference%20image.png")
	if target.UploadURL != "https://upload.example.com" {
		t.Fatalf("unexpected normalized upload URL: %q", target.UploadURL)
	}
}

func TestCreateUploadTargetRejectsUnsafeContentTypes(t *testing.T) {
	store, err := NewQiniuStorage(validQiniuConfig())
	if err != nil {
		t.Fatalf("create object store: %v", err)
	}

	for _, contentType := range []string{"not-a-mime", "image/*", "!application/json", "image/png; image/jpeg"} {
		t.Run(contentType, func(t *testing.T) {
			target, err := store.CreateUploadTarget(context.Background(), UploadRequest{
				ContentType:   contentType,
				ContentLength: 1,
			})
			if !errors.Is(err, ErrInvalidUploadRequest) {
				t.Fatalf("expected invalid upload request, got target=%+v err=%v", target, err)
			}
		})
	}
}

func TestGetObjectMetadataUsesBucketManager(t *testing.T) {
	store, err := NewQiniuStorage(validQiniuConfig())
	if err != nil {
		t.Fatalf("create object store: %v", err)
	}
	manager := &bucketManagerStub{info: qiniustorage.FileInfo{MimeType: "image/webp", Fsize: 42}}
	store.bucketManager = manager

	metadata, err := store.GetObjectMetadata(context.Background(), "uploads/object.webp")
	if err != nil {
		t.Fatalf("get object metadata: %v", err)
	}
	if manager.statBucket != "asset-bucket" || manager.statKey != "uploads/object.webp" {
		t.Fatalf("unexpected stat call: bucket=%q key=%q", manager.statBucket, manager.statKey)
	}
	if metadata.ObjectKey != "uploads/object.webp" || metadata.ContentType != "image/webp" || metadata.ContentLength != 42 {
		t.Fatalf("unexpected metadata: %+v", metadata)
	}
	assertQiniuPrivateURL(t, metadata.ObjectURL, "https://cdn.example.com/uploads/object.webp")
}

func TestGetObjectMetadataClassifiesMissingObject(t *testing.T) {
	store, err := NewQiniuStorage(validQiniuConfig())
	if err != nil {
		t.Fatalf("create object store: %v", err)
	}
	store.bucketManager = &bucketManagerStub{statErr: &qiniuclient.ErrorInfo{
		Code: 612,
		Err:  "no such file or directory",
	}}

	if _, err := store.GetObjectMetadata(context.Background(), "uploads/missing.png"); !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("expected object-not-found error, got %v", err)
	}
}

func TestReferenceNormalizationAndResolutionUsesManagedStorage(t *testing.T) {
	store, err := NewQiniuStorage(validQiniuConfig())
	if err != nil {
		t.Fatalf("create object store: %v", err)
	}
	store.now = func() time.Time { return time.Unix(1_700_000_000, 0) }

	ownURL := "https://cdn.example.com/projects/7/reference%20image.png?e=1700001800&token=signed"
	got, err := store.normalizeReference(ownURL)
	if err != nil {
		t.Fatalf("normalize own URL: %v", err)
	}
	if got != "projects/7/reference image.png" {
		t.Fatalf("expected own URL to normalize to key, got %q", got)
	}
	persisted, err := store.PersistReference(context.Background(), ownURL)
	if err != nil || persisted != "projects/7/reference image.png" {
		t.Fatalf("persist own URL = %q, err=%v", persisted, err)
	}
	resolved, err := store.ResolveReference(context.Background(), "projects/7/reference image.png")
	if err != nil {
		t.Fatalf("resolve object key: %v", err)
	}
	assertQiniuPrivateURL(t, resolved, "https://cdn.example.com/projects/7/reference%20image.png")
}

func TestReferencePersistenceAndResolutionRejectExternalURLs(t *testing.T) {
	store, err := NewQiniuStorage(validQiniuConfig())
	if err != nil {
		t.Fatalf("create object store: %v", err)
	}

	for _, value := range []string{
		"https://images.example.org/reference.png",
		"https://cdn.example.com.attacker.test/reference.png",
		"http://127.0.0.1:8080/admin",
		"http://10.0.0.5/internal",
		"//cdn.example.com/reference.png",
		"ftp://cdn.example.com/reference.png",
	} {
		t.Run(value, func(t *testing.T) {
			if _, err := store.PersistReference(context.Background(), value); !errors.Is(err, ErrUntrustedReference) {
				t.Fatalf("persist error = %v, want untrusted reference", err)
			}
			if _, err := store.ResolveReference(context.Background(), value); !errors.Is(err, ErrUntrustedReference) {
				t.Fatalf("resolve error = %v, want untrusted reference", err)
			}
		})
	}
}

func TestCreatePrivateURLUsesS3SignatureForQiniuS3Endpoint(t *testing.T) {
	cfg := validQiniuConfig()
	cfg.Domain = "https://feijiqilaifeiqilai1.s3.cn-east-1.qiniucs.com"
	cfg.DownloadURLExpiry = 30 * time.Minute
	store, err := NewQiniuStorage(cfg)
	if err != nil {
		t.Fatalf("create object store: %v", err)
	}
	store.now = func() time.Time { return time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC) }

	privateURL, err := store.privateURL(context.Background(), "uploads/reference image.png")
	if err != nil {
		t.Fatalf("create private URL: %v", err)
	}
	parsed, err := url.Parse(privateURL)
	if err != nil {
		t.Fatalf("parse private URL: %v", err)
	}
	query := parsed.Query()
	if query.Get("X-Amz-Algorithm") != "AWS4-HMAC-SHA256" ||
		!strings.Contains(query.Get("X-Amz-Credential"), "/cn-east-1/s3/aws4_request") ||
		query.Get("X-Amz-Expires") != "1800" || query.Get("X-Amz-Signature") == "" {
		t.Fatalf("unexpected S3 signature query: %v", query)
	}
	if parsed.EscapedPath() != "/uploads/reference%20image.png" {
		t.Fatalf("unexpected object path: %q", parsed.EscapedPath())
	}
}

func TestNewQiniuStorageRejectsS3DownloadExpiryOverSevenDays(t *testing.T) {
	cfg := validQiniuConfig()
	cfg.Domain = "https://asset-bucket.s3.cn-east-1.qiniucs.com"
	cfg.DownloadURLExpiry = 7*24*time.Hour + time.Second
	store, err := NewQiniuStorage(cfg)
	if !errors.Is(err, ErrInvalidStorageConfig) || store != nil {
		t.Fatalf("expected invalid S3 expiry, got store=%+v err=%v", store, err)
	}
}

func TestPersistReferenceUploadsDataURLAndReturnsObjectKey(t *testing.T) {
	store, err := NewQiniuStorage(validQiniuConfig())
	if err != nil {
		t.Fatalf("create object store: %v", err)
	}
	store.random = strings.NewReader("0123456789abcdef")
	uploader := &formUploaderStub{}
	store.uploader = uploader

	objectKey, err := store.PersistReference(
		context.Background(),
		"data:image/png;base64,aGVsbG8=",
	)
	if err != nil {
		t.Fatalf("persist reference: %v", err)
	}
	if objectKey != "uploads/30313233343536373839616263646566.png" {
		t.Fatalf("unexpected object key: %q", objectKey)
	}
	if uploader.key != objectKey || string(uploader.data) != "hello" || uploader.extra.MimeType != "image/png" || uploader.uptoken == "" {
		t.Fatalf("unexpected upload: key=%q data=%q extra=%+v", uploader.key, uploader.data, uploader.extra)
	}
}

func TestQiniuStoragePutsAndDeletesExplicitResourceKey(t *testing.T) {
	store, err := NewQiniuStorage(validQiniuConfig())
	if err != nil {
		t.Fatalf("create object store: %v", err)
	}
	uploader := &formUploaderStub{}
	bucketManager := &bucketManagerStub{}
	store.uploader = uploader
	store.bucketManager = bucketManager
	objectKey := "projects/42/scenery/batch/layers/1.png"

	if err := store.PutObject(context.Background(), objectKey, "image/png", []byte("png")); err != nil {
		t.Fatalf("put object: %v", err)
	}
	if uploader.key != objectKey || string(uploader.data) != "png" || uploader.extra.MimeType != "image/png" {
		t.Fatalf("unexpected upload: key=%q data=%q extra=%+v", uploader.key, uploader.data, uploader.extra)
	}
	if err := store.DeleteObject(context.Background(), objectKey); err != nil {
		t.Fatalf("delete object: %v", err)
	}
	if len(bucketManager.deleted) != 1 || bucketManager.deleted[0] != "asset-bucket:"+objectKey {
		t.Fatalf("unexpected deletes: %v", bucketManager.deleted)
	}
}

func TestPersistReferenceAtUploadsToProvidedObjectKey(t *testing.T) {
	store, err := NewQiniuStorage(validQiniuConfig())
	if err != nil {
		t.Fatalf("create object store: %v", err)
	}
	uploader := &formUploaderStub{}
	store.uploader = uploader

	if err := store.PersistReferenceAt(
		context.Background(),
		"uploads/prototype-2-unprocessed.png",
		"data:image/png;base64,aGVsbG8=",
	); err != nil {
		t.Fatalf("persist reference at key: %v", err)
	}
	if uploader.key != "uploads/prototype-2-unprocessed.png" ||
		string(uploader.data) != "hello" || uploader.extra.MimeType != "image/png" {
		t.Fatalf("unexpected exact-key upload: key=%q data=%q extra=%+v", uploader.key, uploader.data, uploader.extra)
	}
}

func TestDeleteObjectsUsesExactImmutableKeys(t *testing.T) {
	store, err := NewQiniuStorage(validQiniuConfig())
	if err != nil {
		t.Fatalf("create object store: %v", err)
	}
	manager := &bucketManagerStub{batchRet: []qiniustorage.BatchOpRet{{Code: 200}, {Code: 200}}}
	store.bucketManager = manager

	keys := []string{"uploads/tile-1.png", "uploads/tile-2.png"}
	if err := store.DeleteObjects(context.Background(), keys); err != nil {
		t.Fatalf("delete objects: %v", err)
	}
	want := []string{
		qiniustorage.URIDelete("asset-bucket", keys[0]),
		qiniustorage.URIDelete("asset-bucket", keys[1]),
	}
	if !slices.Equal(manager.batchOps, want) {
		t.Fatalf("unexpected delete operations: got %v want %v", manager.batchOps, want)
	}
}

func TestDeleteObjectsReportsPartialFailure(t *testing.T) {
	store, err := NewQiniuStorage(validQiniuConfig())
	if err != nil {
		t.Fatalf("create object store: %v", err)
	}
	failed := qiniustorage.BatchOpRet{Code: 500}
	failed.Data.Error = "delete unavailable"
	store.bucketManager = &bucketManagerStub{batchRet: []qiniustorage.BatchOpRet{{Code: 200}, failed}}

	err = store.DeleteObjects(context.Background(), []string{"uploads/tile-1.png", "uploads/tile-2.png"})
	if err == nil || !strings.Contains(err.Error(), "uploads/tile-2.png") || !strings.Contains(err.Error(), "delete unavailable") {
		t.Fatalf("expected exact partial-delete error, got %v", err)
	}
}

func assertQiniuPrivateURL(t *testing.T, value string, wantPrefix string) {
	t.Helper()
	parsed, err := url.Parse(value)
	if err != nil {
		t.Fatalf("parse private URL: %v", err)
	}
	if !strings.HasPrefix(value, wantPrefix+"?") || parsed.Query().Get("e") == "" || parsed.Query().Get("token") == "" {
		t.Fatalf("unexpected private URL: %q", value)
	}
}

func TestQiniuStoragePropagatesContextAndSDKErrors(t *testing.T) {
	store, err := NewQiniuStorage(validQiniuConfig())
	if err != nil {
		t.Fatalf("create object store: %v", err)
	}
	wantErr := errors.New("qiniu unavailable")
	store.bucketManager = &bucketManagerStub{statErr: wantErr}

	if _, err := store.GetObjectMetadata(context.Background(), "uploads/file"); !errors.Is(err, wantErr) {
		t.Fatalf("expected stat error, got %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.CreateUploadTarget(cancelled, UploadRequest{ContentType: "image/png", ContentLength: 1}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled create, got %v", err)
	}
}

func decodeUploadPolicy(t *testing.T, token string) qiniustorage.PutPolicy {
	t.Helper()
	parts := strings.Split(token, ":")
	if len(parts) != 3 {
		t.Fatalf("unexpected upload token format: %q", token)
	}
	encodedPolicy, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatalf("decode upload policy: %v", err)
	}
	var policy qiniustorage.PutPolicy
	if err := json.Unmarshal(encodedPolicy, &policy); err != nil {
		t.Fatalf("unmarshal upload policy: %v", err)
	}
	return policy
}

func TestPutArtifactValidatesAndUploadsArtifact(t *testing.T) {
	store, err := NewQiniuStorage(validQiniuConfig())
	if err != nil {
		t.Fatal(err)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := store.PutArtifact(cancelled, "exports/result.zip", "application/zip", []byte("zip")); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled context error = %v", err)
	}

	for _, test := range []struct {
		name  string
		key   string
		type_ string
		data  []byte
		want  error
	}{
		{name: "missing key", key: "", type_: "application/zip", data: []byte("zip"), want: ErrInvalidUploadRequest},
		{name: "invalid media type", key: "exports/result.zip", type_: ";", data: []byte("zip"), want: ErrInvalidObjectData},
		{name: "empty data", key: "exports/result.zip", type_: "application/zip", want: ErrInvalidObjectData},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := store.PutArtifact(context.Background(), test.key, test.type_, test.data); !errors.Is(err, test.want) {
				t.Fatalf("PutArtifact error = %v, want %v", err, test.want)
			}
		})
	}

	uploader := &formUploaderStub{}
	store.uploader = uploader
	if err := store.PutArtifact(context.Background(), " exports/result.zip ", "application/zip; charset=binary", []byte("zip")); err != nil {
		t.Fatalf("PutArtifact success error = %v", err)
	}
	if uploader.key != "exports/result.zip" || string(uploader.data) != "zip" || uploader.extra == nil || uploader.extra.MimeType != "application/zip" {
		t.Fatalf("unexpected artifact upload: key=%q data=%q extra=%+v", uploader.key, uploader.data, uploader.extra)
	}

	putErr := errors.New("upload unavailable")
	uploader.err = putErr
	if err := store.PutArtifact(context.Background(), "exports/result.zip", "application/zip", []byte("zip")); !errors.Is(err, putErr) {
		t.Fatalf("PutArtifact uploader error = %v, want %v", err, putErr)
	}
}

func newTestQiniuStorage(t *testing.T) *QiniuStorage {
	t.Helper()
	store, err := NewQiniuStorage(validQiniuConfig())
	if err != nil {
		t.Fatalf("create test qiniu storage: %v", err)
	}
	return store
}

func TestQiniuStoragePutObject(t *testing.T) {
	store := newTestQiniuStorage(t)

	// cancelled context
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := store.PutObject(canceledCtx, "assets/img.png", "image/png", []byte("data")); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	// invalid key
	if err := store.PutObject(context.Background(), "", "image/png", []byte("data")); !errors.Is(err, ErrInvalidUploadRequest) {
		t.Fatalf("expected ErrInvalidUploadRequest, got %v", err)
	}

	// non-image media type
	if err := store.PutObject(context.Background(), "assets/img.png", "application/json", []byte("data")); !errors.Is(err, ErrInvalidObjectData) {
		t.Fatalf("expected ErrInvalidObjectData, got %v", err)
	}

	// success
	uploader := &formUploaderStub{}
	store.uploader = uploader
	if err := store.PutObject(context.Background(), " assets/img.png ", "image/png; charset=utf-8", []byte("pngdata")); err != nil {
		t.Fatalf("PutObject success error: %v", err)
	}
	if uploader.key != "assets/img.png" || string(uploader.data) != "pngdata" {
		t.Fatalf("unexpected upload: key=%q data=%q", uploader.key, uploader.data)
	}
}

func TestQiniuStoragePersistReferenceAt(t *testing.T) {
	store := newTestQiniuStorage(t)

	// invalid key
	if err := store.PersistReferenceAt(context.Background(), "", "data:image/png;base64,aGVsbG8="); !errors.Is(err, ErrInvalidUploadRequest) {
		t.Fatalf("expected ErrInvalidUploadRequest, got %v", err)
	}

	// invalid data url
	if err := store.PersistReferenceAt(context.Background(), "assets/test.png", "invalid-url"); !errors.Is(err, ErrInvalidObjectData) {
		t.Fatalf("expected ErrInvalidObjectData, got %v", err)
	}

	// success
	uploader := &formUploaderStub{}
	store.uploader = uploader
	raw := []byte("hello png")
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw)
	if err := store.PersistReferenceAt(context.Background(), "assets/test.png", dataURL); err != nil {
		t.Fatalf("PersistReferenceAt error: %v", err)
	}
	if uploader.key != "assets/test.png" || string(uploader.data) != string(raw) {
		t.Fatalf("unexpected upload: key=%q data=%q", uploader.key, uploader.data)
	}
}

func TestQiniuStorageDeleteObject(t *testing.T) {
	store := newTestQiniuStorage(t)

	// cancelled context
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := store.DeleteObject(canceledCtx, "assets/img.png"); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	// invalid key
	if err := store.DeleteObject(context.Background(), " "); !errors.Is(err, ErrInvalidUploadRequest) {
		t.Fatalf("expected ErrInvalidUploadRequest, got %v", err)
	}

	// bucket error
	bm := &bucketManagerStub{deleteErr: errors.New("delete failed")}
	store.bucketManager = bm
	if err := store.DeleteObject(context.Background(), "assets/img.png"); err == nil {
		t.Fatal("expected delete error")
	}

	// success
	bm.deleteErr = nil
	bm.deleted = nil
	if err := store.DeleteObject(context.Background(), " assets/img.png "); err != nil {
		t.Fatalf("DeleteObject error: %v", err)
	}
	if len(bm.deleted) != 1 || bm.deleted[0] != store.bucket+":assets/img.png" {
		t.Fatalf("unexpected deleted key: %v", bm.deleted)
	}
}

func TestQiniuStorageDeleteObjects(t *testing.T) {
	store := newTestQiniuStorage(t)

	// cancelled context
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := store.DeleteObjects(canceledCtx, []string{"assets/img.png"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	// empty slice
	if err := store.DeleteObjects(context.Background(), nil); err != nil {
		t.Fatalf("empty slice error: %v", err)
	}

	// invalid key in slice
	if err := store.DeleteObjects(context.Background(), []string{""}); !errors.Is(err, ErrInvalidUploadRequest) {
		t.Fatalf("expected ErrInvalidUploadRequest, got %v", err)
	}

	// success
	bm := &bucketManagerStub{
		batchRet: []qiniustorage.BatchOpRet{
			{Code: 200},
			{Code: 200},
		},
	}
	store.bucketManager = bm
	if err := store.DeleteObjects(context.Background(), []string{"assets/1.png", "assets/2.png"}); err != nil {
		t.Fatalf("DeleteObjects error: %v", err)
	}
	if len(bm.batchOps) != 2 {
		t.Fatalf("expected 2 batch operations, got %d", len(bm.batchOps))
	}

	// batch error
	bm.batchErr = errors.New("batch error")
	if err := store.DeleteObjects(context.Background(), []string{"assets/1.png", "assets/2.png"}); err == nil {
		t.Fatal("expected batch error")
	}

	// partial item error in batch
	bm.batchErr = nil
	bm.batchRet = []qiniustorage.BatchOpRet{
		{Code: 200},
		{Code: 404},
	}
	if err := store.DeleteObjects(context.Background(), []string{"assets/1.png", "assets/2.png"}); err == nil {
		t.Fatal("expected partial error")
	}
}

func TestQiniuStorageNewObjectKey(t *testing.T) {
	store := newTestQiniuStorage(t)

	// invalid media type
	if _, err := store.NewObjectKey("application/json"); !errors.Is(err, ErrInvalidObjectData) {
		t.Fatalf("expected ErrInvalidObjectData, got %v", err)
	}

	// valid
	key, err := store.NewObjectKey("image/png")
	if err != nil {
		t.Fatalf("NewObjectKey error: %v", err)
	}
	if !strings.HasSuffix(key, ".png") {
		t.Fatalf("expected .png suffix, got %q", key)
	}
}
