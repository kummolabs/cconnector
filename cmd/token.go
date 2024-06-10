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
		Use:     "token:generate",
		Short:   "Generate bearer authentication token for current host",
		Long:    "A command to generate bearer authentication token for current host, to verify incoming request from manager",
		GroupID: "token",
		Run: func(cmd *cobra.Command, args []string) {
			if currentConfig, err := getConfig(t.configPath); err != nil {
				fmt.Printf("Failed to identify cconector config, you can initiate your config by running `cconector config:initiate`. Errors:\n%v\n", err)
				return
			} else {
				token, err := generateBearerToken(32)
				if err != nil {
					fmt.Printf("Failed to generate bearer token. Errors:\n%v\n", err)
					return
				}
				currentConfig.HostToken = token

				if err := editConfig(t.configPath, *currentConfig); err != nil {
					fmt.Printf("Failed to updating new config. Errors:\n%v\n", err)
					return
				}

				fmt.Printf("Token generation is complete, your host token: `%s`\n", currentConfig.HostToken)
			}
		},
	}
}

func (t *Token) Manager() *cobra.Command {
	return &cobra.Command{
		Use:     "token:manager",
		Short:   "Set manager token",
		Long:    "A command to set manager token.",
		GroupID: "token",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if currentConfig, err := getConfig(t.configPath); err != nil {
				fmt.Printf("Failed to identify cconector config, you can initiate your config by running `cconector config:initiate`. Errors:\n%v\n", err)
				return
			} else {
				currentConfig.ManagerToken = args[0]
				if err := editConfig(t.configPath, *currentConfig); err != nil {
					fmt.Printf("Failed to updating new config. Errors:\n%v\n", err)
					return
				}

				fmt.Println("Manager token is succesfully set...")
			}
		},
	}
}

func (t *Token) Reset() *cobra.Command {
	return &cobra.Command{
		Use:     "token:reset",
		Short:   "Reset manager token and generate new token host token in the process",
		GroupID: "token",
		Run: func(cmd *cobra.Command, args []string) {
			t.Generate().Run(cmd, args)

			if currentConfig, err := getConfig(t.configPath); err != nil {
				fmt.Printf("Failed to identify cconector config, you can initiate your config by running `cconector config:initiate`. Errors:\n%v\n", err)
				return
			} else {
				currentConfig.ManagerToken = ""
				if err := editConfig(t.configPath, *currentConfig); err != nil {
					fmt.Printf("Failed to updating new config. Errors:\n%v\n", err)
					return
				}

				fmt.Println("Manager token is succesfully reset...")
			}
		},
	}
}
