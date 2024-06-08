package cmd

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/insomnius/agent/entity"
	"gopkg.in/yaml.v2"
)

var ErrDockerSocketNotExists = fmt.Errorf("docker socket is not exists")
var ErrDockerSocketCantBeReach = fmt.Errorf("docker socket cannot be reached")
var ErrConfigFileNotFound = fmt.Errorf("config file not found")
var ErrWritingDefaultConfig = fmt.Errorf("error writing default config")

func checkDockerSocketRunning() error {
	dockerSocket := "/var/run/docker.sock"

	if _, err := os.Stat(dockerSocket); os.IsNotExist(err) {
		return ErrDockerSocketNotExists
	}

	conn, err := net.Dial("unix", dockerSocket)
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()
	if err != nil {
		return ErrDockerSocketCantBeReach
	}

	return nil
}

func checkConfig(configPath string) error {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return ErrConfigFileNotFound
	} else if err != nil {
		return err
	}
	return nil
}

// createDefaultConfig creates a new config file with default values.
func createDefaultConfig(configPath string) error {
	defaultConfig := entity.CconnectorConfig{
		HostToken:    "",
		ManagerToken: "",
	}

	configData, err := yaml.Marshal(&defaultConfig)
	if err != nil {
		return errors.Join(ErrWritingDefaultConfig, err)
	}

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return errors.Join(ErrWritingDefaultConfig, err)
	}

	// Write the default config to the file
	file, err := os.Create(configPath)
	if err != nil {
		return errors.Join(ErrWritingDefaultConfig, err)
	}
	defer file.Close()

	if _, err := file.Write(configData); err != nil {
		return errors.Join(ErrWritingDefaultConfig, err)
	}

	return nil
}

// ensureConfig checks if the config file exists, and creates it if it doesn't.
func ensureConfig(configPath string) error {
	err := checkConfig(configPath)
	if err != nil && err != ErrConfigFileNotFound {
		return err
	}

	if err := createDefaultConfig(configPath); err != nil {
		return err
	}

	return nil
}
