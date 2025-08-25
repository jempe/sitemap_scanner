package sitemapscanner

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// SitemapURL represents a URL entry in the sitemap
type SitemapURL struct {
	SiteMapIndexURL string `json:"sitemap"`
	Location        string `json:"loc" xml:"loc"`
	LastModified    string `json:"lastmod,omitempty" xml:"lastmod,omitempty"`
	ChangeFreq      string `json:"changefreq,omitempty" xml:"changefreq,omitempty"`
	Priority        string `json:"priority,omitempty" xml:"priority,omitempty"`
}

// Sitemap represents the sitemap structure
type Sitemap struct {
	XMLName xml.Name     `xml:"urlset"`
	URLs    []SitemapURL `json:"urls" xml:"url"`
}

// SitemapIndex represents a sitemap index file
type SitemapIndex struct {
	XMLName  xml.Name          `xml:"sitemapindex"`
	Sitemaps []SitemapIndexURL `json:"sitemaps" xml:"sitemap"`
}

// SitemapIndexURL represents a sitemap reference in an index
type SitemapIndexURL struct {
	Location     string `json:"loc" xml:"loc"`
	LastModified string `json:"lastmod,omitempty" xml:"lastmod,omitempty"`
}

// SitemapResult represents the final result
type SitemapResult struct {
	URLs     []SitemapURL `json:"urls"`
	Sitemaps []string     `json:"sitemap_urls"`
	Error    string       `json:"error,omitempty"`
}

// GetSitemap retrieves sitemap data by first checking robots.txt and returns it as JSON
func GetSitemap(targetURL string) (SitemapResult, error) {
	// Parse the target URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return SitemapResult{}, err
	}

	// Ensure we have a scheme
	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "https"
	}

	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	// Get sitemap URLs from robots.txt
	sitemapURLs, err := getSitemapURLsFromRobots(baseURL)
	if err != nil || len(sitemapURLs) == 0 {
		// Fallback to common sitemap locations
		sitemapURLs = []string{
			baseURL + "/sitemap.xml",
			baseURL + "/sitemap_index.xml",
			baseURL + "/sitemaps.xml",
		}
	}

	var allURLs []SitemapURL
	var validSitemaps []string

	// Process each sitemap URL
	for _, sitemapURL := range sitemapURLs {
		urls, err := processSitemap(sitemapURL)
		if err == nil && len(urls) > 0 {
			allURLs = append(allURLs, urls...)
			validSitemaps = append(validSitemaps, sitemapURL)
		}
	}

	result := SitemapResult{
		URLs:     allURLs,
		Sitemaps: validSitemaps,
	}

	if len(allURLs) == 0 {
		result.Error = "No sitemap data found"
	}

	return result, nil
}

// getSitemapURLsFromRobots fetches robots.txt and extracts sitemap URLs
func getSitemapURLsFromRobots(baseURL string) ([]string, error) {
	robotsURL := baseURL + "/robots.txt"

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(robotsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("robots.txt not found: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Extract sitemap URLs using regex
	sitemapRegex := regexp.MustCompile(`(?i)sitemap:\s*(.+)`)
	matches := sitemapRegex.FindAllStringSubmatch(string(body), -1)

	var sitemapURLs []string
	for _, match := range matches {
		if len(match) > 1 {
			sitemapURL := strings.TrimSpace(match[1])
			sitemapURLs = append(sitemapURLs, sitemapURL)
		}
	}

	return sitemapURLs, nil
}

// processSitemap fetches and parses a sitemap XML file
func processSitemap(sitemapURL string) ([]SitemapURL, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(sitemapURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sitemap not found: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Try to parse as sitemap index first
	var sitemapIndex SitemapIndex
	if err := xml.Unmarshal(body, &sitemapIndex); err == nil && len(sitemapIndex.Sitemaps) > 0 {
		// This is a sitemap index, process each sitemap
		var allURLs []SitemapURL
		for _, indexSitemap := range sitemapIndex.Sitemaps {
			urls, err := processSitemap(indexSitemap.Location)
			if err == nil {
				allURLs = append(allURLs, urls...)
			}
		}
		return allURLs, nil
	}

	// Try to parse as regular sitemap
	var sitemap Sitemap
	if err := xml.Unmarshal(body, &sitemap); err != nil {
		return nil, fmt.Errorf("failed to parse sitemap XML: %v", err)
	}

	for i, _ := range sitemap.URLs {
		sitemap.URLs[i].SiteMapIndexURL = sitemapURL
	}

	return sitemap.URLs, nil
}
