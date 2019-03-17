package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

func NewPipelinesDeleteCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "delete [flags]",
		Short: "Delete a pipeline configuration",
		Long:  "Delete a pipeline configuration.",
		RunE:  pipelinesDeleteRun,
	}

	flags := cmd.Flags()
	flags.StringP("filename", "f", "", "Filename, URL to files that contains the pipeline configuration to set")

	return cmd
}

func pipelinesDeleteRun(cmd *cobra.Command, args []string) error {
	var (
		name string
		keys []types.NamespacedName
	)

	flags := cmd.Flags()

	if len(args) > 0 {
		name = args[0]
	}

	f, err := flags.GetString("filename")
	if err != nil {
		return err
	}

	if name == "" && f == "" {
		return errors.New("pipeline name or filename must be specified")
	}
	if name != "" && f != "" {
		return errors.New("when filename is provided as input, you may not specify resource arguments as well")
	}

	if f != "" {
		pipelines, err := loadPipelinesFromFile(f)
		if err != nil {
			return err
		}

		for _, p := range pipelines {
			keys = append(keys, types.NamespacedName{
				Name:      p.Name,
				Namespace: p.Namespace,
			})
		}
	} else {
		keys = append(keys, types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		})
	}

	for _, key := range keys {
		if key.Namespace == "" {
			key.Namespace = namespace
		}

		pipeline := v1alpha1.Pipeline{}
		err = c.Get(context.TODO(), key, &pipeline)
		if err != nil {
			return err
		}

		err = c.Delete(context.TODO(), &pipeline)
		if err != nil {
			return err
		}

		fmt.Printf("pipeline \"%s\" deleted\n", pipeline.Name)
	}

	return nil
}
