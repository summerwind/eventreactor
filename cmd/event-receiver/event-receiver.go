package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/summerwind/eventreactor/api/v1alpha1"
)

var (
	namespace string
	addr      string
	port      int
	certFile  string
	keyFile   string

	c   client.Client
	log logr.Logger
)

var allowedContentType = []string{
	"text/",
	"application/json",
	"application/xml",
	"application/x-www-form-urlencoded",
	"application/cloudevents+json",
}

const supportedSpecVersion = "1.0"

type CloudEvent struct {
	SpecVersion     string          `json:"specversion"`
	ID              string          `json:"id"`
	Source          string          `json:"source"`
	Type            string          `json:"type"`
	DataContentType string          `json:"datacontenttype"`
	DataSchema      string          `json:"dataschema"`
	Subject         string          `json:"subject"`
	Time            string          `json:"time"`
	Data            json.RawMessage `json:"data"`
}

func parseRequest(r *http.Request) (*v1alpha1.Event, error) {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return nil, errors.New("content-type header must be specified")
	}

	allowed := false
	for _, prefix := range allowedContentType {
		if strings.HasPrefix(contentType, prefix) {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, fmt.Errorf("unsupported content-type: %s", contentType)
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	ev := v1alpha1.Event{}
	if contentType == "application/cloudevents+json" {
		ce := &CloudEvent{}
		if err := json.Unmarshal(body, ce); err != nil {
			return nil, err
		}

		if ce.SpecVersion != supportedSpecVersion {
			return nil, fmt.Errorf("unsupported specversion: %s", ce.SpecVersion)
		}

		ev.Spec = v1alpha1.EventSpec{
			ID:              ce.ID,
			Source:          ce.Source,
			Type:            ce.Type,
			DataContentType: ce.DataContentType,
			DataSchema:      ce.DataSchema,
			Subject:         ce.Subject,
			Data:            string(ce.Data),
		}

		if ce.Time != "" {
			cet, err := time.Parse(time.RFC3339, ce.Time)
			if err == nil {
				t := metav1.NewTime(cet)
				ev.Spec.Time = &t
			}
		}
	} else {
		specVersion := r.Header.Get("ce-specversion")
		if specVersion != supportedSpecVersion {
			return nil, fmt.Errorf("unsupported specversion: %s", specVersion)
		}

		ev.Spec = v1alpha1.EventSpec{
			ID:              r.Header.Get("ce-id"),
			Source:          r.Header.Get("ce-source"),
			Type:            r.Header.Get("ce-type"),
			DataContentType: r.Header.Get("content-type"),
			DataSchema:      r.Header.Get("ce-dataschema"),
			Subject:         r.Header.Get("ce-subject"),
			Data:            string(body),
		}

		if r.Header.Get("ce-time") != "" {
			cet, err := time.Parse(time.RFC3339, r.Header.Get("ce-time"))
			if err == nil {
				t := metav1.NewTime(cet)
				ev.Spec.Time = &t
			}
		}
	}

	return &ev, nil
}

func eventHandler(w http.ResponseWriter, r *http.Request) {
	reqLog := log.WithValues("remote_addr", r.RemoteAddr)

	if r.Method != http.MethodPost {
		reqLog.V(1).Info("Invalid request method", "method", r.Method)
		http.Error(w, "Not Implemented", http.StatusNotImplemented)
		return
	}

	ev, err := parseRequest(r)
	if err != nil {
		reqLog.Error(err, "Invalid request")
		http.Error(w, "Bad Request", http.StatusBadRequest)
	}

	ev.ObjectMeta = metav1.ObjectMeta{
		Namespace: namespace,
		Name:      v1alpha1.NewEventName(),
	}

	if err := c.Create(context.Background(), ev); err != nil {
		reqLog.Error(err, "Failed to create event resource", "name", ev.Name, "namespace", ev.Namespace)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	reqLog.Info("Event resource created", "name", ev.Name, "namespace", ev.Namespace)
}

func run(cmd *cobra.Command, args []string) error {
	config, err := config.GetConfig()
	if err != nil {
		return err
	}

	mapper, err := apiutil.NewDynamicRESTMapper(config)
	if err != nil {
		return err
	}

	sc := scheme.Scheme
	if err := v1alpha1.AddToScheme(sc); err != nil {
		return err
	}

	c, err = client.New(config, client.Options{Scheme: sc, Mapper: mapper})
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1alpha1/events", eventHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", addr, port),
		Handler: mux,
	}

	go func() {
		log.Info("Starting server", "addr", addr)
		if certFile != "" && keyFile != "" {
			server.ListenAndServeTLS(certFile, keyFile)
		} else {
			server.ListenAndServe()
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM)
	<-sigCh

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Info("Stopping server")
	if err := server.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}

func main() {
	ctrl.SetLogger(zap.New(func(o *zap.Options) {
		o.Development = true
	}))
	log = ctrl.Log.WithName("event-receiver")

	var cmd = &cobra.Command{
		Use:   "event-receiver",
		Short: "Receives CloudEvents formatted event and creates resource",
		RunE:  run,

		SilenceErrors: true,
		SilenceUsage:  true,
	}

	flags := cmd.Flags()
	flags.StringVarP(&namespace, "namespace", "n", "default", "The namespace to create Event resources")
	flags.StringVar(&addr, "bind-address", "0.0.0.0", "The IP address on which to listen")
	flags.IntVar(&port, "port", 14380, "The port on which to listen")
	flags.StringVar(&certFile, "tls-cert-file", "", "File containing the default x509 Certificate for HTTPS")
	flags.StringVar(&keyFile, "tls-private-key-file", "", "File containing the default x509 private key matching --tls-cert-file")

	err := cmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
