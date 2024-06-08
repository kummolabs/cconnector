package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

type Token struct {
	configPath string
}

func NewToken(configPath string) *Token {
	return &Token{
		configPath: configPath,
	}
}

func (t *Token) Generate() *cobra.Command {
	return &cobra.Command{
		Use:   "token:generate",
		Short: "Generate bearer authentication token for current host",
		Long:  "A command to generate bearer authentication token for current host, to verify incoming request from manager",
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureConfig(t.configPath); err != nil {
				fmt.Printf("Failed to generate token, errors when ensuring the config file. Errors:\n%v\n", err)
				return
			}
		},
	}
}
