package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

func NewPipelinesRunCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "run <pipeline> [flags]",
		Short: "Run the specified pipeline",
		Long:  "Run the specified pipeline.",
		RunE:  pipelinesRunRun,
	}

	flags := cmd.Flags()
	flags.StringP("event", "e", "", "The event that triggers pipeline")
	flags.StringP("action", "a", "", "The action that triggers pipeline")

	return cmd
}

func pipelinesRunRun(cmd *cobra.Command, args []string) error {
	var (
		name   string
		action *v1alpha1.Action
		err    error
	)

	flags := cmd.Flags()

	if len(args) > 0 {
		name = args[0]
	}

	eventName, err := flags.GetString("event")
	if err != nil {
		return err
	}
	actionName, err := flags.GetString("action")
	if err != nil {
		return err
	}

	if name == "" {
		return errors.New("pipeline name must be specified")
	}
	if eventName == "" && actionName == "" {
		return errors.New("event or action must be specified")
	}
	if eventName != "" && actionName != "" {
		return errors.New("exactly one of event or action must be specified")
	}

	key := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}

	pipeline := &v1alpha1.Pipeline{}
	err = c.Get(context.TODO(), key, pipeline)
	if err != nil {
		return err
	}

	if actionName == "" {
		ekey := types.NamespacedName{
			Name:      eventName,
			Namespace: namespace,
		}

		e := &v1alpha1.Event{}
		err := c.Get(context.TODO(), ekey, e)
		if err != nil {
			return err
		}

		action, err = pipeline.NewActionWithEvent(e)
		if err != nil {
			return err
		}
	} else {
		akey := types.NamespacedName{
			Name:      actionName,
			Namespace: namespace,
		}

		a := &v1alpha1.Action{}
		err := c.Get(context.TODO(), akey, a)
		if err != nil {
			return err
		}

		action, err = pipeline.NewActionWithAction(a)
		if err != nil {
			return err
		}
	}

	err = c.Create(context.Background(), action)
	if err != nil {
		return err
	}

	fmt.Println(action.Name)

	return nil
}
