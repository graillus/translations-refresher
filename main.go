package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/robfig/cron/v3"
	"k8s.io/client-go/util/homedir"
)

func main() {
	var (
		kubeconfig    *string
		enableCron    *bool
		enableWebhook *bool

		cronSpec *string
	)

	// Kubeconfig flag
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	// Boolean flags
	enableCron = flag.Bool("cron", true, "Enable periodic translations refreshes in the backgound")
	enableWebhook = flag.Bool("webhook", false, "Enable mutation webhook endpoint")

	// Cron rule flag
	cronSpec = flag.String("cronSpec", "*/2 * * * *", "Cron of the translations refreshes")

	flag.Parse()

	_, err := cron.ParseStandard(*cronSpec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cron flag cannot be parsed: %s", err)
	}

	config := &Config{
		enableCron:    *enableCron,
		enableWebhook: *enableWebhook,
		kubeconfig:    *kubeconfig,
		cronSpec:      *cronSpec,

		locoAPIKeys: map[string]string{
			"catalog":   os.Getenv("LOCO_API_KEY_CATALOG"),
			"documents": os.Getenv("LOCO_API_KEY_DOCUMENTS"),
			"emails":    os.Getenv("LOCO_API_KEY_EMAILS"),
		},
		tlsCertFile:       os.Getenv("TLS_CERT_FILE"),
		tlsPrivateKeyFile: os.Getenv("TLS_PRIVATE_KEY_FILE"),
	}

	app := newApp(config)
	err = app.Init()
	if err != nil {
		panic(err.Error())
	}

	app.Run()
}
