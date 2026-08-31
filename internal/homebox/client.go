// Package homebox is a minimal read-only client for the HomeBox API.
//
// Deliberately not generated from the OpenAPI spec: this server exposes a
// handful of GETs, and a generated client would be several thousand lines
// of surface we do not use and would have to keep in step with upstream.
package homebox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to one HomeBox instance as one API key.
//
// HomeBox scopes data by GROUP, not by user: every item belongs to the group
// the key's owner belongs to. Two people in the same group see the same
// inventory, which for a household is the point rather than a limitation.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// New returns a client for baseURL (e.g. "http://homebox").
//
// The API lives under /api — swagger reports basePath "/api" and the paths
// themselves carry HomeBox's own /v1 version segment, so a full URL looks
// like http://homebox/api/v1/entities.
func New(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// get fetches path (relative to /api/v1) and decodes the JSON body.
func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	u := c.baseURL + "/api/v1" + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}

	// HomeBox's only auth scheme: its own bearer token. It does NOT accept
	// an OIDC token -- the OIDC endpoints are a browser redirect flow that
	// ends by issuing one of these.
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Cap the body: an HTML error page from a misconfigured proxy is
		// otherwise pasted whole into an LLM's context.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("GET %s: %s: %s", path, resp.Status, strings.TrimSpace(string(body)))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

// Raw fetches a path and returns the decoded JSON as-is.
//
// The tools hand HomeBox's own shapes straight through rather than
// remodelling them. A model reads the field names fine, and every struct we
// defined here would be one more thing to drift from upstream.
func (c *Client) Raw(ctx context.Context, path string, query url.Values) (any, error) {
	var out any
	if err := c.get(ctx, path, query, &out); err != nil {
		return nil, err
	}

	return out, nil
}

// Self returns the authenticated user, used as the startup health check:
// it proves the token works and reports which group it can see.
func (c *Client) Self(ctx context.Context) (name, email, group string, err error) {
	var out struct {
		Item struct {
			Name           string `json:"name"`
			Email          string `json:"email"`
			DefaultGroupID string `json:"defaultGroupId"`
		} `json:"item"`
	}
	if err := c.get(ctx, "/users/self", nil, &out); err != nil {
		return "", "", "", err
	}

	return out.Item.Name, out.Item.Email, out.Item.DefaultGroupID, nil
}
