package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	whhttp "github.com/slok/kubewebhook/pkg/http"
	whlog "github.com/slok/kubewebhook/pkg/log"
	"github.com/slok/kubewebhook/pkg/observability/metrics"
	mutatingwh "github.com/slok/kubewebhook/pkg/webhook/mutating"
)

// MutatingWebhook is the mutating webhook
type MutatingWebhook struct {
	callback        ResourceHandler
	metricsRecorder *metrics.Prometheus
}

// NewMutatingWebhook creates a new MutatingWebhook instance
func NewMutatingWebhook(
	callback ResourceHandler,
	rec *metrics.Prometheus,
) *MutatingWebhook {
	return &MutatingWebhook{callback, rec}
}

// ListenAndServeTLS creates the HTTPS server and start listening connections
func (m *MutatingWebhook) ListenAndServeTLS(port string, tlsCertFile string, tlsPrivateKeyFile string) error {
	// Get the daemonsets handler for our webhook.
	ds, err := createWebhookHandler("daemonsets", &appsv1.DaemonSet{}, m.callback, m.metricsRecorder)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating webhook handler for daemonsets: %s", err)
		return err
	}

	// Get the deployments handler for our webhook.
	deploy, err := createWebhookHandler("deployments", &appsv1.Deployment{}, m.callback, m.metricsRecorder)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating webhook handler for deployments: %s", err)
		return err
	}

	// Get the statefulsets handler for our webhook.
	sts, err := createWebhookHandler("statefulsets", &appsv1.StatefulSet{}, m.callback, m.metricsRecorder)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating webhook handler for statefulsets: %s", err)
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/daemonsets", ds)
	mux.Handle("/deployments", deploy)
	mux.Handle("/statefulsets", sts)

	// Create HTTPS server
	fmt.Println("Webhook listening on port " + port)
	err = http.ListenAndServeTLS(port, tlsCertFile, tlsPrivateKeyFile, mux)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating webhook handler: %s", err)
		return err
	}

	return nil
}

func createWebhookHandler(name string, obj metav1.Object, callback ResourceHandler, m *metrics.Prometheus) (http.Handler, error) {
	config := mutatingwh.WebhookConfig{
		Name: name,
		Obj:  obj,
	}

	fn := handlerFunc(callback)

	// Create our mutator
	mutatorFunc := mutatingwh.MutatorFunc(fn)

	webhook, err := mutatingwh.NewWebhook(config, mutatorFunc, nil, m, &whlog.Std{Debug: true})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating webhook: %s", err)
		return nil, err
	}

	// Get the handler for our webhook.
	handler, err := whhttp.HandlerFor(webhook)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating webhook handler: %s", err)
		return nil, err
	}

	return handler, nil
}

func handlerFunc(callback ResourceHandler) func(ctx context.Context, obj metav1.Object) (bool, error) {
	return func(ctx context.Context, obj metav1.Object) (bool, error) {
		callback(ctx, &obj)

		return false, nil
	}
}
