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
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
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
}

func NewValidatingProxy(specPath, upstreamURL string, mode string) (*ValidatingProxy, error) {
	loader := openapi3.NewLoader()

	var spec *openapi3.T
	var err error

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

	// Create colored console handler similar to zerolog
	logger := slog.New(&ColoredHandler{
		output: os.Stderr,
		level:  slog.LevelInfo,
	})

	vp := &ValidatingProxy{
		spec:     spec,
		upstream: upstream,
		mode:     Mode(mode),
		logger:   logger,
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
	// Only validate JSON responses
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return nil
	}

	// Read and buffer the body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// Skip huge responses
	if len(bodyBytes) > 10*1024*1024 {
		vp.logger.Warn("Response too large, skipping validation")
		return nil
	}

	// Find the operation in the spec
	router, _ := gorillamux.NewRouter(vp.spec)
	route, pathParams, err := router.FindRoute(resp.Request)
	if err != nil {
		vp.logger.Warn("Undocumented endpoint",
			"method", resp.Request.Method,
			"path", resp.Request.URL.Path)
		return nil
	}

	// Validate using the library's built-in validator
	ctx := context.Background()
	input := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request:    resp.Request,
			PathParams: pathParams,
			Route:      route,
		},
		Status: resp.StatusCode,
		Header: resp.Header,
		Body:   io.NopCloser(bytes.NewReader(bodyBytes)),
	}

	if err := openapi3filter.ValidateResponse(ctx, input); err != nil {
		vp.logger.Error("Response validation failed",
			"error", err,
			"method", resp.Request.Method,
			"path", resp.Request.URL.Path,
			"status", resp.StatusCode)

		// In strict mode, replace the response with an error
		if vp.mode == ModeStrict {
			errorBody, _ := json.Marshal(map[string]string{
				"error":   "Response validation failed",
				"details": err.Error(),
			})
			resp.Body = io.NopCloser(bytes.NewReader(errorBody))
			resp.StatusCode = 500
		}
	}

	return nil
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

	timestamp := time.Now().Format("15:04:05")

	fmt.Fprintf(h.output, "%s%s %s%s %s",
		colorGray, timestamp,
		color, record.Level.String(),
		colorReset)

	fmt.Fprintf(h.output, " %s", record.Message)

	record.Attrs(func(attr slog.Attr) bool {
		fmt.Fprintf(h.output, " %s=%v", attr.Key, attr.Value)
		return true
	})

	for _, attr := range h.attrs {
		fmt.Fprintf(h.output, " %s=%v", attr.Key, attr.Value)
	}

	fmt.Fprintln(h.output)
	return nil
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
