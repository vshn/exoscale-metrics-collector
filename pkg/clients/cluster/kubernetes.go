package cluster

import (
	"fmt"

	cloudscaleapis "github.com/vshn/provider-cloudscale/apis"
	exoapis "github.com/vshn/provider-exoscale/apis"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewClient creates a k8s client from the server url and token url
// If kubeconfig (path to it) is supplied, that takes precedence. Its use is mainly for local development
// since local clusters usually don't have a valid certificate.
func NewClient(kubeconfig, url, token string) (client.Client, error) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("core scheme: %w", err)
	}
	if err := exoapis.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("exoscale scheme: %w", err)
	}
	if err := cloudscaleapis.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("cloudscale scheme: %w", err)
	}

	config, err := restConfig(kubeconfig, url, token)
	if err != nil {
		return nil, fmt.Errorf("k8s rest config: %w", err)
	}

	c, err := client.New(config, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot initialize k8s client: %w", err)
	}
	return c, nil
}

// NewDynamicClient creates a dynamic k8s client from the server url and token url
func NewDynamicClient(kubeconfig, url, token string) (dynamic.Interface, error) {
	config, err := restConfig(kubeconfig, url, token)
	if err != nil {
		return nil, fmt.Errorf("k8s rest config: %w", err)
	}
	c, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize k8s client: %w", err)
	}
	return c, nil
}

func restConfig(kubeconfig string, url string, token string) (*rest.Config, error) {
	// kubeconfig takes precedence if set.
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return &rest.Config{Host: url, BearerToken: token}, nil
}
