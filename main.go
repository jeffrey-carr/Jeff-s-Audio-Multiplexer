package main

import (
	"fmt"
	"mediacenter/client"
	"mediacenter/server"
	"os"

	"gopkg.in/yaml.v3"
)

func main() {
	configFile, err := os.ReadFile("config.yaml")
	if err != nil {
		panic(err)
	}

	var config Config
	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		panic(fmt.Errorf("Error reading config file: %s", err.Error()))
	}

	role := os.Getenv("MC_ROLE")

	var shutdown func() error
	switch role {
	case "test":
		RunPlayground()
		os.Exit(0)
	case "server":
		mediaServer := server.NewMediaServer(config.ServerPort, config.DiscoveryPort)
		shutdown, err = mediaServer.Start()
	default:
		role = "client"
		mediaClient := client.NewMediaClient(config.DiscoveryPort)
		shutdown, err = mediaClient.Start()
	}
	if err != nil {
		panic(err)
	}
	defer shutdown()

	fmt.Printf("%s running, press Enter to stop\n", role)
	fmt.Scanln()
}
