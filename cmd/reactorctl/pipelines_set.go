package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/ghodss/yaml"
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

	var buf []byte

	switch {
	case strings.HasPrefix(f, "http://") || strings.HasPrefix(f, "https://"):
		res, err := http.Get(f)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		buf, err = ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
	case f == "-":
		buf, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
	default:
		buf, err = ioutil.ReadFile(f)
		if err != nil {
			return err
		}
	}

	new := v1alpha1.Pipeline{}
	new.Namespace = namespace

	err = yaml.Unmarshal(buf, &new)
	if err != nil {
		return err
	}

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

		err = c.Create(context.TODO(), &new)
		if err != nil {
			return err
		}

		fmt.Println(new.Name)

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

	fmt.Println(p.Name)

	return nil
}
