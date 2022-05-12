package main

import (
	"fmt"
	"log"

	"k8s.io/client-go/kubernetes"
)

// Refresher can detect outdated translations and refresh kubernetes resources accordingly
type Refresher struct {
	repository *Repository
	namespaces []string
	parser     *AnnotationParserWriter
}

// NewRefresher create a new Refresher instance
func NewRefresher(clientset *kubernetes.Clientset) *Refresher {
	return &Refresher{
		NewRepository(clientset, "translations.etsglobal.org/refresh=true"),
		[]string{"default"},
		&AnnotationParserWriter{"translations.etsglobal.org"},
	}
}

// Refresh runs the refresh process on all namespaces
func (r Refresher) Refresh(hashes *Hashes) {
	for _, ns := range r.namespaces {
		deployments := r.repository.FindDeployments(ns)
		log.Printf("There are %d deployments subscribed to translation refresh in namespace %s\n", len(deployments), ns)

		for idx := range deployments {
			deploy := &deployments[idx]

			// Apply the translation sync logic
			handleResource(r.parser, hashes, &deploy.ObjectMeta, &deploy.Spec.Template)
		}

		r.repository.UpdateDeployments(ns, deployments)
	}
}

// computeChangeset compares the hashes extracted on the resource with the fresh translation hashes
func computeChangeset(domains []string, currentHashes map[string]string, hashes Hashes) map[string]string {
	// Make sure the translation domains from annotations exist
	validDomains := []string{}
	for _, domain := range domains {
		if _, ok := hashes[domain]; ok {
			validDomains = append(validDomains, domain)

			continue
		}

		log.Println("Warning: Unknown translation domain " + domain)
	}

	changeset := make(map[string]string)
	for _, domain := range validDomains {
		if hash, ok := currentHashes[domain]; ok {
			// Compare the hash from the current annotation to the hash from Loco
			if hash == hashes[domain] {
				fmt.Println("Translations for domain " + domain + " up-to-date.")
				continue
			}

			// The hash from the annotation is outdated, let's update with newer hash
			log.Println("Translations for domain " + domain + " outdated.")
			changeset[domain] = hashes[domain]

			continue
		}

		log.Println("Warning: No hash annotation found for domain " + domain)

		// The deployment doesn't have the annotation matching the domain, we will create it.
		changeset[domain] = hashes[domain]
	}

	return changeset
}
