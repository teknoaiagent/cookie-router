package media

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMediaManagerStoreAndGet(t *testing.T) {
	mgr := NewManager(3)

	item1 := &MediaItem{
		ID:          "img_1",
		Type:        "image",
		ContentType: "image/png",
		Data:        []byte("fakepngdata"),
		Prompt:      "test cat",
		CreatedAt:   time.Now(),
	}
	mgr.Store(item1)

	got := mgr.Get("img_1")
	if got == nil || string(got.Data) != "fakepngdata" {
		t.Fatalf("expected to retrieve item1 data")
	}

	// Test extension trimming
	gotExt := mgr.Get("img_1.png")
	if gotExt == nil || gotExt.ID != "img_1" {
		t.Fatalf("expected to retrieve item1 via img_1.png")
	}

	// Store more items to test LRU eviction
	mgr.Store(&MediaItem{ID: "img_2", Type: "image", Data: []byte("2"), CreatedAt: time.Now()})
	mgr.Store(&MediaItem{ID: "img_3", Type: "image", Data: []byte("3"), CreatedAt: time.Now()})
	mgr.Store(&MediaItem{ID: "img_4", Type: "image", Data: []byte("4"), CreatedAt: time.Now()})

	// img_1 should have been evicted
	if mgr.Get("img_1") != nil {
		t.Fatalf("expected img_1 to be evicted when maxItems=3 exceeded")
	}
	if mgr.Get("img_4") == nil {
		t.Fatalf("expected img_4 to exist")
	}
}

func TestMediaServeHTTPRangeRequests(t *testing.T) {
	mgr := NewManager(5)
	videoData := []byte("0123456789abcdefghij") // 20 bytes

	mgr.Store(&MediaItem{
		ID:          "vid_test",
		Type:        "video",
		ContentType: "video/mp4",
		Data:        videoData,
		Prompt:      "robot dancing",
		CreatedAt:   time.Now(),
	})

	// Test full GET
	req := httptest.NewRequest("GET", "/v1/videos/vid_test.mp4", nil)
	rr := httptest.NewRecorder()
	mgr.ServeHTTP(rr, req, "vid_test.mp4")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if rr.Header().Get("Accept-Ranges") != "bytes" {
		t.Errorf("expected Accept-Ranges: bytes")
	}
	if rr.Header().Get("Content-Type") != "video/mp4" {
		t.Errorf("expected Content-Type: video/mp4, got %s", rr.Header().Get("Content-Type"))
	}

	// Test Range Request (bytes 5-10)
	rangeReq := httptest.NewRequest("GET", "/v1/videos/vid_test.mp4", nil)
	rangeReq.Header.Set("Range", "bytes=5-10")
	rrRange := httptest.NewRecorder()
	mgr.ServeHTTP(rrRange, rangeReq, "vid_test.mp4")

	if rrRange.Code != http.StatusPartialContent {
		t.Fatalf("expected status 206 Partial Content, got %d", rrRange.Code)
	}
	if rrRange.Body.String() != "56789a" {
		t.Errorf("expected slice '56789a', got %q", rrRange.Body.String())
	}
}

func TestGenerateDynamicImageAndVideo(t *testing.T) {
	// Test Image Generation
	imgBytes, cType, err := GenerateDynamicImage("Seekor kucing lucu", "gemini-imagen", "16:9")
	if err != nil {
		t.Fatalf("failed to generate dynamic image: %v", err)
	}
	fmt.Printf("Dynamic Image Result: size=%d, type=%s\n", len(imgBytes), cType)
	if len(imgBytes) < 500 {
		t.Fatalf("image too small, got %d bytes", len(imgBytes))
	}

	// Test Video Generation (uses ffmpeg or embedded fallback)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	vidBytes, err := GenerateDynamicVideo(ctx, "Pemandangan gunung salju", "veo-2", 3, "16:9")
	if err != nil {
		t.Fatalf("failed to generate dynamic video: %v", err)
	}
	if len(vidBytes) < 1000 {
		t.Fatalf("video too small, got %d bytes", len(vidBytes))
	}
}
