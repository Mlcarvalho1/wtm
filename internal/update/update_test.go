package update

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func withServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	prev := apiBase
	apiBase = srv.URL
	t.Cleanup(func() { apiBase = prev })
}

func TestCheckLatest_FromRelease(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/releases/latest") {
			_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v0.2.0"})
			return
		}
		http.NotFound(w, r)
	})

	got, err := CheckLatest()
	if err != nil {
		t.Fatalf("CheckLatest: %v", err)
	}
	if got != "0.2.0" {
		t.Fatalf("got %q, want %q", got, "0.2.0")
	}
}

func TestCheckLatest_FallsBackToTags(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/releases/latest"):
			http.NotFound(w, r)
		case strings.HasSuffix(r.URL.Path, "/tags"):
			_ = json.NewEncoder(w).Encode([]map[string]string{
				{"name": "v0.1.0"},
				{"name": "v0.3.0"},
				{"name": "v0.2.0"},
				{"name": "not-a-version"},
			})
		default:
			http.NotFound(w, r)
		}
	})

	got, err := CheckLatest()
	if err != nil {
		t.Fatalf("CheckLatest: %v", err)
	}
	if got != "0.3.0" {
		t.Fatalf("got %q, want %q", got, "0.3.0")
	}
}

func TestCheckLatest_NoneYet(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/releases/latest"):
			http.NotFound(w, r)
		case strings.HasSuffix(r.URL.Path, "/tags"):
			_ = json.NewEncoder(w).Encode([]map[string]string{})
		default:
			http.NotFound(w, r)
		}
	})

	got, err := CheckLatest()
	if err != nil {
		t.Fatalf("CheckLatest: %v", err)
	}
	if got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestCheckLatest_ServerError(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	if _, err := CheckLatest(); err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestInstall_NoGoOnPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	err := Install("0.0.1")
	if err == nil || !strings.Contains(err.Error(), "go toolchain not found") {
		t.Fatalf("got %v, want a go-toolchain-not-found error", err)
	}
}
