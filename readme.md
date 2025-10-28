# cfgx

[![Go Reference](https://pkg.go.dev/badge/github.com/gomantics/cfgx.svg)](https://pkg.go.dev/github.com/gomantics/cfgx)
[![CI](https://github.com/gomantics/cfgx/actions/workflows/ci.yml/badge.svg)](https://github.com/gomantics/cfgx/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/gomantics/cfgx)](https://goreportcard.com/report/github.com/gomantics/cfgx)

Type-safe configuration code generation for Go.

Define your config in TOML, generate strongly-typed Go code with zero runtime dependencies.

```toml
# config.toml
[server]
addr = ":8080"
timeout = "30s"
```

```bash
$ cfgx generate --in config.toml --out config/config.go
```

```go
// Generated
var Server = ServerConfig{
    Addr:    ":8080",
    Timeout: 30 * time.Second,
}
```

## Status

**v0.x.x** — The API may introduce breaking changes between minor versions. That said, `cfgx` is a small, focused tool that's production-ready for baking configuration at build time. We encourage you to use it in production systems.

## Why

Traditional config loading in Go involves:

1. Writing structs by hand
2. Loading files at runtime
3. Unmarshaling with error handling
4. Runtime parsing overhead

`cfgx` generates everything at build time. No runtime parsing, no reflection, no config files to deploy. Just plain Go code with your values baked in.

**vs. viper/koanf:**

- Zero runtime overhead
- Compile-time type safety
- Self-contained binaries
- No reflection

**Trade-off:** Config is baked at build time. For multi-environment setups, generate from different TOML files per environment during your build process.

## Install

```bash
go install github.com/gomantics/cfgx/cmd/cfgx@latest
```

## Usage

### Basic

```bash
cfgx generate --in config.toml --out config/config.go
```

### With go:generate

```go
//go:generate cfgx generate --in config.toml --out config/config.go
```

Then run:

```bash
go generate ./...
```

### Multiple configs

```bash
cfgx generate --in server.toml --out config/server.go
cfgx generate --in worker.toml --out config/worker.go
```

## CLI Reference

```
cfgx generate [flags]

Flags:
  -i, --in string          Input TOML file (default "config.toml")
  -o, --out string         Output Go file (required)
  -p, --pkg string         Package name (inferred from output path)
      --mode string        Generation mode: 'static' or 'getter' (default "static")
      --no-env             Disable environment variable overrides (static mode only)
      --max-file-size      Maximum file size for file: references (default "1MB")
                           Supports: KB, MB, GB (e.g., "5MB", "1GB", "512KB")
```

## Modes

`cfgx` supports two generation modes, chosen via the `--mode` flag:

### Static Mode (default)

Values are baked into the binary at build time. Best for:

- Internal tools
- Single-environment deployments
- Maximum performance (zero runtime overhead)

```bash
cfgx generate --in config.toml --out config/config.go --mode static
# or just omit --mode (static is default)
cfgx generate --in config.toml --out config/config.go
```

### Getter Mode

Generates getter methods that check environment variables at runtime, falling back to defaults from TOML. Best for:

- Open source projects
- Docker/container deployments
- Multi-environment apps
- 12-factor apps

```bash
cfgx generate --in config.toml --out config/config.go --mode getter
```

**Input:**

```toml
[server]
addr = ":8080"
timeout = "30s"
debug = true
```

**Generated (getter mode):**

```go
type ServerConfig struct{}

func (ServerConfig) Addr() string {
    if v := os.Getenv("CONFIG_SERVER_ADDR"); v != "" {
        return v
    }
    return ":8080"
}

func (ServerConfig) Timeout() time.Duration {
    if v := os.Getenv("CONFIG_SERVER_TIMEOUT"); v != "" {
        if d, err := time.ParseDuration(v); err == nil {
            return d
        }
    }
    return 30 * time.Second
}

func (ServerConfig) Debug() bool {
    if v := os.Getenv("CONFIG_SERVER_DEBUG"); v != "" {
        if b, err := strconv.ParseBool(v); err == nil {
            return b
        }
    }
    return true
}

var Server ServerConfig
```

**Usage:**

```go
// In your application
http.ListenAndServe(config.Server.Addr(), handler)

// Override at runtime
// $ CONFIG_SERVER_ADDR=":3000" ./myapp
```

**Environment variable format:** `CONFIG_SECTION_KEY`

- Nested: `CONFIG_DATABASE_POOL_MAX_SIZE`
- Type-safe parsing with silent fallback to defaults

**File overrides:**

File references can be overridden at runtime by passing file paths via env vars:

```bash
# Use embedded file (from build time)
./myapp

# Override with production certificate
CONFIG_SERVER_TLS_CERT=/etc/ssl/certs/prod.crt ./myapp
```

**Kubernetes example:**

```yaml
env:
  - name: CONFIG_SERVER_TLS_CERT
    value: /etc/tls/tls.crt
  - name: CONFIG_SERVER_TLS_KEY
    value: /etc/tls/tls.key
volumeMounts:
  - name: tls-secret
    mountPath: /etc/tls
    readOnly: true
volumes:
  - name: tls-secret
    secret:
      secretName: my-tls-secret
```

**Behavior:**

- If env var is set to a file path and file is readable, use that file
- If file doesn't exist or can't be read, silently fall back to embedded bytes
- Embedded bytes from build time are always available as fallback

**Limitations in getter mode:**

- Arrays cannot be overridden via env vars (always use defaults)

## Features

### Type Detection

Automatic type inference with smart duration detection:

```toml
[server]
port = 8080           # int64
host = "localhost"    # string
debug = true          # bool
timeout = "30s"       # time.Duration (auto-detected)

[database]
max_conns = 25
retry_delay = "5s"
```

Generates:

```go
type ServerConfig struct {
    Port    int64
    Host    string
    Debug   bool
    Timeout time.Duration
}

var Server = ServerConfig{
    Port:    8080,
    Host:    "localhost",
    Debug:   true,
    Timeout: 30 * time.Second,
}
```

### Nested Structures

```toml
[database.primary]
dsn = "postgres://localhost/app"

[database.replica]
dsn = "postgres://replica/app"
```

Generates nested structs automatically.

### Arrays

```toml
[app]
hosts = ["api.example.com", "web.example.com"]
ports = [8080, 8081]
intervals = ["30s", "1m", "5m"]
```

### Environment Variable Overrides

Override any value at generation time:

```bash
export CONFIG_SERVER_ADDR=":3000"
export CONFIG_DATABASE_MAX_CONNS="50"
cfgx generate --in config.toml --out config/config.go
```

The generated code will contain the overridden values. Useful for CI/CD pipelines where you inject secrets at build time.

Use `--no-env` to disable this feature.

### File Embedding

Embed file contents directly into your generated code using the `file:` prefix:

```toml
[server]
addr = ":8080"
tls_cert = "file:certs/server.crt"
tls_key = "file:certs/server.key"

[app]
logo = "file:assets/logo.png"
sql_schema = "file:migrations/schema.sql"
```

Generates:

```go
type ServerConfig struct {
    Addr    string
    TlsCert []byte  // Embedded certificate bytes
    TlsKey  []byte  // Embedded key bytes
}

var Server = ServerConfig{
    Addr: ":8080",
    TlsCert: []byte{
        0x2d, 0x2d, 0x2d, 0x2d, 0x2d, 0x42, 0x45, 0x47, 0x49, 0x4e, 0x20, 0x43,
        // ... actual cert bytes ...
    },
    TlsKey: []byte{ /* ... key bytes ... */ },
}
```

**Key features:**

- **Paths are relative** to the TOML file location
- **Files are read at generation time** - no runtime I/O
- **Self-contained binaries** - no need to distribute separate files
- **Size limits** - defaults to 1MB, configurable via `--max-file-size`
- **Any file type** - text, json, binary, certificates, images, etc.

**Example usage:**

```bash
cfgx generate --in config.toml --out config/config.go --max-file-size 5MB
```

**Use cases:**

- TLS certificates and keys
- SQL migration schemas
- Template files
- Small assets (logos, icons)
- Configuration snippets
- Test fixtures

## Supported Types

- **Primitives:** `string`, `int64`, `float64`, `bool`
- **Duration:** `time.Duration` (auto-detected from Go duration strings: `"30s"`, `"5m"`, `"2h30m"`)
- **File content:** `[]byte` (use `"file:path/to/file"` prefix)
- **Arrays:** Arrays of any supported type
- **Nested tables:** Becomes nested structs
- **Array of tables:** `[]StructType`

## Multi-Environment Config

### Approach 1: Getter Mode (Recommended for Docker/OSS)

Use `--mode getter` to generate runtime-configurable code:

```dockerfile
FROM golang:1.25.1 as builder
WORKDIR /app
COPY . .
RUN go install github.com/gomantics/cfgx/cmd/cfgx@latest
RUN cfgx generate --in config.toml --out config/config.go --mode getter
RUN go build -o app

FROM alpine
COPY --from=builder /app/app /app
CMD ["/app"]
```

Users can now configure via environment variables:

```bash
docker run -e CONFIG_SERVER_ADDR=":3000" \
           -e CONFIG_DATABASE_DSN="postgres://mydb/app" \
           -e CONFIG_SERVER_TIMEOUT="60s" \
           yourapp:latest
```

### Approach 2: Static Mode with Separate Files

For locked-down production deployments:

```bash
# Development
cfgx generate --in config.dev.toml --out config/config.go --mode static

# Production
cfgx generate --in config.prod.toml --out config/config.go --mode static
```

### Approach 3: Static Mode with Build-Time Env Vars

```dockerfile
FROM golang:1.25.1 as builder
WORKDIR /app
COPY . .
RUN go install github.com/gomantics/cfgx/cmd/cfgx@latest
# Inject secrets at build time via --no-env flag is disabled (enabled by default)
RUN cfgx generate --in config.toml --out config/config.go --mode static
RUN go build -o app
```

Set environment variables in your CI system:

```bash
CONFIG_DATABASE_DSN="postgres://prod.example.com/db"
CONFIG_SERVER_ADDR=":443"
```

### Approach 4: Build Matrix

```bash
cfgx generate --in config.prod.toml --out config/config.go && go build -o app-prod
cfgx generate --in config.dev.toml --out config/config.go && go build -o app-dev
```

## FAQ

### When should I use static vs getter mode?

**Use static mode when:**

- Building internal tools or services
- You control the deployment environment
- Performance is critical (zero runtime overhead)
- Config rarely changes between environments
- You want maximum security (no runtime overrides possible)

**Use getter mode when:**

- Building open source applications
- Distributing via Docker/containers
- Users need to configure without rebuilding
- Following 12-factor app principles
- You want sensible defaults with easy overrides

**Rule of thumb:** If you're shipping to users who can't rebuild from source, use getter mode.

### Should I commit generated code?

**Yes.** Like `sqlc` and `protoc`, commit the generated code. It's part of your source tree and should be versioned.

**However:** Don't commit TOML files with production secrets. Keep those in your secrets manager and inject via environment variables during build.

### How do I use different TLS certs in production?

**Getter mode (recommended):**

```bash
# Development: uses embedded certs from config.toml
./myapp

# Production: override via env vars pointing to file paths
CONFIG_SERVER_TLS_CERT=/etc/ssl/prod.crt \
CONFIG_SERVER_TLS_KEY=/etc/ssl/prod.key \
./myapp
```

In Kubernetes, mount secrets as files and point to them:

```yaml
env:
  - name: CONFIG_SERVER_TLS_CERT
    value: /etc/tls/tls.crt
volumeMounts:
  - name: tls-secret
    mountPath: /etc/tls
volumes:
  - name: tls-secret
    secret:
      secretName: prod-tls-secret
```

**Static mode:**

Use different TOML files for each environment and generate at build time.

### Do I need to distribute config files?

**No.** The whole point is that config is baked into your binary. No runtime file loading needed.

### Can I modify generated code?

**No.** The generated file includes a `// Code generated ... DO NOT EDIT` marker. Regenerate instead.

### Why TOML only?

TOML is designed for config: readable, has comments, unambiguous types, no indentation issues. YAML and JSON don't handle config as well.

### Coming Soon

- Secret manager integration (AWS Secrets Manager, Google Secret Manager)
- Config validation rules
- Default value annotations

## Inspiration

- [sqlc](https://sqlc.dev) — If sqlc can generate type-safe database code, why not config?
- [protoc](https://protobuf.dev) — Schema-first development works

## Contributing

Issues and PRs welcome. This is a small, focused tool — let's keep it that way.

For feature requests, start a discussion [here](https://github.com/gomantics/cfgx/discussions/categories/ideas).

## License

[MIT](LICENSE)
