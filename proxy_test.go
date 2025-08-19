package main

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestParseMode(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    Mode
		expectError bool
	}{
		{
			name:        "strict mode",
			input:       "strict",
			expected:    ModeStrict,
			expectError: false,
		},
		{
			name:        "warn mode",
			input:       "warn",
			expected:    ModeWarn,
			expectError: false,
		},
		{
			name:        "report mode",
			input:       "report",
			expected:    ModeReport,
			expectError: false,
		},
		{
			name:        "uppercase input",
			input:       "STRICT",
			expected:    ModeStrict,
			expectError: false,
		},
		{
			name:        "mixed case input",
			input:       "WaRn",
			expected:    ModeWarn,
			expectError: false,
		},
		{
			name:        "invalid mode",
			input:       "invalid",
			expected:    "",
			expectError: true,
		},
		{
			name:        "empty mode",
			input:       "",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseMode(tt.input)
			if (err != nil) != tt.expectError {
				t.Errorf("parseMode() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if result != tt.expected {
				t.Errorf("parseMode() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestIsUndocumentedEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "no matching operation",
			err:      errors.New("no matching operation found"),
			expected: true,
		},
		{
			name:     "path not found",
			err:      errors.New("path not found"),
			expected: true,
		},
		{
			name:     "operation not found",
			err:      errors.New("operation not found"),
			expected: true,
		},
		{
			name:     "case insensitive match",
			err:      errors.New("No Route Found"),
			expected: true,
		},
		{
			name:     "unrelated error",
			err:      errors.New("connection refused"),
			expected: false,
		},
		{
			name:     "empty error message",
			err:      errors.New(""),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isUndocumentedEndpoint(tt.err)
			if result != tt.expected {
				t.Errorf("isUndocumentedEndpoint() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestValidatingProxy_ReadResponseBody(t *testing.T) {
	tests := []struct {
		name          string
		contentLength string
		bodySize      int
		expectSkipped bool
	}{
		{
			name:          "small response",
			contentLength: "100",
			bodySize:      100,
			expectSkipped: false,
		},
		{
			name:          "large content-length header",
			contentLength: "20971520", // 20MB
			bodySize:      0,
			expectSkipped: true,
		},
		{
			name:          "no content-length but large body",
			contentLength: "",
			bodySize:      20971520, // 20MB
			expectSkipped: true,
		},
		{
			name:          "exactly at limit",
			contentLength: "10485760", // 10MB
			bodySize:      10485760,
			expectSkipped: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := make([]byte, tt.bodySize)
			for i := range body {
				body[i] = 'a'
			}

			resp := &http.Response{
				Header: make(http.Header),
				Body:   io.NopCloser(bytes.NewReader(body)),
			}

			if tt.contentLength != "" {
				resp.Header.Set("Content-Length", tt.contentLength)
			}

			vp := &ValidatingProxy{
				logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
			}

			result, err := vp.readResponseBody(resp)

			if err != nil {
				t.Errorf("readResponseBody() unexpected error: %v", err)
				return
			}

			if tt.expectSkipped {
				if result != nil {
					t.Errorf("readResponseBody() expected nil for large response, got %d bytes", len(result))
				}
			} else {
				if result == nil {
					t.Errorf("readResponseBody() expected body data, got nil")
				} else if len(result) != tt.bodySize {
					t.Errorf("readResponseBody() expected %d bytes, got %d", tt.bodySize, len(result))
				}
			}
		})
	}
}

func TestValidatingProxy_ReplaceResponseWithError(t *testing.T) {
	resp := &http.Response{
		Header:     make(http.Header),
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"original": "data"}`)),
	}
	resp.Header.Set("Content-Type", "application/json")
	resp.Header.Set("Content-Encoding", "gzip")
	resp.Header.Set("ETag", "123456")

	vp := &ValidatingProxy{}
	testErr := errors.New("test validation error")

	vp.replaceResponseWithError(resp, testErr)

	if resp.StatusCode != 500 {
		t.Errorf("replaceResponseWithError() status code = %d, expected 500", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("replaceResponseWithError() content-type = %q, expected application/json", resp.Header.Get("Content-Type"))
	}

	if resp.Header.Get("Content-Encoding") != "" {
		t.Errorf("replaceResponseWithError() should have removed Content-Encoding header")
	}
	if resp.Header.Get("ETag") != "" {
		t.Errorf("replaceResponseWithError() should have removed ETag header")
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("replaceResponseWithError() failed to read body: %v", err)
		return
	}

	if !strings.Contains(string(bodyBytes), "Response validation failed") {
		t.Errorf("replaceResponseWithError() body should contain error message")
	}
	if !strings.Contains(string(bodyBytes), "test validation error") {
		t.Errorf("replaceResponseWithError() body should contain validation error details")
	}

	expectedLength := len(bodyBytes)
	actualLength, _ := strconv.Atoi(resp.Header.Get("Content-Length"))
	if actualLength != expectedLength {
		t.Errorf("replaceResponseWithError() content-length = %d, expected %d", actualLength, expectedLength)
	}
}
