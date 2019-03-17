package main

import (
	"github.com/spf13/cobra"
)

func NewPipelinesCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "pipelines",
		Short: "Manage pipeline",
		Long:  "Manage pipeline.",
	}

	cmd.AddCommand(NewPipelinesListCommand())
	cmd.AddCommand(NewPipelinesSetCommand())

	return cmd
}
