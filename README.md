# reverse-proxy

A lightweight reverse proxy server built in Go.

## Features

- HTTP and HTTPS support
- Customizable timeouts and limits
- TLS termination with automatic HTTP to HTTPS redirection
- Easy configuration via YAML files

## Usage

### Prerequisites

- Go 1.20 or higher
- A valid TLS certificate and key (for HTTPS)

### Installation

1. Clone the repository:

```bash
git clone https://github.com/amirhosseinbaderan/reverse-proxy.git
cd reverse-proxy
```

2. Build the project:

```bash
go build -o reverse-proxy ./cmd/app
```

### Configuration

The reverse proxy uses YAML files for configuration. You can find sample configuration files in the `sample/config` directory:

- `settings.yml`: Main server configuration (ports, timeouts, TLS settings)
- `test.local.yml`: Domain-specific proxy configuration

#### Example Configuration

**settings.yml**
```yaml
server:
  listen: ":80"

  timeouts:
    read: 10s
    write: 10s
    idle: 60s

  limits:
    max_header_bytes: 1048576

  tls:
    listen: ":443"
    cert_file: "./certs/fullchain.pem"
    key_file: "./certs/privkey.pem"
    redirect_http: true
```

**test.local.yml**
```yaml
domain: test.local

proxy:
  upstream: http://localhost:5258

timeouts:
  read: 10s
  write: 10s
```

### Running the Proxy

1. Place your TLS certificate and key in the `certs` directory (or update the paths in `settings.yml`).

2. Start the reverse proxy:

```bash
./reverse-proxy
```

This will start the proxy server with:
- HTTP listening on port 80
- HTTPS listening on port 443
- Automatic redirection from HTTP to HTTPS
- Proxying requests for `test.local` to `http://localhost:5258`

## Development

### Testing

Run the tests:

```bash
go test ./...
```

### Building

To build for production:

```bash
GOOS=linux GOARCH=amd64 go build -o reverse-proxy ./cmd/app
```

## License

MIT