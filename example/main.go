package main

import (
	"fmt"
	"time"

	"github.com/gomantics/cfgx/example/config"
)

func main() {
	fmt.Println("=== Application Configuration ===")
	fmt.Printf("Server Address: %s\n", config.Server.Addr)
	fmt.Printf("Read Timeout: %d seconds\n", config.Server.ReadTimeout)
	fmt.Printf("Write Timeout: %d seconds\n", config.Server.WriteTimeout)
	fmt.Println()

	fmt.Printf("Database DSN: %s\n", config.Database.Dsn)
	fmt.Printf("Max Open Connections: %d\n", config.Database.MaxOpenConns)
	fmt.Printf("Max Idle Connections: %d\n", config.Database.MaxIdleConns)
	fmt.Println()

	fmt.Printf("Redis Address: %s\n", config.Redis.Addr)
	fmt.Printf("Redis DB: %d\n", config.Redis.Db)
	fmt.Println()

	fmt.Printf("Log Level: %s\n", config.Logging.Level)
	fmt.Printf("Log Format: %s\n", config.Logging.Format)
	fmt.Printf("Log Outputs: %v\n", config.Logging.Outputs)
	fmt.Println()

	for _, feature := range config.Features {
		fmt.Printf("Feature: %s\n", feature.Name)
		fmt.Printf("Enabled: %v\n", feature.Enabled)
		fmt.Printf("Priority: %d\n", feature.Priority)
		fmt.Println()
	}

	fmt.Println("=== Server would start with: ===")
	fmt.Printf("Address: %s\n", config.Server.Addr)
	fmt.Printf("Read Timeout: %v\n", time.Duration(config.Server.ReadTimeout)*time.Second)
	fmt.Printf("Write Timeout: %v\n", time.Duration(config.Server.WriteTimeout)*time.Second)
}
