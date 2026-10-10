package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendToProviderNtfyUsesPlainTextPublishRequest(t *testing.T) {
	tests := []struct {
		name              string
		token             string
		wantAuthorization string
	}{
		{name: "without token"},
		{name: "with token", token: "test-token", wantAuthorization: "Bearer test-token"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			type capturedRequest struct {
				method        string
				path          string
				contentType   string
				title         string
				priority      string
				authorization string
				body          string
				err           error
			}

			requests := make(chan capturedRequest, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				requests <- capturedRequest{
					method:        r.Method,
					path:          r.URL.Path,
					contentType:   r.Header.Get("Content-Type"),
					title:         r.Header.Get("Title"),
					priority:      r.Header.Get("Priority"),
					authorization: r.Header.Get("Authorization"),
					body:          string(body),
					err:           err,
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			cfg := ProviderConfig{
				Url:    server.URL + "/mos",
				Method: http.MethodPost,
				Token:  test.token,
				Headers: map[string]string{
					"Content-Type":  "application/json",
					"Authorization": "Bearer {{.Token}}",
				},
				Body: map[string]interface{}{
					"topic":    "mos",
					"title":    "{{.Title}}",
					"message":  "{{.Message}}",
					"priority": "{{.Priority}}",
				},
				AlertMapping: map[string]interface{}{
					"warning": float64(4),
				},
			}

			msg := Message{
				Title:    "MOS - ntfy test",
				Message:  "plain notification text",
				Priority: "warning",
			}
			sendToProvider("ntfy", cfg, msg)

			got := <-requests
			if got.err != nil {
				t.Fatalf("reading request body: %v", got.err)
			}
			if got.method != http.MethodPost {
				t.Errorf("method = %q, want %q", got.method, http.MethodPost)
			}
			if got.path != "/mos" {
				t.Errorf("path = %q, want %q", got.path, "/mos")
			}
			if got.contentType != "text/plain; charset=utf-8" {
				t.Errorf("Content-Type = %q, want text/plain", got.contentType)
			}
			if got.title != msg.Title {
				t.Errorf("Title = %q, want %q", got.title, msg.Title)
			}
			if got.priority != "4" {
				t.Errorf("Priority = %q, want %q", got.priority, "4")
			}
			if got.authorization != test.wantAuthorization {
				t.Errorf("Authorization = %q, want %q", got.authorization, test.wantAuthorization)
			}
			if got.body != msg.Message {
				t.Errorf("body = %q, want only message text %q", got.body, msg.Message)
			}
		})
	}
}
