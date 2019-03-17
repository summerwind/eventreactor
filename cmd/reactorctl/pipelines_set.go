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

	new, err := loadPipelineFromFile(f)
	if err != nil {
		return err
	}

	new.Namespace = namespace

	err = new.Validate()
	if err != nil {
		return err
	}

	key := types.NamespacedName{
		Name:      new.Name,
		Namespace: new.Namespace,
	}

	pipeline := &v1alpha1.Pipeline{}
	err = c.Get(context.TODO(), key, pipeline)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		err = c.Create(context.TODO(), new)
		if err != nil {
			return err
		}

		fmt.Printf("Pipeline \"%s\" created\n", new.Name)

		return nil
	}

	p := pipeline.DeepCopy()
	p.ObjectMeta.Labels = new.ObjectMeta.Labels
	p.ObjectMeta.Annotations = new.ObjectMeta.Annotations
	p.Spec = new.Spec

	err = c.Update(context.TODO(), p)
	if err != nil {
		return err
	}

	fmt.Printf("Pipeline \"%s\" configured\n", p.Name)

	return nil
}
