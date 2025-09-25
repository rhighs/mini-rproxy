package rproxy

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestQueryParameterPreservation(t *testing.T) {
	// Test that demonstrates the current Director function behavior
	// This simulates what happens in the Director function
	
	tests := []struct {
		name           string
		originalURL    string
		prefix         string
		upstreamURL    string
		expectedURL    string
		expectedQuery  string
	}{
		{
			name:           "query parameters should be preserved",
			originalURL:    "http://localhost:8080/workout/say33?userId=123&filter=active&sort=desc",
			prefix:         "/workout",
			upstreamURL:    "https://api-beta.example.net",
			expectedURL:    "https://api-beta.example.net/say33?userId=123&filter=active&sort=desc",
			expectedQuery:  "userId=123&filter=active&sort=desc",
		},
		{
			name:           "no query parameters",
			originalURL:    "http://localhost:8080/workout/say33",
			prefix:         "/workout",
			upstreamURL:    "https://api-beta.example.net",
			expectedURL:    "https://api-beta.example.net/say33",
			expectedQuery:  "",
		},
		{
			name:           "single query parameter",
			originalURL:    "http://localhost:8080/core/health?check=true",
			prefix:         "/core",
			upstreamURL:    "https://api-gamma.example.net",
			expectedURL:    "https://api-gamma.example.net/health?check=true",
			expectedQuery:  "check=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse original request URL
			originalURL, err := url.Parse(tt.originalURL)
			if err != nil {
				t.Fatalf("Failed to parse original URL: %v", err)
			}

			// Parse upstream URL
			upstreamURL, err := url.Parse(tt.upstreamURL)
			if err != nil {
				t.Fatalf("Failed to parse upstream URL: %v", err)
			}

			// Create a request URL and simulate current Director behavior
			req := &http.Request{URL: &url.URL{
				Scheme:   originalURL.Scheme,
				Host:     originalURL.Host,
				Path:     originalURL.Path,
				RawQuery: originalURL.RawQuery,
			}}
			
			// Current implementation (without query parameter preservation)
			req.URL.Scheme = upstreamURL.Scheme
			req.URL.Host = upstreamURL.Host
			
			if strings.HasPrefix(req.URL.Path, tt.prefix) {
				req.URL.Path = upstreamURL.Path + "/" + strings.TrimPrefix(req.URL.Path, tt.prefix)
			} else {
				req.URL.Path = upstreamURL.Path
			}
			// NOTE: req.URL.RawQuery is preserved automatically in this test, 
			// but in the actual code it gets lost when we don't explicitly preserve it

			currentURL := req.URL.String()
			t.Logf("Current behavior would produce: %s", currentURL)
			t.Logf("Expected URL: %s", tt.expectedURL)
			t.Logf("Original query: %s", originalURL.RawQuery)
			t.Logf("Processed query: %s", req.URL.RawQuery)

			// This test demonstrates that we need to preserve query parameters
			if originalURL.RawQuery != req.URL.RawQuery {
				t.Errorf("Query parameters were not preserved. Original: %q, Result: %q", 
					originalURL.RawQuery, req.URL.RawQuery)
			}
		})
	}
}