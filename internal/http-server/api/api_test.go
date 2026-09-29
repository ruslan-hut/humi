package api

import "testing"

func TestCacheControl(t *testing.T) {
	const forever = "public, max-age=31536000, immutable"

	tests := []struct {
		path string
		want string
	}{
		{"/main-INKJH662.js", forever},
		{"/chunk--NbucB8y.js", forever},
		{"/chunk-B2PX51-w.js", forever},
		{"/styles-XMXJBW2J.css", forever},
		{"/index.html", "no-cache"},
		{"/ngsw.json", "no-cache"},
		{"/ngsw-worker.js", "no-cache"},
		{"/safety-worker.js", "no-cache"},
		{"/manifest.webmanifest", "no-cache"},
		{"/icons/icon-192.png", "no-cache"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := cacheControl(tt.path); got != tt.want {
				t.Fatalf("cacheControl(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
