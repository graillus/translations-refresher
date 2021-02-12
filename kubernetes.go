package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/util/retry"
)

// AnnotationParser parses annotations to find translation metadata
type AnnotationParser interface {
	// ParseDomains retrieves the list of translation domains
	ParseDomains(annotations map[string]string) []string

	// ParseDomainHashes will retrieve a map of translation domains/hashes
	ParseDomainHashes(annotations map[string]string) map[string]string
}

// AnnotationWriter can write a set of changes to the pod template annotations
type AnnotationWriter interface {
	// WriteChanges updates the pod template annotations with new values from changeset
	WriteChanges(podSpec *corev1.PodTemplateSpec, changeset map[string]string)
}

// AnnotationParserWriter parses and updates annotations
type AnnotationParserWriter struct {
	prefix string
}

// ParseDomains implements the AnnotationParser.ParseDomain method
func (a AnnotationParserWriter) ParseDomains(annotations map[string]string) []string {
	var domains []string
	// Parse the deployment's annotations to find the translation domains
	for key, domain := range annotations {
		if key == a.prefix+"/domains" {
			domains = strings.Split(domain, ",")
		}
	}

	return domains
}

// ParseDomainHashes implements the AnnotationParser.ParseDomainHashes method
func (a AnnotationParserWriter) ParseDomainHashes(annotations map[string]string) map[string]string {
	// Look for current version of the translations in pod template annotations
	hashes := make(map[string]string)
	for k, v := range annotations {
		// Filter annotations prefixed by our annotation domain
		if strings.HasPrefix(k, a.prefix+"/") {
			// Extract the translation domain and hash from the annotation
			hashes[strings.Replace(k, a.prefix+"/", "", 1)] = v
		}
	}

	return hashes
}

// WriteChanges implements the AnnotationWriter.WriteChanges method
func (a AnnotationParserWriter) WriteChanges(podSpec *corev1.PodTemplateSpec, changeset map[string]string) {
	// create an empty map if there is no existing annotations
	if podSpec.Annotations == nil {
		podSpec.Annotations = make(map[string]string)
	}

	// add or erase deployment's annotations from changeset
	for k, v := range changeset {
		podSpec.Annotations[a.prefix+"/"+k] = v
	}
}

// Repository is a kubernetes repository
type Repository struct {
	client   clientappsv1.AppsV1Interface
	selector metav1.ListOptions
}

// NewRepository creates an new Repository instance
func NewRepository(clientset *kubernetes.Clientset, selectorLabel string) *Repository {
	return &Repository{
		clientset.AppsV1(),
		metav1.ListOptions{
			LabelSelector: selectorLabel,
		},
	}
}

// FindDeployments gets the list of eligible Deployment resources
func (r Repository) FindDeployments(ns string) []appsv1.Deployment {
	client := r.client.Deployments(ns)
	list, err := client.List(context.TODO(), r.selector)
	if err != nil {
		log.Printf("No deployments found in namespace %s: %+v\n", ns, err)

		return []appsv1.Deployment{}
	}

	return list.Items
}

// UpdateDeployments applies updates of Deployment resources against the kubernetes API
func (r Repository) UpdateDeployments(ns string, deployments []appsv1.Deployment) {
	client := r.client.Deployments(ns)

	for _, deploy := range deployments {
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			_, err := client.Update(context.TODO(), &deploy, metav1.UpdateOptions{})

			return err
		})

		if retryErr != nil {
			log.Printf("Update failed: %+v\n", retryErr)

			continue
		}

		fmt.Printf("Updated deployment %s\n", deploy.Name)
	}

}

// ResourceHandler is a handler function with all translation sync logic for generic objects
type ResourceHandler func(_ context.Context, obj *metav1.Object)

// Appsv1ResourceHandler returns the handler function with all translation sync logic
// The handler function will take any resource from the apps/v1 API
func Appsv1ResourceHandler(annotationPrefix string, hashes *Hashes) ResourceHandler {
	parser := &AnnotationParserWriter{annotationPrefix}

	return func(_ context.Context, obj *metav1.Object) {
		switch v := (*obj).(type) {
		case *appsv1.DaemonSet:
			ds := (*obj).(*appsv1.DaemonSet)

			handleResource(parser, hashes, &ds.ObjectMeta, &ds.Spec.Template)
		case *appsv1.Deployment:
			deploy := (*obj).(*appsv1.Deployment)

			handleResource(parser, hashes, &deploy.ObjectMeta, &deploy.Spec.Template)
		case *appsv1.StatefulSet:
			sts := (*obj).(*appsv1.StatefulSet)

			handleResource(parser, hashes, &sts.ObjectMeta, &sts.Spec.Template)
		default:
			log.Printf("Warning: Resource of type %T is not supported\n", v)
		}
	}
}

func handleResource(parser *AnnotationParserWriter, hashes *Hashes, meta *metav1.ObjectMeta, podSpec *corev1.PodTemplateSpec) {

	domains := parser.ParseDomains(meta.Annotations)

	log.Printf("Resource %s subscribed to translations domains: %+v\n", meta.Name, domains)

	// Look for current version of the translations in pod template annotations
	currentHashes := parser.ParseDomainHashes(podSpec.Annotations)

	changeset := computeChangeset(domains, currentHashes, *hashes)

	log.Printf("Changeset: %+v\n", changeset)

	parser.WriteChanges(podSpec, changeset)
}
