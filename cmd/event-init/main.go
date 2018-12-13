package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/spf13/cobra"
	"github.com/summerwind/eventreactor/pkg/apis"
	"github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
)

const (
	eventDefaultPath = "/builder/home/event"
)

func run(cmd *cobra.Command, args []string) error {
	namespace, err := cmd.Flags().GetString("namespace")
	if err != nil {
		return err
	}
	if namespace == "" {
		return errors.New("namespace must not be empty")
	}

	eventName, err := cmd.Flags().GetString("event")
	if err != nil {
		return err
	}
	if eventName == "" {
		return errors.New("event must not be empty")
	}

	baseDir, err := cmd.Flags().GetString("path")
	if err != nil {
		return err
	}
	baseDir = filepath.Clean(baseDir)

	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	mapper, err := apiutil.NewDiscoveryRESTMapper(cfg)
	if err != nil {
		return err
	}

	sc := scheme.Scheme
	if err := apis.AddToScheme(sc); err != nil {
		return err
	}

	c, err := client.New(cfg, client.Options{Scheme: sc, Mapper: mapper})
	if err != nil {
		return err
	}

	eventKey := types.NamespacedName{
		Namespace: namespace,
		Name:      eventName,
	}

	ev := &v1alpha1.Event{}
	err = c.Get(context.TODO(), eventKey, ev)
	if err != nil {
		return err
	}

	_, err = os.Stat(baseDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(baseDir, 0755)
		if err != nil {
			return err
		}
	}

	typePath := filepath.Join(baseDir, "type")
	err = ioutil.WriteFile(typePath, []byte(ev.Spec.Type), 0644)
	if err != nil {
		return err
	}

	sourcePath := filepath.Join(baseDir, "source")
	err = ioutil.WriteFile(sourcePath, []byte(ev.Spec.Source), 0644)
	if err != nil {
		return err
	}

	contentTypePath := filepath.Join(baseDir, "content-type")
	err = ioutil.WriteFile(contentTypePath, []byte(ev.Spec.ContentType), 0644)
	if err != nil {
		return err
	}

	dataPath := filepath.Join(baseDir, "data")
	err = ioutil.WriteFile(dataPath, []byte(ev.Spec.Data), 0644)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	var cmd = &cobra.Command{
		Use:   "event-init",
		Short: "Event initializer",
		RunE:  run,

		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.Flags().StringP("namespace", "n", "default", "The namespace")
	cmd.Flags().StringP("event", "e", "", "The event name")
	cmd.Flags().StringP("path", "p", eventDefaultPath, "The path of event file")

	err := cmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
