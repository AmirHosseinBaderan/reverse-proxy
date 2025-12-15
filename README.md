# Reverse Proxy

A high-performance, configurable reverse proxy server with advanced routing capabilities.

## Features

- **Host-based Virtual Hosting**: Route requests to different backends based on host headers
- **Path-based Routing**: Route different URL paths to different upstream servers
- **Load Balancing**: Round-robin and random load balancing algorithms
- **TLS Support**: HTTPS termination with automatic HTTP->HTTPS redirect
- **Configuration**: YAML-based configuration with hot reloading support
- **Performance**: Optimized connection pooling and timeout management
- **Observability**: Detailed request/response logging

## Project Structure

```
.
├── cmd/
│   └── app/              # Main application entry point
├── internal/
│   ├── application/      # Core application logic
│   │   ├── config/       # Configuration loading and parsing
│   │   ├── host/         # Host-based routing
│   │   └── site/         # Site handling and proxy logic
│   └── models/           # Data models and structures
│       └── global/       # Shared configuration models
├── sample/               # Example configurations and certificates
│   ├── config/           # Sample configuration files
│   └── certs/            # Sample TLS certificates
├── go.mod, go.sum        # Go module dependencies
└── README.md             # Project documentation
```

## Getting Started

### Prerequisites

- Go 1.20+
- Docker (optional, for containerized deployment)

### Installation

```bash
# Clone the repository
git clone https://github.com/amirhosseinbaderan/reverse-proxy.git
cd reverse-proxy

# Build the application
go build ./cmd/app/

# Run the application
./app --config ./sample/config/
```

### Configuration

The application uses YAML configuration files. See `sample/config/` for examples.

#### Basic Configuration (`settings.yml`)

```yaml
server:
  listen: ":80"              # HTTP listen address
  timeouts:
    read: 30s               # Read timeout
    write: 30s              # Write timeout
    idle: 60s               # Idle timeout
  limits:
    max_header_bytes: 1048576 # Maximum header size
  tls:
    listen: ":443"            # HTTPS listen address
    cert_file: "./certs/fullchain.pem"  # TLS certificate
    key_file: "./certs/privkey.pem"     # TLS private key
    redirect_http: true      # Redirect HTTP to HTTPS
```

#### Site Configuration (`example.com.yml`)

```yaml
domain: example.com

proxy:
  # Single upstream configuration
  upstream: http://localhost:3000
  
  # OR multiple upstreams with load balancing
  # upstreams:
  #   - http://server1:3000
  #   - http://server2:3000
  #   - http://server3:3000
  # load_balance:
  #   algorithm: round-robin  # or "random"
  
  # Path-based routing (optional)
  paths:
    - path: /api/
      upstream: http://api-backend:3000
      headers:
        X-API-Key: secret-key
    - path: /static/
      upstream: http://static-server:8080
    - path: /
      upstream: http://frontend:3000

  # Custom headers (applied to all requests)
  headers:
    X-Forwarded-For: $remote_addr
    X-Forwarded-Proto: $scheme

timeouts:
  read: 10s
  write: 10s
```

## Deployment

### Docker

```bash
# Build Docker image
docker build -t reverse-proxy .

# Run container
docker run -p 80:80 -p 443:443 \
  -v ./sample/config:/app/config \
  -v ./sample/certs:/app/certs \
  reverse-proxy
```

### Systemd

Create a systemd service file at `/etc/systemd/system/reverse-proxy.service`:

```ini
[Unit]
Description=Reverse Proxy Server
After=network.target

[Service]
User=reverse-proxy
Group=reverse-proxy
WorkingDirectory=/opt/reverse-proxy
ExecStart=/opt/reverse-proxy/app --config /opt/reverse-proxy/config/
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## Architecture

### Request Flow

```
Client Request
    ↓
HTTP/HTTPS Server
    ↓
Host-based Router
    ↓
Site Handler (per domain)
    ↓
Path-based Router (optional)
    ↓
Load Balancer (optional)
    ↓
Upstream Proxy
    ↓
Backend Server
```

### Key Components

- **Host Router**: Routes requests to the appropriate site handler based on the Host header
- **Site Handler**: Manages proxy configuration for a specific domain
- **Path Router**: Routes requests to different backends based on URL paths
- **Load Balancer**: Distributes traffic across multiple upstream servers
- **Reverse Proxy**: Forwards requests to backend servers and returns responses to clients

## Configuration Examples

See the `sample/config/` directory for comprehensive configuration examples:

- `settings.yml`: Server-wide settings
- `test.local.yml`: Basic single upstream configuration
- `test-paths.local.yml`: Path-based routing example
- `test-paths-lb.local.yml`: Load-balanced path routing example

## Development

### Running Tests

```bash
go test ./... -v
```

### Building

```bash
go build ./cmd/app/
```

### Code Structure

The codebase follows Go best practices:

- **Clean Architecture**: Separation of concerns with clear layer boundaries
- **Dependency Injection**: Configurable components with explicit dependencies
- **Error Handling**: Comprehensive error handling with context
- **Testing**: Unit tests for core functionality
- **Documentation**: Package-level documentation and inline comments

## Contributing

Contributions are welcome! Please open issues and pull requests on GitHub.

## Support

For questions or issues, please open a GitHub issue.