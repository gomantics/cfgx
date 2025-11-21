# cfgx

[![Go Reference](https://pkg.go.dev/badge/github.com/gomantics/cfgx.svg)](https://pkg.go.dev/github.com/gomantics/cfgx)
[![CI](https://github.com/gomantics/cfgx/actions/workflows/ci.yml/badge.svg)](https://github.com/gomantics/cfgx/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/gomantics/cfgx)](https://goreportcard.com/report/github.com/gomantics/cfgx)

Type-safe configuration code generation for Go. Define your config in TOML, generate strongly-typed Go code with zero runtime dependencies.

## Installation

```bash
go install github.com/gomantics/cfgx/cmd/cfgx@latest
```

## Documentation

For complete documentation, CLI reference, generation modes, and multi-environment setup, visit:

**https://gomantics.dev/cfgx**

## Quick Example

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
// Generated code - ready to use
var Server = ServerConfig{
    Addr:    ":8080",
    Timeout: 30 * time.Second,
}
```

## Key Features

- Zero runtime overhead - config baked at build time
- Compile-time type safety
- Two generation modes: static and getter (with env var overrides)
- File embedding support
- Environment variable overrides
- Multi-environment configuration

## Status

**v0.x.x** — Production-ready for build-time configuration generation. API may introduce breaking changes between minor versions.

## License

[MIT](LICENSE)
