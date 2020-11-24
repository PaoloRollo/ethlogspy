package main

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

var Logger *logrus.Logger

func SetupLogger() {
	Logger = &logrus.Logger{
		Out:       os.Stdout,
		Formatter: &logrus.TextFormatter{DisableColors: false, FullTimestamp: true},
		Level:     logrus.InfoLevel,
	}
}

func ValidatePath(path string) (*string, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("'%s' is not a valid directory", path)
	}
	return &path, nil
}
