# cfgx Roadmap

This document outlines planned CLI commands and features for cfgx, organized by priority. We aim to keep cfgx minimal and focused on individual developers and small teams.

## Status: v0.x.x

The current focus is on stability and core functionality refinement. New commands will be added incrementally based on user feedback.

---

## ✅ Implemented

### Core Commands

- **`generate`** - Generate type-safe Go code from TOML config
- **`version`** - Display version information
- **`watch`** - Auto-regenerate on TOML file changes (✨ NEW)

---

## 🚀 Immediate Priority

### `diff`

Compare two TOML files and highlight configuration differences.

**Purpose:** Help developers understand what changes between environments (dev vs prod, base vs override).

**Usage:**

```bash
# Compare two config files
cfgx diff config.dev.toml config.prod.toml

# Show only changed keys
cfgx diff config.dev.toml config.prod.toml --keys-only

# Output as JSON for scripting
cfgx diff base.toml override.toml --format json
```

**Output Example:**

```
Differences between config.dev.toml and config.prod.toml:

  server.addr
    - ":8080"     (config.dev.toml)
    + ":443"      (config.prod.toml)

  database.max_conns
    - 10          (config.dev.toml)
    + 100         (config.prod.toml)

  + server.tls_enabled = true     (only in config.prod.toml)
  - server.debug = true           (only in config.dev.toml)
```

**Priority:** High - Common use case for multi-environment deployments

---

### `merge`

Combine multiple TOML files with override semantics (last wins).

**Purpose:** Enable base + environment-specific config pattern without duplication.

**Usage:**

```bash
# Merge configs and generate
cfgx merge config.base.toml config.prod.toml --out merged.toml

# Merge and generate in one step
cfgx merge config.base.toml config.dev.toml | cfgx generate --in - --out config.go

# Multiple layers
cfgx merge base.toml region.toml env.toml --out final.toml
```

**Merge Behavior:**

- Later files override earlier ones
- Arrays are replaced, not merged
- Nested tables merged recursively
- Preserves comments from last file with the key

**Priority:** High - Eliminates config duplication across environments

---

## 📦 High Value

### `init`

Bootstrap a new project with sensible defaults and examples.

**Purpose:** Quick start for new users, reduces friction.

**Usage:**

```bash
# Create example config in current directory
cfgx init

# Specify output directory
cfgx init --dir config/

# Choose a template
cfgx init --template web-server
cfgx init --template cli-tool
```

**Creates:**

```
config/
├── config.toml          # Example TOML with comments
├── config.go            # Generated output (gitignored)
└── gen.go               # go:generate directive
```

**Priority:** Medium - Improves onboarding experience

---

### `validate`

Validate TOML syntax and check for common configuration issues.

**Purpose:** Catch errors before generation, provide helpful feedback.

**Usage:**

```bash
# Validate single file
cfgx validate config.toml

# Validate with generation mode checks
cfgx validate config.toml --mode getter

# Check all TOML files in directory
cfgx validate --dir config/

# CI mode: exit code only
cfgx validate config.toml --quiet
```

**Checks:**

- TOML syntax errors
- File references exist and are readable
- Duration strings are valid
- Array type consistency
- Reasonable file sizes for `file:` references
- Warn about common mistakes (e.g., forgetting quotes on duration strings)

**Priority:** Medium - Improves error messages and catches issues early

---

### `fmt`

Format TOML files with consistent style.

**Purpose:** Like `go fmt` but for TOML - standardize formatting across projects.

**Usage:**

```bash
# Format in place
cfgx fmt config.toml

# Check if formatted
cfgx fmt --check config.toml

# Format all TOML files
cfgx fmt config/*.toml

# Preview changes
cfgx fmt --diff config.toml
```

**Formatting Rules:**

- Consistent indentation (2 spaces)
- Alphabetize keys within sections
- Blank line between sections
- Comments stay with their keys
- Arrays formatted consistently

**Priority:** Medium - Nice-to-have for team consistency

---

## 🎨 Nice-to-Have

### `preview`

Show generated code without writing to disk (dry-run).

**Usage:**

```bash
# Print to stdout
cfgx preview --in config.toml

# Preview with syntax highlighting (if terminal supports it)
cfgx preview --in config.toml --color

# Preview specific section
cfgx preview --in config.toml --section server
```

**Priority:** Low - Can achieve with `--out /dev/stdout` or similar

---

### `inspect`

Display parsed TOML structure in human-readable format.

**Purpose:** Debug what cfgx sees, understand nested structures.

**Usage:**

```bash
# Show structure
cfgx inspect config.toml

# As JSON
cfgx inspect config.toml --format json

# Show types that will be generated
cfgx inspect config.toml --show-types
```

**Output Example:**

```
server (table)
  ├─ addr: ":8080" (string)
  ├─ timeout: "30s" (duration)
  └─ debug: true (bool)

database (table)
  ├─ dsn: "..." (string)
  └─ pool (table)
      ├─ max_size: 10 (int64)
      └─ timeout: "5s" (duration)
```

**Priority:** Low - Useful for debugging, not critical

---

### `comment-sync`

Extract TOML comments and generate Go documentation.

**Purpose:** Keep config documentation in one place (TOML), flow to generated code.

**Usage:**

```bash
# Generate with inline comments as doc strings
cfgx generate --in config.toml --out config.go --with-comments

# Or as separate command
cfgx comment-sync --in config.toml --out config.go
```

**Example:**

```toml
[server]
# The address to bind the HTTP server to.
# Supports host:port format, e.g., "localhost:8080" or ":8080" for all interfaces.
addr = ":8080"

# Request timeout duration. Must be at least 1 second.
timeout = "30s"
```

Generates:

```go
type ServerConfig struct {
    // Addr is the address to bind the HTTP server to.
    // Supports host:port format, e.g., "localhost:8080" or ":8080" for all interfaces.
    Addr string

    // Timeout is the request timeout duration. Must be at least 1 second.
    Timeout time.Duration
}
```

**Priority:** Low - Nice for documentation, but adds complexity

---

## 🚫 Out of Scope

These features go against cfgx's minimal philosophy or are better suited as separate projects:

- **Secret manager integration** - Too complex, should use external tools (e.g., inject at build time)
- **GUI/web interface** - CLI-first tool, GUIs add maintenance burden
- **LSP/IDE plugins** - Separate project if needed
- **Multi-format support** (YAML, JSON, etc.) - TOML is purposefully chosen for config
- **Config encryption** - Use external secrets management
- **Remote config fetching** - Violates build-time philosophy
- **Dynamic reloading** - Runtime concern, not generation tool's job

---

## Contributing

Have an idea for a command? Start a discussion in [GitHub Discussions](https://github.com/gomantics/cfgx/discussions/categories/ideas).

Keep it minimal! Commands should:

- Solve a real, common problem
- Align with cfgx's philosophy (build-time, type-safe, simple)
- Not duplicate functionality available elsewhere
- Be implementable in <200 lines of code
