# cfgx 🧩

**Type-safe config generation for Go**

Stop writing config structs by hand. Define your config in TOML, generate Go code.

## 💡 What is this?

`cfgx` is a code generator that turns your TOML config into Go code.

The TOML is parsed at generation time, and values are baked directly into your Go code as pre-initialized variables. No runtime parsing, no reflection, no dependencies.

```toml
# config.toml
[server]
addr = ":8080"
timeout = 30
```

```bash
cfgx generate --in config.toml --out config/config.go
```

```go
// Generated: config/config.go
package config

type ServerConfig struct {
    Addr    string
    Timeout int
}

var Server = ServerConfig{
    Addr:    ":8080",
    Timeout: 30,
}
```

Use it:

```go
import "yourapp/config"

func main() {
    fmt.Println(config.Server.Addr)  // ":8080"
}
```

## 🤔 Why?

**The problem:** In Go apps, we define config multiple times:

1. Values in `config.toml`
2. Struct types to unmarshal into
3. Loading logic with error handling

**The solution:** Define config once in TOML. Generate everything else.

**Compared to viper/koanf:**

- ✅ No runtime parsing overhead
- ✅ No reflection
- ✅ Compile-time type safety
- ✅ Self-contained binaries (no config files to deploy)
- ✅ Simple - just generated Go code

**Trade-off:** Config is baked at build time. For environment-specific config, generate from different TOML files per environment during build.

## 📦 Install

```bash
go install github.com/gomantics/cfgx/cmd/cfgx@latest
```

## 🚀 Usage

### 1. Create config.toml

```toml
[server]
addr = ":8080"
read_timeout = 15

[database]
dsn = "postgres://localhost/myapp"
max_conns = 25

[app]
name = "myservice"
debug = true
```

### 2. Generate Go code

```bash
cfgx generate --in config.toml --out internal/config/config.go
```

Or use `go:generate`:

```go
//go:generate cfgx generate --in config.toml --out internal/config/config.go
```

### 3. Use it

```go
package main

import "yourapp/internal/config"

func main() {
    server := &http.Server{
        Addr:        config.Server.Addr,
        ReadTimeout: time.Duration(config.Server.ReadTimeout) * time.Second,
    }
    server.ListenAndServe()
}
```

## ⚙️ CLI

```bash
cfgx generate --in <file> --out <file> [options]
```

**Commands:**

- `generate` — Generate type-safe Go code from TOML config
- `version` — Print version information

**Options:**

- `--in, -i` — Input TOML file (default: `config.toml`)
- `--out, -o` — Output Go file (required)
- `--pkg, -p` — Package name (default: inferred from output path or `config`)
- `--no-env` — Disable environment variable overrides

**Examples:**

```bash
# Basic
cfgx generate --in config/config.toml --out config/config.go

# Custom package
cfgx generate --in app.toml --out pkg/appcfg/config.go --pkg appcfg

# Multiple configs
cfgx generate --in config/server.toml --out config/server.go
cfgx generate --in config/worker.toml --out config/worker.go

# Check version
cfgx version
```

## ❓ FAQ

**Q: What about environment-specific config (dev/staging/prod)?**

Create separate config files per environment and generate from the appropriate one during deployment:

```bash
# Development
cfgx generate --in config/config.dev.toml --out config/config.go

# Production
cfgx generate --in config/config.prod.toml --out config/config.go
```

In your CI/CD pipeline or Dockerfile:

```dockerfile
# Dockerfile
FROM golang:1.25.1 as builder
COPY config.${ENV}.toml config.toml
RUN cfgx generate --in config.toml --out config/config.go
RUN go build -o app
```

Or build different binaries:

```bash
# CI pipeline
cfgx generate --in config/config.prod.toml --out config/config.go && go build -o app-prod
cfgx generate --in config/config.dev.toml --out config/config.go && go build -o app-dev
```

**Q: What about secrets and environment variables?**

`cfgx` supports environment variable overrides out of the box. Any config value can be overridden at generate time (not runtime) using environment variables with the pattern `CONFIG_<SECTION>_<KEY>`.

```toml
# config.toml
[database]
dsn = "postgres://localhost/myapp"
max_conns = 25

[server]
addr = ":8080"
```

```bash
# Set environment variables before generation
export CONFIG_DATABASE_DSN="postgres://prod-db:5432/myapp?sslmode=require"
export CONFIG_SERVER_ADDR=":3000"

# Generate config with overrides baked in
cfgx generate --in config.toml --out config/config.go
```

The generated code will have the overridden values:

```go
// Generated config/config.go
var Database = DatabaseConfig{
    Dsn: "postgres://prod-db:5432/myapp?sslmode=require",  // Overridden value
    MaxConns: 25,
}

var Server = ServerConfig{
    Addr: ":3000",  // Overridden value
}
```

Use it in your application:

```go
import "yourapp/config"

func main() {
    db := sql.Open("postgres", config.Database.Dsn)
    server := &http.Server{Addr: config.Server.Addr}
}
```

This keeps your config as a single source of truth with values baked at build time.

**Coming soon:** Support for pulling secrets from Google Secret Manager and AWS Secrets Manager during build time.

**Q: Do I commit the generated code?**

Yes. Like sqlc and protoc, generated code is part of your source tree.

However, **do not commit production config files that contain secrets** (e.g., `config.prod.toml` with API keys or passwords). Instead:

1. Keep production TOML files out of source control (add to `.gitignore`)
2. Generate prod config during deployment from secrets stored in your CI/CD system or secret manager
3. For local dev, use non-sensitive config files or placeholder values

For production secrets, combine config with environment variables as shown in the "secrets and environment variables" FAQ above.

**Q: Why TOML only?**

TOML is better for config: comments, clear types, human-friendly, no indentation issues.

**Q: What types are supported?**

- Primitives: `string`, `int`, `float64`, `bool`
- Arrays: `[]string`, `[]int`, etc.
- Nested tables (structs)
- Arrays of tables

For time-related config (timeouts, durations), use integers representing seconds/milliseconds and convert them in your application code (e.g., `time.Duration(config.Server.ReadTimeout) * time.Second`)

## ✨ Features

- **Zero runtime overhead** — Config is parsed at generation time and baked into Go code
- **Simple** — Just structs and variables, no magic
- **TOML 1.0** — Full spec support via BurntSushi/toml
- **Nested structures** — Tables become nested structs
- **Arrays** — Support for arrays of primitives and tables
- **Multiple types** — string, int, float64, bool
- **Environment variable overrides** — Override any config value at generation time
- **No dependencies** — Generated code has zero runtime dependencies

## 💡 Inspiration

- [sqlc](https://sqlc.dev) — Type-safe SQL
- [protoc](https://protobuf.dev) — Schema-first development

## 📄 License

MIT
