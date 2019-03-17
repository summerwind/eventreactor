package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func NewPipelinesSetCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "set [flags]",
		Short: "Set a pipeline configuration",
		Long:  "Set a pipeline configuration.",
		RunE:  pipelinesSetRun,
	}

	flags := cmd.Flags()
	flags.StringP("filename", "f", "", "Filename, URL to files that contains the pipeline configuration to set")

	return cmd
}

func pipelinesSetRun(cmd *cobra.Command, args []string) error {
	flags := cmd.Flags()

	f, err := flags.GetString("filename")
	if err != nil {
		return err
	}

	if f == "" {
		return errors.New("filename must be specified")
	}

	pipelines, err := loadPipelinesFromFile(f)
	if err != nil {
		return err
	}

	for _, p := range pipelines {
		err = p.Validate()
		if err != nil {
			return fmt.Errorf("pipeline \"%s\" is invalid: %v", p.Name, err)
		}
	}

	for _, p := range pipelines {
		if p.Namespace == "" {
			p.Namespace = namespace
		}

		key := types.NamespacedName{
			Name:      p.Name,
			Namespace: p.Namespace,
		}

		pipeline := &v1alpha1.Pipeline{}
		err = c.Get(context.TODO(), key, pipeline)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}

			err = c.Create(context.TODO(), p)
			if err != nil {
				return err
			}

			fmt.Printf("pipeline \"%s\" created\n", p.Name)

			return nil
		}

		pipeline.ObjectMeta.Labels = p.ObjectMeta.Labels
		pipeline.ObjectMeta.Annotations = p.ObjectMeta.Annotations
		pipeline.Spec = p.Spec

		err = c.Update(context.TODO(), pipeline)
		if err != nil {
			return err
		}

		fmt.Printf("pipeline \"%s\" configured\n", pipeline.Name)
	}

	return nil
}
