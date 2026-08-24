package main

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port                   string
	RequireOperator        bool
	ShutdownTimeoutSeconds int
}

func loadConfig() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
		port = "8080"
	}
	requireOperator := true
	if value := os.Getenv("REQUIRE_OPERATOR"); value == "0" || strings.EqualFold(value, "false") {
		requireOperator = false
	}
	return Config{Port: port, RequireOperator: requireOperator, ShutdownTimeoutSeconds: 10}
}
