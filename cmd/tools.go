package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/insomnius/agent/entity"
	"gopkg.in/yaml.v2"
)

var ErrDockerSocketNotExists = fmt.Errorf("docker socket does not exist")
var ErrDockerSocketCantBeReached = fmt.Errorf("docker socket cannot be reached")
var ErrConfigFileNotFound = fmt.Errorf("config file not found")
var ErrWritingDefaultConfig = fmt.Errorf("error writing default config")
var ErrEditingConfig = fmt.Errorf("error editing config")

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
		return ErrDockerSocketCantBeReached
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

func getConfig(configPath string) (*entity.CconnectorConfig, error) {
	err := checkConfig(configPath)
	if err != nil {
		return nil, err
	}

	// Read the existing config file
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Unmarshal the config data into a struct
	currentConfig := &entity.CconnectorConfig{}
	if err := yaml.Unmarshal(configData, currentConfig); err != nil {
		return nil, err
	}

	return currentConfig, nil
}

// editConfig edits an existing config file with given new values.
func editConfig(configPath string, newConfig entity.CconnectorConfig) error {
	// Check if the config file exists
	err := checkConfig(configPath)
	if err != nil {
		return errors.Join(ErrEditingConfig, err)
	}

	// Marshal the modified struct back to YAML
	updatedConfigData, err := yaml.Marshal(&newConfig)
	if err != nil {
		return errors.Join(ErrEditingConfig, err)
	}

	// Write the updated YAML back to the configuration file
	err = os.WriteFile(configPath, updatedConfigData, 0644)
	if err != nil {
		return errors.Join(ErrEditingConfig, err)
	}

	return nil
}

func generateBearerToken(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("token length must be greater than zero")
	}

	// Generate random bytes
	tokenBytes := make([]byte, length)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Encode the bytes to a hexadecimal string
	token := hex.EncodeToString(tokenBytes)

	return token, nil
}
