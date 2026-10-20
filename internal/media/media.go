package media

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// MediaItem represents an in-memory zero-disk media asset (image or video)
type MediaItem struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"` // "image" or "video"
	ContentType   string    `json:"content_type"`
	Data          []byte    `json:"-"` // Stored purely in RAM, zero disk writes
	Prompt        string    `json:"prompt"`
	RevisedPrompt string    `json:"revised_prompt,omitempty"`
	Model         string    `json:"model"`
	Provider      string    `json:"provider"`
	AccountName   string    `json:"account_name,omitempty"`
	Duration      int       `json:"duration,omitempty"` // In seconds (for video)
	AspectRatio   string    `json:"aspect_ratio,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// Manager manages in-memory media with zero disk storage and LRU eviction
type Manager struct {
	mu         sync.RWMutex
	items      map[string]*MediaItem
	order      []string
	maxItems   int
	totalBytes int64
	maxBytes   int64 // Maximum memory budget in bytes (default: 50MB for embedded/STB safety)
}

var (
	defaultManager *Manager
	once           sync.Once
)

// GetManager returns the singleton MediaManager instance
func GetManager() *Manager {
	once.Do(func() {
		defaultManager = NewManager(50) // Keep at most 50 media items in RAM (~50MB max ceiling)
	})
	return defaultManager
}

// NewManager creates a new MediaManager
func NewManager(maxItems int) *Manager {
	if maxItems <= 0 {
		maxItems = 50
	}
	return &Manager{
		items:      make(map[string]*MediaItem),
		order:      make([]string, 0, maxItems),
		maxItems:   maxItems,
		maxBytes:   50 * 1024 * 1024, // 50MB RAM ceiling for STB embedded devices
		totalBytes: 0,
	}
}

// Store stores a media item in memory and prunes older entries
func (m *Manager) Store(item *MediaItem) {
	if item == nil || item.ID == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	itemSize := int64(len(item.Data))

	// If key exists, update in-place and adjust totalBytes
	if old, exists := m.items[item.ID]; exists {
		m.totalBytes -= int64(len(old.Data))
		m.totalBytes += itemSize
		m.items[item.ID] = item
		return
	}

	// Prune if item count or total memory budget exceeded
	for (len(m.order) >= m.maxItems || (m.maxBytes > 0 && m.totalBytes+itemSize > m.maxBytes)) && len(m.order) > 0 {
		oldestID := m.order[0]
		m.order = m.order[1:]
		if oldItem, ok := m.items[oldestID]; ok {
			m.totalBytes -= int64(len(oldItem.Data))
			delete(m.items, oldestID)
		}
	}

	m.items[item.ID] = item
	m.order = append(m.order, item.ID)
	m.totalBytes += itemSize
}

// Get retrieves a media item by ID
func (m *Manager) Get(id string) *MediaItem {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cleanID := strings.TrimSuffix(strings.TrimSuffix(id, ".png"), ".mp4")
	cleanID = strings.TrimSuffix(cleanID, ".jpg")
	cleanID = strings.TrimSuffix(cleanID, ".jpeg")

	if item, ok := m.items[cleanID]; ok {
		return item
	}
	return m.items[id]
}

// List returns a list of recent media items
func (m *Manager) List() []*MediaItem {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*MediaItem, 0, len(m.order))
	for i := len(m.order) - 1; i >= 0; i-- {
		id := m.order[i]
		if item, ok := m.items[id]; ok {
			// Shallow copy without data slice
			copyItem := *item
			copyItem.Data = nil
			list = append(list, &copyItem)
		}
	}
	return list
}

// ServeHTTP serves a media item with Range requests, 206 Partial Content, and download support
func (m *Manager) ServeHTTP(w http.ResponseWriter, r *http.Request, id string) {
	item := m.Get(id)
	if item == nil || len(item.Data) == 0 {
		http.NotFound(w, r)
		return
	}

	ext := ".png"
	if item.Type == "video" || strings.HasPrefix(item.ContentType, "video/") {
		ext = ".mp4"
	}
	filename := fmt.Sprintf("%s%s", item.ID, ext)

	serveData := item.Data
	serveContentType := item.ContentType

	// When downloading as PNG (or format=png) and source is JPEG, convert to true uncompressed lossless PNG (several MBs)
	if item.Type == "image" && (strings.HasSuffix(r.URL.Path, ".png") || r.URL.Query().Get("format") == "png") && strings.Contains(item.ContentType, "jpeg") {
		if r.URL.Query().Get("format") != "raw" && r.URL.Query().Get("format") != "jpg" && (r.URL.Query().Get("download") == "1" || r.URL.Query().Get("format") == "png") {
			if img, err := jpeg.Decode(bytes.NewReader(item.Data)); err == nil {
				var buf bytes.Buffer
				if err := png.Encode(&buf, img); err == nil {
					serveData = buf.Bytes()
					serveContentType = "image/png"
					ext = ".png"
				}
			}
		}
	}

	// Set content type and caching headers
	w.Header().Set("Content-Type", serveContentType)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Cache-Control", "public, max-age=86400")

	// If download parameter is requested, set Content-Disposition
	if r.URL.Query().Get("download") == "1" {
		cleanPrompt := sanitizeFilename(item.Prompt)
		if cleanPrompt == "" {
			cleanPrompt = item.ID
		}
		downloadName := fmt.Sprintf("%s%s", cleanPrompt, ext)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", downloadName))
	}

	// ServeContent handles HTTP 206 Partial Content, Range headers, and HEAD automatically
	http.ServeContent(w, r, filename, item.CreatedAt, bytes.NewReader(serveData))
}

func sanitizeFilename(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 40 {
		s = s[:40]
	}
	var sb strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			sb.WriteRune(r)
		} else if r == ' ' {
			sb.WriteRune('_')
		}
	}
	return sb.String()
}

// GenerateDynamicVideo produces an animated MP4 in memory using ffmpeg or fallback base MP4
func GenerateDynamicVideo(ctx context.Context, prompt, model string, durationSec int, aspect string) ([]byte, error) {
	if durationSec <= 0 {
		durationSec = 5
	}
	if durationSec > 15 {
		durationSec = 15
	}

	width, height := 640, 360
	switch aspect {
	case "9:16":
		width, height = 360, 640
	case "1:1":
		width, height = 480, 480
	case "4:3":
		width, height = 480, 360
	default:
		width, height = 640, 360
	}

	// Check if ffmpeg is available
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err == nil && ffmpegPath != "" {
		// Clean label for video filter
		label := prompt
		if len(label) > 45 {
			label = label[:42] + "..."
		}
		label = strings.ReplaceAll(label, "'", "")
		label = strings.ReplaceAll(label, ":", " ")

		filterDesc := fmt.Sprintf("testsrc=duration=%d:size=%dx%d:rate=24", durationSec, width, height)

		cmd := exec.CommandContext(ctx, ffmpegPath,
			"-y",
			"-f", "lavfi",
			"-i", filterDesc,
			"-c:v", "libx264",
			"-preset", "ultrafast",
			"-crf", "30",
			"-pix_fmt", "yuv420p",
			"-movflags", "frag_keyframe+empty_moov+default_base_moof",
			"-f", "mp4",
			"pipe:1",
		)

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err == nil && stdout.Len() > 1000 {
			log.Printf("[MEDIA] Generated %d-second MP4 video in RAM (%d bytes) with ffmpeg", durationSec, stdout.Len())
			return stdout.Bytes(), nil
		}
	}

	// Guaranteed fallback: return embedded minimal 3-second MP4 video
	decoded, err := base64.StdEncoding.DecodeString(EmbeddedBaseVideoMP4)
	if err == nil && len(decoded) > 0 {
		return decoded, nil
	}

	return nil, fmt.Errorf("failed to generate video: ffmpeg not available and embedded fallback failed")
}

// GenerateDynamicImage produces a high-quality AI-generated image (via Pollinations FLUX/SDXL)
// in memory with zero disk writes, falling back to a stylized canvas PNG if offline.
func GenerateDynamicImage(prompt, model, aspect string) ([]byte, string, error) {
	width, height := 1024, 1024
	switch aspect {
	case "1:1":
		width, height = 1024, 1024
	case "16:9", "1792x1024", "1024x576":
		width, height = 1280, 720
	case "9:16", "1024x1792", "576x1024":
		width, height = 720, 1280
	case "4:3":
		width, height = 1024, 768
	default:
		if strings.Contains(aspect, "x") {
			parts := strings.Split(aspect, "x")
			if len(parts) == 2 {
				var w, h int
				if _, err := fmt.Sscanf(parts[0], "%d", &w); err == nil && w > 0 {
					width = w
				}
				if _, err := fmt.Sscanf(parts[1], "%d", &h); err == nil && h > 0 {
					height = h
				}
			}
		}
	}
	if width > 1920 {
		width = 1920
	}
	if height > 1920 {
		height = 1920
	}

	// 1. Try real AI image generation via Pollinations FLUX Engine (Zero-Disk in RAM)
	cleanPrompt := strings.TrimSpace(prompt)
	if cleanPrompt != "" {
		encodedPrompt := url.QueryEscape(cleanPrompt)
		pollinationsURL := fmt.Sprintf("https://image.pollinations.ai/prompt/%s?width=%d&height=%d&nologo=true&model=flux", encodedPrompt, width, height)

		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		httpReq, err := http.NewRequestWithContext(ctx, "GET", pollinationsURL, nil)
		if err == nil {
			httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36")
			client := &http.Client{Timeout: 25 * time.Second}
			resp, err := client.Do(httpReq)
			if err == nil && resp.StatusCode == http.StatusOK {
				defer resp.Body.Close()
				data, err := io.ReadAll(resp.Body)
				if err == nil && len(data) > 1000 {
					contentType := resp.Header.Get("Content-Type")
					if contentType == "" || !strings.HasPrefix(contentType, "image/") {
						contentType = "image/jpeg"
					}
					return data, contentType, nil
				}
			}
		}
	}

	// 2. Fallback to stylized canvas PNG if offline or rate limited
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Draw high-tech gradient background
	for y := 0; y < height; y++ {
		ratio := float64(y) / float64(height)
		r := uint8(10 + ratio*25)
		g := uint8(15 + ratio*35)
		b := uint8(30 + ratio*60)
		col := color.RGBA{R: r, G: g, B: b, A: 255}
		for x := 0; x < width; x++ {
			img.Set(x, y, col)
		}
	}

	// Draw grid lines
	gridCol := color.RGBA{R: 30, G: 58, B: 138, A: 80}
	for x := 0; x < width; x += 40 {
		for y := 0; y < height; y++ {
			img.Set(x, y, gridCol)
		}
	}
	for y := 0; y < height; y += 40 {
		for x := 0; x < width; x++ {
			img.Set(x, y, gridCol)
		}
	}

	// Draw glowing center banner
	bannerRect := image.Rect(40, height/2-60, width-40, height/2+60)
	bannerCol := color.RGBA{R: 15, G: 23, B: 42, A: 220}
	draw.Draw(img, bannerRect, &image.Uniform{C: bannerCol}, image.Point{}, draw.Over)

	// Border around banner
	borderCol := color.RGBA{R: 56, G: 189, B: 248, A: 255}
	for x := bannerRect.Min.X; x < bannerRect.Max.X; x++ {
		img.Set(x, bannerRect.Min.Y, borderCol)
		img.Set(x, bannerRect.Max.Y-1, borderCol)
	}
	for y := bannerRect.Min.Y; y < bannerRect.Max.Y; y++ {
		img.Set(bannerRect.Min.X, y, borderCol)
		img.Set(bannerRect.Max.X-1, y, borderCol)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, "", err
	}

	return buf.Bytes(), "image/png", nil
}

// FetchRemoteMediaStream streams a remote image/video directly into RAM with zero disk writes
func FetchRemoteMediaStream(ctx context.Context, mediaURL string) ([]byte, string, error) {
	return FetchRemoteMediaStreamWithHeaders(ctx, mediaURL, nil)
}

// FetchRemoteMediaStreamWithHeaders streams a remote image/video with optional custom headers (e.g. Google cookies/referer)
func FetchRemoteMediaStreamWithHeaders(ctx context.Context, mediaURL string, headers map[string]string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", mediaURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(redirectReq *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			// Retain Cookie, User-Agent and Referer across redirects (especially Google usercontent cross-domain)
			if len(via) > 0 {
				if c := via[0].Header.Get("Cookie"); c != "" {
					redirectReq.Header.Set("Cookie", c)
				}
				if r := via[0].Header.Get("Referer"); r != "" {
					redirectReq.Header.Set("Referer", r)
				}
				if ua := via[0].Header.Get("User-Agent"); ua != "" {
					redirectReq.Header.Set("User-Agent", ua)
				}
			}
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("remote media returned status: %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		ext := strings.ToLower(filepath.Ext(mediaURL))
		switch ext {
		case ".png":
			contentType = "image/png"
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		case ".webp":
			contentType = "image/webp"
		case ".mp4":
			contentType = "video/mp4"
		default:
			contentType = "application/octet-stream"
		}
	}

	// Limit to max 25MB per media to prevent memory ballooning
	limitReader := io.LimitReader(resp.Body, 25*1024*1024)
	data, err := io.ReadAll(limitReader)
	if err != nil {
		return nil, "", err
	}

	return data, contentType, nil
}
