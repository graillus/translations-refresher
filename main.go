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
		enableWebhook *bool
		kubeconfig    *string
		schedule      *string
	)

	// Kubeconfig flag
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	enableWebhook = flag.Bool("webhook", false, "Enable mutation webhook endpoint")
	schedule = flag.String("schedule", "", "Cron schedule expression of the translations refreshes")

	flag.Parse()

	if *schedule != "" {
		_, err := cron.ParseStandard(*schedule)
		if err != nil {
			fmt.Fprintf(os.Stderr, "-schedule flag cannot be parsed: %s", err)
		}
	}

	config := &Config{
		enableWebhook: *enableWebhook,
		kubeconfig:    *kubeconfig,
		schedule:      *schedule,

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
