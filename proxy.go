/**
    SpecGate - A lightweight OpenAPI validation proxy for real-time API response validation.
    Copyright (C) 2025 Søren Johanson

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.
**/

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

type Mode string

const (
	ModeStrict Mode = "strict"
	ModeWarn   Mode = "warn"
	ModeReport Mode = "report"
)

type ValidatingProxy struct {
	spec     *openapi3.T
	upstream *url.URL
	proxy    *httputil.ReverseProxy
	mode     Mode
	logger   *slog.Logger
	router   routers.Router
}

func NewValidatingProxy(specPath, upstreamURL string, mode string) (*ValidatingProxy, error) {
	// Validate mode first
	validMode, err := parseMode(mode)
	if err != nil {
		return nil, err
	}

	loader := openapi3.NewLoader()

	var spec *openapi3.T

	// Check if specPath is a URL
	if strings.HasPrefix(specPath, "http://") || strings.HasPrefix(specPath, "https://") {
		specURL, parseErr := url.Parse(specPath)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid spec URL: %w", parseErr)
		}
		spec, err = loader.LoadFromURI(specURL)
	} else {
		spec, err = loader.LoadFromFile(specPath)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load spec: %w", err)
	}

	upstream, err := url.Parse(upstreamURL)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream URL: %w", err)
	}

	// Override the servers block with the upstream URL
	spec.Servers = []*openapi3.Server{
		{URL: upstreamURL},
	}

	logger := slog.New(&ColoredHandler{
		output: os.Stderr,
		level:  slog.LevelInfo,
	})

	router, _ := gorillamux.NewRouter(spec)

	vp := &ValidatingProxy{
		spec:     spec,
		upstream: upstream,
		mode:     validMode,
		logger:   logger,
		router:   router,
	}

	vp.proxy = &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = upstream.Scheme
			req.URL.Host = upstream.Host
			req.Host = upstream.Host
		},
		ModifyResponse: vp.validateResponse,
	}

	return vp, nil
}

func (vp *ValidatingProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vp.proxy.ServeHTTP(w, r)
}

func (vp *ValidatingProxy) validateResponse(resp *http.Response) error {
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return nil
	}

	const maxSize = 10 * 1024 * 1024 // 10MB
	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		if size, err := strconv.ParseInt(contentLength, 10, 64); err == nil && size > maxSize {
			vp.logger.Warn("Response too large, skipping validation", "size", size)
			return nil
		}
	}

	limited := io.LimitReader(resp.Body, maxSize+1)
	bodyBytes, err := io.ReadAll(limited)
	if err != nil {
		return err
	}

	if len(bodyBytes) > maxSize {
		vp.logger.Warn("Response too large, skipping validation", "size", len(bodyBytes))
		return nil
	}

	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	route, pathParams, err := vp.router.FindRoute(resp.Request)
	if err != nil {
		if isUndocumentedEndpoint(err) {
			vp.logger.Warn("Undocumented endpoint",
				"method", resp.Request.Method,
				"path", resp.Request.URL.Path)
			return nil
		} else {
			vp.logger.Error("Error finding route",
				"error", err,
				"method", resp.Request.Method,
				"path", resp.Request.URL.Path)
			return fmt.Errorf("route finding error: %w", err)
		}
	}

	// For validation, use a separate reader as the previous one has already been consumed
	// Otherwise, "Transferred partial file" errors will start showing up
	validationReader := io.NopCloser(bytes.NewReader(bodyBytes))

	// Use the request context to respect cancellation signals
	ctx := resp.Request.Context()
	input := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request:    resp.Request,
			PathParams: pathParams,
			Route:      route,
		},
		Status: resp.StatusCode,
		Header: resp.Header,
		Body:   validationReader,
	}

	if err := openapi3filter.ValidateResponse(ctx, input); err != nil {
		vp.logger.Error("Response validation failed",
			"error", err,
			"method", resp.Request.Method,
			"path", resp.Request.URL.Path,
			"status", resp.StatusCode)

		if vp.mode == ModeStrict {
			errorBody, _ := json.Marshal(map[string]string{
				"error":   "Response validation failed",
				"details": err.Error(),
			})

			// Update headers to match the new response
			resp.Body = io.NopCloser(bytes.NewReader(errorBody))
			resp.StatusCode = 500
			resp.Header.Set("Content-Type", "application/json")
			resp.Header.Set("Content-Length", strconv.Itoa(len(errorBody)))

			// Remove headers that are no longer valid for the error response
			resp.Header.Del("Content-Encoding")
			resp.Header.Del("Transfer-Encoding")
			resp.Header.Del("ETag")
			resp.Header.Del("Last-Modified")
		}
	}

	return nil
}

func parseMode(mode string) (Mode, error) {
	switch strings.ToLower(mode) {
	case "strict":
		return ModeStrict, nil
	case "warn":
		return ModeWarn, nil
	case "report":
		return ModeReport, nil
	default:
		return "", fmt.Errorf("invalid mode '%s': must be one of 'strict', 'warn', or 'report'", mode)
	}
}

func isUndocumentedEndpoint(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	undocumentedPatterns := []string{
		"no matching operation",
		"path not found",
		"no route found",
		"operation not found",
		"no match found",
		"unknown path",
	}

	for _, pattern := range undocumentedPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// ColoredHandler provides colored console output similar to zerolog
type ColoredHandler struct {
	output io.Writer
	level  slog.Level
	attrs  []slog.Attr
}

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorGreen  = "\033[32m"
	colorBlue   = "\033[34m"
	colorGray   = "\033[90m"
)

func (h *ColoredHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *ColoredHandler) Handle(ctx context.Context, record slog.Record) error {
	var color string
	switch record.Level {
	case slog.LevelError:
		color = colorRed
	case slog.LevelWarn:
		color = colorYellow
	case slog.LevelInfo:
		color = colorGreen
	case slog.LevelDebug:
		color = colorBlue
	default:
		color = colorReset
	}

	var builder strings.Builder
	builder.Grow(256) // Pre-allocate reasonable capacity

	timestamp := time.Now().Format("15:04:05")
	builder.WriteString(colorGray)
	builder.WriteString(timestamp)
	builder.WriteString(" ")
	builder.WriteString(color)
	builder.WriteString(record.Level.String())
	builder.WriteString(colorReset)

	builder.WriteString(" ")
	builder.WriteString(record.Message)

	record.Attrs(func(attr slog.Attr) bool {
		builder.WriteString(" ")
		builder.WriteString(attr.Key)
		builder.WriteString("=")
		builder.WriteString(fmt.Sprintf("%v", attr.Value))
		return true
	})

	for _, attr := range h.attrs {
		builder.WriteString(" ")
		builder.WriteString(attr.Key)
		builder.WriteString("=")
		builder.WriteString(fmt.Sprintf("%v", attr.Value))
	}

	builder.WriteString("\n")

	_, err := h.output.Write([]byte(builder.String()))
	return err
}

func (h *ColoredHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &ColoredHandler{
		output: h.output,
		level:  h.level,
		attrs:  newAttrs,
	}
}

func (h *ColoredHandler) WithGroup(name string) slog.Handler {
	return h
}
