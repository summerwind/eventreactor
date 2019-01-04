package main

import (
	"github.com/spf13/cobra"
)

func NewPipelineCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "pipelines",
		Short: "Manage pipeline",
		Long:  "Manage pipeline.",
	}

	cmd.AddCommand(NewPipelineListCommand())

	return cmd
}
