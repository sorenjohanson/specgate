package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func main() {
	var (
		specPath = flag.String("spec", "openapi.yaml", "Path to OpenAPI spec")
		upstream = flag.String("upstream", "http://localhost:3000", "Upstream API URL")
		port     = flag.String("port", "8080", "Proxy port")
		mode     = flag.String("mode", "warn", "Mode: strict|warn|report")
	)
	flag.Parse()

	// Check if spec URL matches upstream URL and warn user
	if strings.HasPrefix(*specPath, "http://") || strings.HasPrefix(*specPath, "https://") {
		if err := validateSpecUpstreamMatch(*specPath, *upstream); err != nil {
			fmt.Printf("WARNING: %s\n", err.Error())
			fmt.Print("Do you want to continue? (y/N): ")
			
			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				log.Fatal("Failed to read user input:", err)
			}
			
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Println("Aborted.")
				os.Exit(1)
			}
		}
	}

	proxy, err := NewValidatingProxy(*specPath, *upstream, *mode)
	if err != nil {
		log.Fatal("Failed to create proxy:", err)
	}

	fmt.Printf("Starting validation proxy on: %s\n", *port)
	fmt.Printf("Proxying to %s\n", *upstream)
	fmt.Printf("Mode: %s\n", *mode)

	if err := http.ListenAndServe(":"+*port, proxy); err != nil {
		log.Fatal(err)
	}
}

func validateSpecUpstreamMatch(specURL, upstreamURL string) error {
	specParsed, err := url.Parse(specURL)
	if err != nil {
		return fmt.Errorf("invalid spec URL: %s", specURL)
	}
	
	upstreamParsed, err := url.Parse(upstreamURL)
	if err != nil {
		return fmt.Errorf("invalid upstream URL: %s", upstreamURL)
	}
	
	// Compare host and scheme
	if specParsed.Host != upstreamParsed.Host || specParsed.Scheme != upstreamParsed.Scheme {
		return fmt.Errorf("spec URL (%s) does not match upstream URL (%s)", 
			specParsed.Scheme+"://"+specParsed.Host, 
			upstreamParsed.Scheme+"://"+upstreamParsed.Host)
	}
	
	return nil
}
