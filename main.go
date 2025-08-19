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
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func main() {
	var (
		specPath = flag.String("spec", "openapi.yaml", "Path to OpenAPI spec")
		upstream = flag.String("upstream", "http://localhost:3000", "Upstream API URL")
		port     = flag.String("port", "8080", "Proxy port")
		mode     = flag.String("mode", "warn", "Mode: strict|warn|report")
	)
	flag.Parse()

	// GPL required copyright notice
	fmt.Println("SpecGate Copyright (C) 2025 Søren Johanson")
	fmt.Println("This program comes with ABSOLUTELY NO WARRANTY.")
	fmt.Println("This is free software, and you are welcome to redistribute it")
	fmt.Println("under certain conditions; see LICENSE file for details.")
	fmt.Println()

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

	fmt.Printf("Starting validation proxy on port: %s\n", *port)
	fmt.Printf("Proxying to: %s\n", *upstream)
	fmt.Printf("Mode: %s\n", *mode)

	server := &http.Server{
		Addr:         ":" + *port,
		Handler:      proxy,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
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

	if specParsed.Host != upstreamParsed.Host || specParsed.Scheme != upstreamParsed.Scheme {
		return fmt.Errorf("spec URL (%s) does not match upstream URL (%s)",
			specParsed.Scheme+"://"+specParsed.Host,
			upstreamParsed.Scheme+"://"+upstreamParsed.Host)
	}

	return nil
}
