package main

import (
	"os"

	"github.com/insomnius/agent/cmd"
)

func main() {
	configPath := "/etc/cconnector/config.yaml"
	if envConfigPath := os.Getenv("CCONECTOR_CONFIG_PATH"); envConfigPath != "" {
		configPath = envConfigPath
	}

	configCmd := cmd.NewConfig(configPath)
	tokenCmd := cmd.NewToken(configPath)

	cconnector := cmd.NewRoot().Cconnector()
	cconnector.AddCommand(
		configCmd.Config(),
		configCmd.Initiate(),
		tokenCmd.Generate(),
		tokenCmd.Manager(),
	)
	_ = cconnector.Execute()
}
