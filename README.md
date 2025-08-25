# Sitemap Scanner

A Go web service that fetches and parses XML sitemaps from websites with built-in caching and optional basic authentication.

## Features

- **Fast sitemap fetching**: Efficiently retrieves and parses XML sitemaps
- **Smart caching**: Configurable cache duration (default 24 hours) to reduce server load
- **Cache refresh**: Force refresh cached data via API parameter
- **Basic authentication**: Optional username/password protection
- **JSON API**: Clean REST API with JSON responses
- **Structured logging**: JSON-formatted logs for better monitoring
- **Graceful shutdown**: Proper signal handling for clean server shutdown

## Installation

### Prerequisites

- Go 1.24.5 or later
- Git

### Build from source

1. Clone the repository:
```bash
git clone https://github.com/jempe/sitemap_scanner.git
cd sitemap_scanner
```

2. Install dependencies:
```bash
go mod tidy
```

3. Build the application:
```bash
go build -o sitemap-scanner ./cmd/get_sitemap
```

## Usage

### Basic Usage

Start the server with default settings:
```bash
./get_sitemap
```

The server will start on port 4000 by default.

### Configuration Options

```bash
./get_sitemap [options]
```

**Available flags:**
- `-port`: Server port (default: 4000)
- `-username`: Username for basic authentication (optional)
- `-password`: Password for basic authentication (optional)
- `-cache-minutes`: Cache duration in minutes (default: 1440 = 24 hours)

### Examples

**Start with custom port:**
```bash
./get_sitemap -port 8080
```

**Start with authentication:**
```bash
./get_sitemap -username admin -password secret123
```

**Start with custom cache duration (2 hours):**
```bash
./get_sitemap -cache-minutes 120
```

**Full configuration:**
```bash
./get_sitemap -port 8080 -username admin -password secret123 -cache-minutes 360
```

## API Reference

### Endpoint

`POST /get-sitemap`

### Request Format

```json
{
  "url": "https://example.com/sitemap.xml",
  "refresh_cache": false
}
```

**Parameters:**
- `url` (string, required): The URL of the sitemap to fetch
- `refresh_cache` (boolean, optional): Set to `true` to force refresh cached data

### Response Format

**Success (200 OK):**
```json
{
  "sitemap": {
    // Parsed sitemap data structure
  }
}
```

**Error (4xx/5xx):**
```json
{
  "error": "Error description"
}
```

### Example Usage

**Basic request:**
```bash
curl -X POST http://localhost:4000/get-sitemap \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/sitemap.xml"}'
```

**With authentication:**
```bash
curl -X POST http://localhost:4000/get-sitemap \
  -H "Content-Type: application/json" \
  -u admin:secret123 \
  -d '{"url": "https://example.com/sitemap.xml"}'
```

**Force cache refresh:**
```bash
curl -X POST http://localhost:4000/get-sitemap \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/sitemap.xml", "refresh_cache": true}'
```

## Caching Behavior

- **Cache Key**: The sitemap URL
- **Default Duration**: 24 hours (1440 minutes)
- **Cleanup Interval**: Every 1/5 of cache duration (default: ~5 hours)
- **Cache Refresh**: Use `refresh_cache: true` to force fetch fresh data
- **Memory Storage**: Cache is stored in memory and will be lost on server restart

### Cache Logging

The application logs cache behavior:
- **Cache Hit**: When data is served from cache
- **Cache Miss**: When fresh data needs to be fetched
- **Cache Refresh**: When cache is manually refreshed

## Development

### Project Structure

```
sitemap_scanner/
├── cmd/
│   └── get_sitemap/
│       └── main.go          # Main application entry point
├── internal/
│   └── jsonlog/
│       └── jsonlog.go       # JSON logging utilities
├── sitemap_scanner/
│   └── scanner.go           # Core sitemap scanning logic
├── go.mod                   # Go module definition
├── go.sum                   # Go module checksums
└── README.md               # This file
```

### Running in Development

```bash
go run ./cmd/get_sitemap -port 4000
```

### Testing

Test the API endpoint:
```bash
# Start the server
go run ./cmd/get_sitemap &

# Test the endpoint
curl -X POST http://localhost:4000/get-sitemap \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/sitemap.xml"}'
```

## Dependencies

- **github.com/patrickmn/go-cache**: In-memory caching
- **Standard Go libraries**: net/http, encoding/json, etc.

## License

This project is licensed under the Apache License 2.0

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## Support

For issues and questions, please open an issue on the GitHub repository.
