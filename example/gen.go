package main

//go:generate go run ../cmd/cfgx/main.go generate --in config/config.toml --out config/config.go --pkg config
//go:generate go run ../cmd/cfgx/main.go generate --in config/config.toml --out getter_config/config.go --pkg getter_config --mode getter
