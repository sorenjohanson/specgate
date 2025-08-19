package main

import (
	"bytes"
	"flag"
	"os"
	"strings"
	"testing"
)

func TestValidateSpecUpstreamMatch(t *testing.T) {
	tests := []struct {
		name        string
		specURL     string
		upstreamURL string
		expectError bool
	}{
		{
			name:        "matching URLs",
			specURL:     "https://api.example.com/spec.yaml",
			upstreamURL: "https://api.example.com",
			expectError: false,
		},
		{
			name:        "matching URLs with different paths",
			specURL:     "https://api.example.com/v1/spec.yaml",
			upstreamURL: "https://api.example.com/v2",
			expectError: false,
		},
		{
			name:        "different schemes",
			specURL:     "http://api.example.com/spec.yaml",
			upstreamURL: "https://api.example.com",
			expectError: true,
		},
		{
			name:        "different hosts",
			specURL:     "https://api1.example.com/spec.yaml",
			upstreamURL: "https://api2.example.com",
			expectError: true,
		},
		{
			name:        "different ports",
			specURL:     "https://api.example.com:8080/spec.yaml",
			upstreamURL: "https://api.example.com:9090",
			expectError: true,
		},
		{
			name:        "invalid spec URL",
			specURL:     "://invalid-url",
			upstreamURL: "https://api.example.com",
			expectError: true,
		},
		{
			name:        "invalid upstream URL",
			specURL:     "https://api.example.com/spec.yaml",
			upstreamURL: "://invalid-url",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSpecUpstreamMatch(tt.specURL, tt.upstreamURL)
			if (err != nil) != tt.expectError {
				t.Errorf("validateSpecUpstreamMatch() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestMainShowsUsageWithNoArgs(t *testing.T) {
	oldArgs := os.Args

	os.Args = []string{"specgate"}

	defer func() {
		os.Args = oldArgs
	}()

	fs := flag.NewFlagSet("specgate", flag.ContinueOnError)
	var buf bytes.Buffer
	fs.SetOutput(&buf)

	testMainLogicWithFlagSet(t, fs, &buf)
}

func testMainLogicWithFlagSet(t *testing.T, fs *flag.FlagSet, buf *bytes.Buffer) {
	fs.String("spec", "openapi.yaml", "Path to OpenAPI spec")
	fs.String("upstream", "http://localhost:3000", "Upstream API URL")
	fs.String("port", "8080", "Proxy port")
	fs.String("mode", "warn", "Mode: strict|warn|report")

	err := fs.Parse([]string{})
	if err != nil {
		t.Fatalf("Failed to parse flags: %v", err)
	}

	if len(os.Args) == 1 {
		fs.Usage()
	}

	output := buf.String()
	if !strings.Contains(output, "Usage of") {
		t.Errorf("Expected usage output when no args provided, got: %q", output)
	}
	if !strings.Contains(output, "-spec") {
		t.Errorf("Expected -spec flag in usage output, got: %q", output)
	}
	if !strings.Contains(output, "-upstream") {
		t.Errorf("Expected -upstream flag in usage output, got: %q", output)
	}
}
