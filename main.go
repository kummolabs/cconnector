package main

import (
	"os"

	"github.com/insomnius/agent/cmd"
	"github.com/spf13/cobra"
)

func main() {
	configPath := "/etc/cconnector/config.yaml"
	if envConfigPath := os.Getenv("CCONECTOR_CONFIG_PATH"); envConfigPath != "" {
		configPath = envConfigPath
	}

	configCmd := cmd.NewConfig(configPath)
	tokenCmd := cmd.NewToken(configPath)
	daemonCmd := cmd.NewDaemon(configPath)

	cconnector := cmd.NewRoot().Cconnector()
	cconnector.AddGroup(
		&cobra.Group{
			ID:    "daemon",
			Title: "daemon",
		},
		&cobra.Group{
			ID:    "config",
			Title: "config",
		},
		&cobra.Group{
			ID:    "token",
			Title: "token",
		},
	)
	cconnector.AddCommand(
		configCmd.Config(),
		configCmd.Initiate(),
		tokenCmd.Generate(),
		tokenCmd.Manager(),
		tokenCmd.Reset(),
		daemonCmd.Start(),
	)
	_ = cconnector.Execute()
}
