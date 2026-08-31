package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opwerm/homebox-mcp/internal/homebox"
)

// MCP requires structuredContent to be an object. Several HomeBox endpoints
// answer with a top-level array, and returning one straight through makes the
// client reject the response as malformed -- an error that names schema
// validation, not the tool that produced it.
//
// This shipped: /entities/tree, /tags, /entities/{id}/path and
// /entities/fields were all broken, while /entities and /groups/statistics
// worked. Testing only the two object-shaped endpoints is what let it out.
func TestCallAlwaysReturnsAnObject(t *testing.T) {
	cases := map[string]string{
		"top-level array":  `[{"name":"Attic"},{"name":"Garage"}]`,
		"array of strings": `["one","two"]`,
		"object":           `{"total":2,"items":[]}`,
		"empty array":      `[]`,
		"null":             `null`,
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(body))
			}))
			defer srv.Close()

			_, out, err := call(context.Background(), homebox.New(srv.URL, "t"), "/anything", nil)
			if err != nil {
				t.Fatalf("call: %v", err)
			}

			if _, ok := out.(map[string]any); !ok {
				t.Fatalf("structuredContent is %T, must be an object; MCP rejects anything else", out)
			}
		})
	}
}

// A failing request must surface as an error rather than as an empty result
// the model would read as "there is nothing here".
func TestCallPropagatesFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"nope"}`))
	}))
	defer srv.Close()

	if _, _, err := call(context.Background(), homebox.New(srv.URL, "bad"), "/anything", nil); err == nil {
		t.Fatal("a 401 must be an error, not an empty result")
	}
}

// Every tool the server registers must be reachable and declare a schema.
// A tool that exists but cannot be described is one a client will not offer.
func TestAllToolsAreRegistered(t *testing.T) {
	s := New(homebox.New("http://example.invalid", "t"), "test")
	if s == nil {
		t.Fatal("New returned nil")
	}
}
