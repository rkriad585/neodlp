package extractor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type testExtractor struct {
	matches  []string
	result   *NativeResult
	err      error
	extractC int
}

func (e *testExtractor) Match(url string) bool {
	for _, m := range e.matches {
		if strings.Contains(url, m) {
			return true
		}
	}
	return false
}

func (e *testExtractor) Extract(_ context.Context, url string) (*NativeResult, error) {
	e.extractC++
	if e.err != nil {
		return nil, e.err
	}
	return e.result, nil
}

func resetExtractors() {
	extractors = nil
}

func TestRegisterAndFind(t *testing.T) {
	resetExtractors()

	e := &testExtractor{matches: []string{"example.com"}}
	Register(e)

	found, ok := Find("https://example.com/video")
	if !ok {
		t.Fatal("expected to find extractor")
	}
	if found != e {
		t.Fatal("expected same extractor")
	}
}

func TestFindNoMatch(t *testing.T) {
	resetExtractors()

	_, ok := Find("https://notregistered.com/video")
	if ok {
		t.Fatal("expected no extractor found")
	}
}

func TestExtractReturnsResult(t *testing.T) {
	resetExtractors()

	e := &testExtractor{
		matches: []string{"mysite.com"},
		result:  &NativeResult{Title: "Test", MediaURL: "https://cdn.mysite.com/video.mp4", Platform: "mysite"},
	}
	Register(e)

	result, err := Extract(context.Background(), "https://mysite.com/video/123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Title != "Test" {
		t.Fatalf("expected title 'Test', got %q", result.Title)
	}
	if result.MediaURL != "https://cdn.mysite.com/video.mp4" {
		t.Fatalf("unexpected media URL: %s", result.MediaURL)
	}
	if result.Platform != "mysite" {
		t.Fatalf("unexpected platform: %s", result.Platform)
	}
}

func TestExtractNoExtractor(t *testing.T) {
	resetExtractors()

	_, err := Extract(context.Background(), "https://unknown.com/video")
	if err == nil {
		t.Fatal("expected error for unknown URL")
	}
}

