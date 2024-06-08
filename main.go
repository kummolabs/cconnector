package main

import "github.com/insomnius/agent/cmd"

func main() {
	configPath := "/etc/cconnector/config.yaml"

	configCmd := cmd.NewConfig(configPath)
	tokenCmd := cmd.NewToken(configPath)

	cconnector := cmd.NewRoot().Cconnector()
	cconnector.AddCommand(
		configCmd.Initiate(),
		tokenCmd.Generate(),
	)
	_ = cconnector.Execute()
}
