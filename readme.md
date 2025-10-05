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
  -i, --in string      Input TOML file (default "config.toml")
  -o, --out string     Output Go file (required)
  -p, --pkg string     Package name (inferred from output path)
      --no-env         Disable environment variable overrides
```

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

## Supported Types

- **Primitives:** `string`, `int64`, `float64`, `bool`
- **Duration:** `time.Duration` (auto-detected from Go duration strings: `"30s"`, `"5m"`, `"2h30m"`)
- **Arrays:** Arrays of any supported type
- **Nested tables:** Becomes nested structs
- **Array of tables:** `[]StructType`

## Multi-Environment Config

### Approach 1: Separate files per environment

```bash
# Development
cfgx generate --in config.dev.toml --out config/config.go

# Production
cfgx generate --in config.prod.toml --out config/config.go
```

### Approach 2: CI/CD with environment variables

```dockerfile
FROM golang:1.25.1 as builder
WORKDIR /app
COPY . .
RUN go install github.com/gomantics/cfgx/cmd/cfgx@latest
RUN cfgx generate --in config.toml --out config/config.go
RUN go build -o app
```

Set environment variables in your CI system:

```bash
CONFIG_DATABASE_DSN="postgres://prod.example.com/db"
CONFIG_SERVER_ADDR=":443"
```

### Approach 3: Build matrix

```bash
cfgx generate --in config.prod.toml --out config/config.go && go build -o app-prod
cfgx generate --in config.dev.toml --out config/config.go && go build -o app-dev
```

## FAQ

### Should I commit generated code?

**Yes.** Like `sqlc` and `protoc`, commit the generated code. It's part of your source tree and should be versioned.

**However:** Don't commit TOML files with production secrets. Keep those in your secrets manager and inject via environment variables during build.

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
