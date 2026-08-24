package main

import "time"

func main() {
	config := loadConfig()
	server := newEnterpriseServer(":"+config.Port, newServer(newSampleStore()))
	timeout := time.Duration(config.ShutdownTimeoutSeconds) * time.Second
	if err := serveHTTP(server, timeout); err != nil {
		panic(err)
	}
}
