package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jempe/sitemap_scanner/internal/jsonlog"
	sitemapscanner "github.com/jempe/sitemap_scanner/sitemap_scanner"
	"github.com/patrickmn/go-cache"
)

const version = "1.0.0"

type config struct {
	port     int
	username string
	password string
}

type SitemapRequest struct {
	URL          string `json:"url"`
	RefreshCache bool   `json:"refresh_cache"`
}

var logger *jsonlog.Logger
var cfg config
var wg sync.WaitGroup
var sitemapCache *cache.Cache

func main() {
	logger = jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	// Initialize cache with 24 hour expiration and 30 minute cleanup interval
	sitemapCache = cache.New(24*time.Hour, 30*time.Minute)

	// API Web Server Settings
	flag.IntVar(&cfg.port, "port", 4000, "API server port")

	// Authentication settings
	flag.StringVar(&cfg.username, "username", "", "Username for basic authentication")
	flag.StringVar(&cfg.password, "password", "", "Password for basic authentication")

	flag.Parse()

	// Wrap the handler with basic authentication if credentials are provided
	if cfg.username != "" && cfg.password != "" {
		logger.PrintInfo("Basic authentication enabled", nil)
		http.HandleFunc("/get-sitemap", basicAuth(handleGetSitemap))
	} else {
		logger.PrintInfo("Basic authentication disabled", nil)
		http.HandleFunc("/get-sitemap", handleGetSitemap)
	}
	logger.PrintInfo("Starting server", map[string]string{
		"port": fmt.Sprintf("%d", cfg.port),
	})

	err := serve()
	if err != nil {
		logger.PrintFatal(err, nil)
	}
}

// basicAuth is a middleware that wraps an http.HandlerFunc with basic authentication
func basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get credentials from the request header
		username, password, ok := r.BasicAuth()
		if !ok {
			// No credentials provided, return 401 Unauthorized
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check if credentials are valid using constant-time comparison to prevent timing attacks
		usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(cfg.username)) == 1
		passwordMatch := subtle.ConstantTimeCompare([]byte(password), []byte(cfg.password)) == 1

		if !usernameMatch || !passwordMatch {
			// Invalid credentials, return 401 Unauthorized
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Credentials are valid, call the next handler
		next(w, r)
	}
}

func handleGetSitemap(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		errMessage := map[string]string{
			"error": "Method not allowed",
		}
		apiResponse(w, http.StatusMethodNotAllowed, errMessage)
		return
	}

	// Parse JSON request
	var req SitemapRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		errMessage := map[string]string{
			"error": "Invalid JSON",
		}
		apiResponse(w, http.StatusBadRequest, errMessage)
		return
	}

	// Validate URL
	if req.URL == "" {
		errMessage := map[string]string{
			"error": "URL is required",
		}
		apiResponse(w, http.StatusBadRequest, errMessage)
		return
	}

	if req.RefreshCache {
		// Remove from cache
		sitemapCache.Delete(req.URL)
		logger.PrintInfo("Cache refreshed for URL", map[string]string{
			"url": req.URL,
		})
	}

	// Check cache first
	if cachedData, found := sitemapCache.Get(req.URL); found {
		logger.PrintInfo("Cache hit for URL", map[string]string{
			"url": req.URL,
		})
		apiResponse(w, http.StatusOK, map[string]any{
			"sitemap": cachedData,
		})
		return
	}

	// Cache miss - fetch sitemap
	logger.PrintInfo("Cache miss for URL, fetching sitemap", map[string]string{
		"url": req.URL,
	})

	sitemapData, err := sitemapscanner.GetSitemap(req.URL)

	if err != nil {
		errMessage := map[string]string{
			"error": err.Error(),
		}
		apiResponse(w, http.StatusInternalServerError, errMessage)
		return
	}

	// Store in cache for 24 hours
	sitemapCache.Set(req.URL, sitemapData, cache.DefaultExpiration)

	// Return success response
	apiResponse(w, http.StatusOK, map[string]any{
		"sitemap": sitemapData,
	})
}

func apiResponse(w http.ResponseWriter, status int, message any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(message)
}

func serve() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      nil,
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	shutdownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		logger.PrintInfo("caught signal", map[string]string{
			"signal": s.String(),
		})

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		logger.PrintInfo("completing background tasks", map[string]string{
			"addr": srv.Addr,
		})

		wg.Wait()
		shutdownError <- nil
	}()

	logger.PrintInfo("starting server", map[string]string{
		"addr": srv.Addr,
	})

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownError
	if err != nil {
		return err
	}

	logger.PrintInfo("stopped server", map[string]string{
		"addr":    srv.Addr,
		"version": version,
	})

	return nil
}
