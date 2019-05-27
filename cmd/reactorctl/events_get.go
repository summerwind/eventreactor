package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"

	"github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
)

func NewEventsGetCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "get <event>",
		Short: "Print the information of event",
		Long:  "Print the information of event.",
		RunE:  eventsGetRun,
	}

	return cmd
}

func eventsGetRun(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("Event name required")
	}

	key := types.NamespacedName{
		Name:      args[0],
		Namespace: namespace,
	}

	event := &v1alpha1.Event{}
	err := c.Get(context.TODO(), key, event)
	if err != nil {
		return err
	}

	out, err := render(eventTemplate, event)
	if err != nil {
		return err
	}

	fmt.Print(out)

	return nil
}

const eventTemplate = `
ID:          {{ .Spec.ID }}
Time:        {{ .Spec.Time }}
Type:        {{ .Spec.Type }}
Source:      {{ .Spec.Source }}
ContentType: {{ .Spec.ContentType }}
Data:
{{ .Spec.Data }}
`
