# cfgx 🧩

**Type-safe config generation for Go**

Stop writing config structs by hand. Define your config in TOML, generate Go code.

---

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
cfgx -in config.toml -out config/config.go
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

---

## 🤔 Why?

**The problem:** In Go apps, we define config multiple times:

1. Values in `config.toml`
2. Struct types to unmarshal into
3. Loading logic with error handling
4. Validation

**The solution:** Define config once in TOML. Generate everything else.

**Compared to viper/koanf:**

- ✅ No runtime parsing overhead
- ✅ No reflection
- ✅ Compile-time type safety
- ✅ Self-contained binaries (no config files to deploy)
- ✅ Simple - just generated Go code

**Trade-off:** Config is baked at build time. For environment-specific config, generate separate packages per environment or use build tags.

---

## 📦 Install

```bash
go install github.com/gomantics/cfgx/cmd/cfgx@latest
```

---

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
cfgx -in config.toml -out internal/config/config.go
```

Or use `go:generate`:

```go
//go:generate cfgx -in config.toml -out internal/config/config.go
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

---

## ⚙️ CLI

```bash
cfgx -in <file> -out <file> [options]
```

**Options:**

- `-in` — Input TOML file (default: `config.toml`)
- `-out` — Output Go file (required)
- `-pkg` — Package name (default: `config`)

**Examples:**

```bash
# Basic
cfgx -in config.toml -out config/config.go

# Custom package
cfgx -in app.toml -out pkg/appcfg/config.go -pkg appcfg

# Multiple configs
cfgx -in server.toml -out config/server.go
cfgx -in worker.toml -out config/worker.go
```

---

## ❓ FAQ

**Q: What about environment-specific config (dev/staging/prod)?**

Generate separate packages per environment:

```bash
cfgx -in config.dev.toml -out config/dev/config.go -pkg dev
cfgx -in config.prod.toml -out config/prod/config.go -pkg prod
```

Use build tags:

```go
// +build dev

package config
import devconfig "yourapp/config/dev"
var Server = devconfig.Server
```

**Q: What about secrets and environment variables?**

Mix generated config with runtime values:

```go
import (
    "os"
    "yourapp/config"
)

func main() {
    // Use generated config
    addr := config.Server.Addr

    // Load secrets at runtime
    apiKey := os.Getenv("API_KEY")
    dbPassword := os.Getenv("DB_PASSWORD")

    // Combine them
    dsn := fmt.Sprintf("%s?password=%s", config.Database.DSN, dbPassword)
}
```

**Q: Do I commit the generated code?**

Yes. Like sqlc and protoc, generated code is part of your source tree.

**Q: Why TOML only?**

TOML is better for config: comments, clear types, human-friendly, no indentation issues.

**Q: What types are supported?**

- Primitives: `string`, `int`, `float64`, `bool`
- Time types: `time.Duration`, `time.Time`
- Arrays: `[]string`, `[]int`, etc.
- Nested tables (structs)
- Arrays of tables

---

## ✨ Features

- **Zero runtime overhead** — Config is parsed at generation time and baked into Go code
- **Simple** — Just structs and variables, no magic
- **TOML 1.0** — Full spec support via BurntSushi/toml
- **Nested structures** — Tables become nested structs
- **Arrays** — Support for arrays of primitives and tables
- **Multiple types** — string, int, float64, bool, time.Duration, time.Time
- **Validation** — Optional validation from TOML comments (`@required`, `@enum`, `@range`)
- **No dependencies** — Generated code has zero runtime dependencies

---

## 💡 Inspiration

- [sqlc](https://sqlc.dev) — Type-safe SQL
- [protoc](https://protobuf.dev) — Schema-first development

---

## 📄 License

MIT
