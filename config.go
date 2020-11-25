package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"

	"gopkg.in/yaml.v2"
)

// Contract struct
type Contract struct {
	Address    string   `yaml:"address"`
	Signatures []string `yaml:"signatures"`
}

// Configuration retains the config struct for the EthLogSpy instance
var Configuration *Config

// GetConfig gets a filesystem path as an argument and returns, if successful,
// a Config object that can be used throughout the application
func GetConfig(configPath string, vars ...string) {
	// Retrieve ENV environment variable
	env := os.Getenv("ENV")
	// If vars is provided, the first one is the override for the env
	if len(vars) > 0 {
		env = vars[0]
	}
	// Check if no env is provided
	if env == "" {
		// If it is not provided, set it as development
		env = "development"
	}
	// Check if env is either 'test', 'production' or 'development'
	if env != "test" && env != "production" && env != "development" {
		// Raise an error if it does not match the criteria
		panic(errors.New("environment must be either 'test', 'production' or 'development'"))
	}

	file, err := os.Open(fmt.Sprintf("%s.yml", path.Join(configPath, env)))
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