func TestExtractReturnsError(t *testing.T) {
	resetExtractors()

	e := &testExtractor{
		matches: []string{"broken.com"},
		err:     fmt.Errorf("extraction failed"),
	}
	Register(e)

	_, err := Extract(context.Background(), "https://broken.com/video")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegisterMultipleExtractors(t *testing.T) {
	resetExtractors()

	e1 := &testExtractor{matches: []string{"site1.com"}, result: &NativeResult{Title: "Site1", Platform: "site1"}}
	e2 := &testExtractor{matches: []string{"site2.com"}, result: &NativeResult{Title: "Site2", Platform: "site2"}}
	Register(e1)
	Register(e2)

	r1, err := Extract(context.Background(), "https://site1.com/video")
	if err != nil {
		t.Fatal(err)
	}
	if r1.Title != "Site1" {
		t.Fatalf("expected Site1, got %s", r1.Title)
	}

	r2, err := Extract(context.Background(), "https://site2.com/video")
	if err != nil {
		t.Fatal(err)
	}
	if r2.Title != "Site2" {
		t.Fatalf("expected Site2, got %s", r2.Title)
	}
}

func TestThreadsExtractorMatch(t *testing.T) {
	resetExtractors()
	t.Cleanup(func() { resetExtractors() })

	Register(&threadsExtractor{})

	tests := []struct {
		url   string
		match bool
	}{
		{"https://www.threads.net/@user/post/DYj6_WzgrXC", true},
		{"https://threads.net/@user/post/abc123", true},
		{"https://threads.com/@user/post/abc123", true},
		{"https://www.threads.com/@u/p/xyz", true},
		{"https://www.instagram.com/p/abc123", false},
		{"https://www.youtube.com/watch?v=abc", false},
	}

	for _, tt := range tests {
		_, ok := Find(tt.url)
		if ok != tt.match {
			t.Errorf("Find(%q) = %v, want %v", tt.url, ok, tt.match)
		}
	}
}

func TestIMDBExtractorMatch(t *testing.T) {
	resetExtractors()
	t.Cleanup(func() { resetExtractors() })

	Register(&imdbExtractor{})

	tests := []struct {
		url   string
		match bool
	}{
		{"https://www.imdb.com/video/vi486853401", true},
		{"https://imdb.com/video/vi123456789", true},
		{"https://www.imdb.com/video/vi123/?ref_=ext_shr_lnk", true},
		{"https://www.imdb.com/title/tt0111161", false},
		{"https://www.youtube.com/watch?v=abc", false},
	}

	for _, tt := range tests {
		_, ok := Find(tt.url)
		if ok != tt.match {
			t.Errorf("Find(%q) = %v, want %v", tt.url, ok, tt.match)
		}
	}
}

func TestShouldFallback(t *testing.T) {
	tests := []struct {
		err      error
		fallback bool
	}{
		{fmt.Errorf("Unsupported URL: https://threads.com/..."), true},
		{fmt.Errorf("Unable to extract video data"), true},
		{fmt.Errorf("Unsupported site: example.com"), true},
		{fmt.Errorf("not supported by yt-dlp"), true},
		{fmt.Errorf("Please report this issue on GitHub"), true},
		{fmt.Errorf("exec: yt-dlp: executable file not found"), true},
		{fmt.Errorf("exec: no command"), true},
		{fmt.Errorf("HTTP Error 403: Forbidden"), false},
		{fmt.Errorf("HTTP Error 404: Not Found"), false},
		{fmt.Errorf("connection refused"), false},
		{nil, false},
	}

	for _, tt := range tests {
		got := ShouldFallback(tt.err)
		if got != tt.fallback {
			t.Errorf("ShouldFallback(%v) = %v, want %v", tt.err, got, tt.fallback)
		}
	}
}

func TestIsNativeSupported(t *testing.T) {
	resetExtractors()
	t.Cleanup(func() { resetExtractors() })

	Register(&threadsExtractor{})

	if !IsNativeSupported("https://www.threads.net/@u/p/abc") {
		t.Fatal("expected threads to be supported")
	}
	if IsNativeSupported("https://www.youtube.com/watch?v=abc") {
		t.Fatal("expected youtube to not be supported")
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal title", "normal title"},
		{"file/with\\slashes:and*stars?and\"quotes<more>and|pipes", "file_with_slashes_and_stars_and_quotes_more_and_pipes"},
		{"  trimmed  ", "trimmed"},
		{strings.Repeat("a", 250), strings.Repeat("a", 200)},
	}

	for _, tt := range tests {
		got := sanitizeFilename(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSanitizeFilenameEmpty(t *testing.T) {
	got := sanitizeFilename("")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestTruncateTitle(t *testing.T) {
	short := "Short title"
	if truncateTitle(short) != "Short title" {
		t.Fatalf("expected 'Short title', got %q", truncateTitle(short))
	}

	long := strings.Repeat("x", 150)
	got := truncateTitle(long)
	if len(got) != 120 {
		t.Fatalf("expected 120 chars, got %d: %q", len(got), got)
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatal("expected trailing elipsis")
	}
}

func TestTruncateTitleCleansNewlines(t *testing.T) {
	input := "Title\nwith\rnewlines"
	got := truncateTitle(input)
	if strings.Contains(got, "\n") || strings.Contains(got, "\r") {
		t.Fatalf("newlines not removed: %q", got)
	}
}

func TestDetectExt(t *testing.T) {
	tests := []struct {
		mime string
		want string
	}{
		{"video/mp4", "mp4"},
		{"video/webm", "webm"},
		{"video/x-matroska", "mkv"},
		{"video/quicktime", "mov"},
		{"video/x-msvideo", "avi"},
		{"image/jpeg", "jpg"},
		{"image/png", "png"},
		{"image/webp", "webp"},
		{"image/gif", "gif"},
		{"application/octet-stream", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := detectExt(tt.mime)
		if got != tt.want {
			t.Errorf("detectExt(%q) = %q, want %q", tt.mime, got, tt.want)
		}
	}
}

func TestDeepFind(t *testing.T) {
	data := map[string]any{
		"key1": "value1",
		"nested": map[string]any{
			"key1": "value2",
			"key2": []any{
				map[string]any{"key1": "value3"},
			},
		},
	}

	results := deepFind(data, "key1")
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
}

func TestDeepFindNoMatch(t *testing.T) {
	data := map[string]any{"foo": "bar"}
	results := deepFind(data, "nonexistent")
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestPickBestVideo(t *testing.T) {
	videos := []any{
		map[string]any{"url": "low.mp4", "height": 480.0},
		map[string]any{"url": "high.mp4", "height": 1080.0},
		map[string]any{"url": "medium.mp4", "height": 720.0},
	}

	url := pickBestVideo(videos)
	if url != "high.mp4" {
		t.Fatalf("expected 'high.mp4', got %q", url)
	}
}

func TestPickBestVideoEmpty(t *testing.T) {
	url := pickBestVideo([]any{})
	if url != "" {
		t.Fatalf("expected empty, got %q", url)
	}
}

func TestPickBestImage(t *testing.T) {
	candidates := []any{
		map[string]any{"url": "small.jpg", "width": 100.0, "height": 100.0},
		map[string]any{"url": "large.jpg", "width": 1920.0, "height": 1080.0},
	}

	url := pickBestImage(candidates)
	if url != "large.jpg" {
		t.Fatalf("expected 'large.jpg', got %q", url)
	}
}

func TestPickBestImageEmpty(t *testing.T) {
	url := pickBestImage([]any{})
	if url != "" {
		t.Fatalf("expected empty, got %q", url)
	}
}

func TestIsTimeoutError(t *testing.T) {
	tests := []struct {
		err    error
		isTout bool
	}{
		{fmt.Errorf("context deadline exceeded"), true},
		{fmt.Errorf("Client.Timeout exceeded"), true},
		{fmt.Errorf("request timed out"), true},
		{fmt.Errorf("connection refused"), false},
		{fmt.Errorf("some other error"), false},
	}

	for _, tt := range tests {
		got := isTimeoutError(tt.err)
		if got != tt.isTout {
			t.Errorf("isTimeoutError(%v) = %v, want %v", tt.err, got, tt.isTout)
		}
	}
}

func TestIsExecError(t *testing.T) {
	tests := []struct {
		err     error
		isExec  bool
	}{
		{fmt.Errorf("exec: no command"), true},
		{fmt.Errorf("exec: \"yt-dlp\": executable file not found"), true},
		{fmt.Errorf("fork/exec: no such file or directory"), true},
		{fmt.Errorf("HTTP 403"), false},
		{fmt.Errorf("timeout"), false},
	}

	for _, tt := range tests {
		got := isExecError(tt.err)
		if got != tt.isExec {
			t.Errorf("isExecError(%v) = %v, want %v", tt.err, got, tt.isExec)
		}
	}
}

func TestSynthesizeInfo(t *testing.T) {
	r := &NativeResult{Title: "Test Video", MediaURL: "https://cdn.example.com/vid.mp4", Platform: "example"}
	info := synthesizeInfo(r, "https://example.com/video/123")

	if info.Title == nil || *info.Title != "Test Video" {
		t.Fatalf("unexpected title")
	}
	if info.URL == nil || *info.URL != "https://example.com/video/123" {
		t.Fatalf("unexpected URL")
	}
	if info.Extractor == nil || *info.Extractor != "example" {
		t.Fatalf("unexpected extractor")
	}
	if info.WebpageURL == nil || *info.WebpageURL != "https://example.com/video/123" {
		t.Fatalf("unexpected webpage URL")
	}
}

func TestTryNativeReturnsExtractError(t *testing.T) {
	resetExtractors()
	t.Cleanup(func() { resetExtractors() })

	_, err := TryNative(context.Background(), "https://unknown.com/video")
	if err == nil {
		t.Fatal("expected error for unknown URL")
	}
}

func TestDownloadWithFallbackNativeOnly(t *testing.T) {
	resetExtractors()
	t.Cleanup(func() { resetExtractors() })

	e := &testExtractor{
		matches: []string{"mysite.com"},
		err:     errors.New("test: no network in unit test"),
	}
	Register(e)

	err := DownloadWithFallback(context.Background(), "https://mysite.com/video", t.TempDir(), true)
	if err == nil {
		t.Fatal("expected error from fake extractor")
	}
}
