package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

type Config struct {
	configPath string
}

func NewConfig(configPath string) *Config {
	return &Config{
		configPath: configPath,
	}
}

func (c *Config) Initiate() *cobra.Command {
	return &cobra.Command{
		Use:   "config:initiate",
		Short: "Initiate required config for ccontainer",
		Long:  "A command to initiate config.yaml for ccontainer with default value. Be careful, it will replace all existing value.",
		Run: func(cmd *cobra.Command, args []string) {
			if err := createDefaultConfig(c.configPath); err != nil {
				fmt.Printf("Failed to initiate default config. Errors:\n%v\n", err)
				return
			}

			fmt.Println("Success initiate default config...")
		},
	}
}
