package main

import (
	"log"
	"os"
	"path"
    "strconv"

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
	var nodeHost, corsOrigin string
	nodeHost = os.Getenv("NODE_HOST")
	if len(nodeHost) == 0 {
		nodeHost = "localhost"
	}
	corsOrigin = os.Getenv("CORS_ORIGIN")
	if len(corsOrigin) == 0 {
		corsOrigin = "*"
	}
	Configuration.Node.Host = nodeHost
	Configuration.Server.CorsOrigin = corsOrigin
	var nodePortInt, blockNumberInt int
	nodePort := os.Getenv("NODE_PORT")
	if len(nodePort) == 0 {
		nodePortInt = 8545
	} else {
		nodePortInt, err = strconv.Atoi(nodePort)
		if err != nil {
			nodePortInt = 8545
		}
	}
	Configuration.Node.Port = nodePortInt
	blockNumber := os.Getenv("BLOCK_NUMBER")
	if len(blockNumber) == 0 {
		blockNumberInt = 0
	} else {
		blockNumberInt, err = strconv.Atoi(blockNumber)
		if err != nil {
			blockNumberInt = 8545
		}
	}
	Configuration.Server.FromBlock = uint64(blockNumberInt)
}
