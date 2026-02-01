package main

// Config defines the config for the app
type Config struct {
	ServerHost    string `yaml:"server_host"`
	ServerPort    int    `yaml:"server_port"`
	DiscoveryPort int    `yaml:"discovery_port"`
}
