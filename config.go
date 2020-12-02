package main

import (
	"log"
	"os"
	"path"

	"gopkg.in/yaml.v2"
)

// Configuration retains the config struct for the EthLogSpy instance
var Configuration *Config

// GetConfig gets a filesystem path as an argument and returns, if successful,
// a Config object that can be used throughout the application
func GetConfig(configPath string, vars ...string) {
	file, err := os.Open(path.Join(configPath, "config.yml"))
	if err != nil {
		// Return an error if it's given
		log.Fatalf("error opening file: %v", err)
	}
	// Decode the yaml file into the struct
	yamlDecoder := yaml.NewDecoder(file)
	if err := yamlDecoder.Decode(&Configuration); err != nil {
		log.Fatalf("error decoding file: %v", err)
	}
}
