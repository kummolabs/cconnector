package cmd

import "github.com/spf13/cobra"

type Root struct{}

func NewRoot() *Root {
	return &Root{}
}

func (r *Root) Cconnector() *cobra.Command {
	return &cobra.Command{
		Use:   "cconector",
		Short: "Host container connector",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
}
