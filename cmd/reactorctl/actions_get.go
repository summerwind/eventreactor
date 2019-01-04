package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"

	"github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
)

func NewActionsGetCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "get <action>",
		Short: "Print the information of action",
		Long:  "Print the information of action.",
		RunE:  actionsGetRun,
	}

	return cmd
}

func actionsGetRun(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("Action name required")
	}

	key := types.NamespacedName{
		Name:      args[0],
		Namespace: namespace,
	}

	action := &v1alpha1.Action{}
	err := c.Get(context.TODO(), key, action)
	if err != nil {
		return err
	}

	out, err := render(actionTemplate, action)
	if err != nil {
		return err
	}

	fmt.Print(out)

	return nil
}

const actionTemplate = `
Name: {{ .ObjectMeta.Name }}
Date: {{ .ObjectMeta.CreationTimestamp }}

{{ range $i, $s := .Status.BuildStatus.StepsCompleted -}}
[{{ $s }}]
{{ index $.Status.StepLogs $i }}
{{ end -}}
`
