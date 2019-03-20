package main

import (
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
	"k8s.io/apimachinery/pkg/util/yaml"
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

func loadPipelinesFromFile(f string) ([]*v1alpha1.Pipeline, error) {
	var (
		reader io.Reader
		err    error
	)

	switch {
	case strings.HasPrefix(f, "http://") || strings.HasPrefix(f, "https://"):
		res, err := http.Get(f)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()

		reader = res.Body
	case f == "-":
		reader = os.Stdin
	default:
		reader, err = os.Open(f)
		if err != nil {
			return nil, err
		}
	}

	decoder := yaml.NewYAMLOrJSONDecoder(reader, 4096)

	pipelines := []*v1alpha1.Pipeline{}
	for {
		p := &v1alpha1.Pipeline{}
		err := decoder.Decode(p)
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		}

		pipelines = append(pipelines, p)
	}

	return pipelines, nil
}
