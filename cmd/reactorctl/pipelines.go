package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
)

func NewPipelinesCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "pipelines",
		Short: "Manage pipeline",
		Long:  "Manage pipeline.",
	}

	cmd.AddCommand(NewPipelinesListCommand())
	cmd.AddCommand(NewPipelinesSetCommand())
	cmd.AddCommand(NewPipelinesDeleteCommand())

	return cmd
}

func loadPipelineFromFile(f string) (*v1alpha1.Pipeline, error) {
	var (
		buf []byte
		err error
	)

	switch {
	case strings.HasPrefix(f, "http://") || strings.HasPrefix(f, "https://"):
		res, err := http.Get(f)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()

		buf, err = ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
	case f == "-":
		buf, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			return nil, err
		}
	default:
		buf, err = ioutil.ReadFile(f)
		if err != nil {
			return nil, err
		}
	}

	pipeline := v1alpha1.Pipeline{}

	err = yaml.Unmarshal(buf, &pipeline)
	if err != nil {
		return nil, err
	}

	return &pipeline, nil
}
