package cmd

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"github.com/obezsmertnyi/k8s-custom-controller/pkg/informer"
)

// StartComponents initializes and runs all enabled components (informer and API server)
func StartComponents(config *Config) error {
	// Create a context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Create Kubernetes client
	var clientset *kubernetes.Clientset
	if !config.Kubernetes.InCluster {
		kubePath := kubeconfig
		if kubePath == "" {
			kubePath = config.Kubernetes.Kubeconfig
		}
		log.Info().Str("kubeconfig", kubePath).Msg("Using kubeconfig file for authentication")
		var err error
		clientset, err = informer.CreateClientset(kubePath, false, config.ToInformerOptions())
		if err != nil {
			log.Error().Err(err).Msg("Failed to create Kubernetes clientset from kubeconfig")
			return err
		}
	} else {
		log.Info().Msg("Using in-cluster authentication")
		var err error
		clientset, err = informer.CreateClientset("", true, config.ToInformerOptions())
		if err != nil {
			log.Error().Err(err).Msg("Failed to create Kubernetes clientset with in-cluster config")
			return err
		}
	}
	log.Info().Msg("Successfully connected to Kubernetes cluster")

	// Create shared informer factory
	informerOpts := config.ToInformerOptions()
	var factory informers.SharedInformerFactory

	// Create informer factory with options
	factoryOptions := []informers.SharedInformerOption{}

	// Set namespace if specified
	if config.Informer.Namespace != "" {
		log.Debug().Str("namespace", config.Informer.Namespace).Msg("Limiting informer to namespace")
		factoryOptions = append(factoryOptions, informers.WithNamespace(config.Informer.Namespace))
	}

	// Create the factory
	factory = informers.NewSharedInformerFactoryWithOptions(
		clientset,
		informerOpts.ResyncPeriod,
		factoryOptions...,
	)

	// Start informer factory if informer is enabled
	if !config.Kubernetes.DisableInformer {
		// Start all requested informers
		log.Info().Msg("Starting shared informer factory")
		factory.Start(ctx.Done())
	}

	// Start components
	wg := sync.WaitGroup{}

	// Start API server if not disabled
	if !config.Kubernetes.DisableAPI {
		wg.Add(1)
		host := serverHost
		port := serverPort
		go func() {
			defer wg.Done()
			log.Info().Msg("Starting API server...")
			if err := StartAPIServer(ctx, clientset, factory, host, port); err != nil {
				log.Error().Err(err).Msg("Error running API server")
			}
		}()
	} else {
		log.Info().Msg("API server is disabled via configuration")
	}

	// Start informer if not disabled
	if !config.Kubernetes.DisableInformer {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Info().Msg("Starting deployment informer...")
			// Use options from configuration
			informerOpts := config.ToInformerOptions()
			if err := informer.StartDeploymentInformer(ctx, clientset, informerOpts); err != nil {
				log.Error().Err(err).Msg("Error running deployment informer")
			}
		}()
	} else {
		log.Info().Msg("Deployment informer is disabled via configuration")
	}

	// Wait for termination signal
	log.Info().Msg("Press Ctrl+C to stop the service")
	<-sigCh
	log.Info().Msg("Shutdown signal received, stopping services...")

	// Cancel context and wait for goroutines to complete
	cancel()
	wgCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(wgCh)
	}()

	// Wait for all components to stop with timeout
	shutdownTimeout := 5 * time.Second
	select {
	case <-wgCh:
		log.Info().Msg("All services stopped gracefully")
	case <-time.After(shutdownTimeout):
		log.Warn().Msg("Some services did not stop gracefully within the timeout")
	}

	return nil
}
