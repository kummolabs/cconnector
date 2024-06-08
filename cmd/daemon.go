package cmd

import "github.com/spf13/cobra"

type Daemon struct {
	configPath string
}

func NewDaemon(configPath string) *Daemon {
	return &Daemon{
		configPath: configPath,
	}
}

func (d *Daemon) Start() *cobra.Command {
	return &cobra.Command{
		Use:     "daemon:start",
		Short:   "Run cconector daemon",
		Long:    "Run cconector daemon, which include: HTTP servers",
		GroupID: "daemon",
		Run: func(cmd *cobra.Command, args []string) {

		},
	}
}
