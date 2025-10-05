# Example Usage

This example demonstrates how to use cfgx in a real application.

## Generate Config

```bash
# Generate the config code
go generate
```

## Run the Example

```bash
go run main.go
```

## With Environment Variable Overrides

```bash
# Override config at generation time
export CONFIG_SERVER_ADDR=":3000"
export CONFIG_DATABASE_MAX_OPEN_CONNS="100"
go generate
go run main.go
```
