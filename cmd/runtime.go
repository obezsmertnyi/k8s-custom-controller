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

	// Determine whether components are enabled
	apiServerEnabled := !(config.APIServer.Enabled == false || config.Kubernetes.DisableAPI)
	informerEnabled := !(config.Informer.Enabled == false || config.Kubernetes.DisableInformer)

	log.Debug().Bool("api_server_enabled", apiServerEnabled).Bool("informer_enabled", informerEnabled).Msg("Component activation status")

	// Always initialize Kubernetes client for both CLI commands and services
	var clientset *kubernetes.Clientset
	var factory informers.SharedInformerFactory
	var err error

	// Initialize kubernetes client
	log.Debug().Msg("Initializing Kubernetes client")

	informerOpts := config.ToInformerOptions()

	if config.Kubernetes.InCluster {
		log.Info().Msg("Using in-cluster authentication")
		clientset, err = informer.CreateClientset("", true, informerOpts)
	} else {
		// Expand kubeconfig path if it starts with ~
		kubePath := config.Kubernetes.Kubeconfig
		if kubePath != "" && kubePath[0] == '~' {
			homeDir, err := os.UserHomeDir()
			if err == nil && homeDir != "" {
				if kubePath == "~" {
					kubePath = homeDir
				} else if len(kubePath) > 1 && kubePath[1] == '/' {
					kubePath = homeDir + kubePath[1:]
				}
				log.Debug().Str("expanded_path", kubePath).Msg("Expanded tilde in kubeconfig path")
			} else {
				log.Warn().Err(err).Msg("Failed to expand tilde in kubeconfig path")
			}
		}

		log.Info().Str("kubeconfig", kubePath).Msg("Using kubeconfig file for authentication")
		clientset, err = informer.CreateClientset(kubePath, false, informerOpts)
	}

	if err != nil {
		log.Error().Err(err).Msg("Failed to create Kubernetes clientset")
		return err
	}

	log.Debug().Msg("Successfully connected to Kubernetes cluster")

	// Create shared informer factory only if informer is enabled
	var factoryOptions []informers.SharedInformerOption

	// Always prepare factory options in case we need them for API server
	if config.Informer.Namespace != "" {
		log.Debug().Str("namespace", config.Informer.Namespace).Msg("Limiting informer to namespace")
		factoryOptions = append(factoryOptions, informers.WithNamespace(config.Informer.Namespace))
	}

	// Create the factory but only start it if informer is enabled
	factory = informers.NewSharedInformerFactoryWithOptions(
		clientset,
		config.ToInformerOptions().ResyncPeriod,
		factoryOptions...,
	)

	// Only start informer factory if informer is enabled
	if informerEnabled {
		log.Info().Msg("Starting shared informer factory")
		factory.Start(ctx.Done())
	} else {
		log.Debug().Msg("Informer factory created but not started - informer disabled")
	}

	// Start components
	wg := sync.WaitGroup{}

	// Determine whether to start the API server based on configuration priority
	// The default is to run API server unless explicitly disabled by configuration
	// By default, config.APIServer.Enabled is true and config.Kubernetes.DisableAPI is false
	// 1. New explicit setting (api_server.enabled: false) takes highest priority
	// 2. Legacy setting (kubernetes.disable_api: true) can also disable it
	log.Debug().Bool("api_server_enabled_config", config.APIServer.Enabled).Bool("kubernetes_disable_api", config.Kubernetes.DisableAPI).Msg("API server configuration")

	// Only don't start if explicitly disabled by new or legacy config
	if !(config.APIServer.Enabled == false || config.Kubernetes.DisableAPI) {
		wg.Add(1)
		// Use values from config, which already reflect CLI overrides if any
		host := config.APIServer.Host
		port := config.APIServer.Port

		go func() {
			defer wg.Done()
			log.Info().Msg("Starting API server...")
			if err := StartAPIServer(ctx, clientset, factory, host, port, config); err != nil {
				log.Error().Err(err).Msg("Error running API server")
			}
		}()
	} else {
		log.Info().Msg("API server is disabled via configuration")
	}

	// Determine whether to start the Informer based on configuration priority
	// The default is to run Informer unless explicitly disabled by configuration
	// 1. New explicit setting (informer.enabled: false) takes highest priority
	// 2. Legacy setting (kubernetes.disable_informer: true) can also disable it
	log.Debug().Bool("informer_enabled_config", config.Informer.Enabled).Bool("kubernetes_disable_informer", config.Kubernetes.DisableInformer).Msg("Informer controller configuration")

	// Only don't start if explicitly disabled by new or legacy config
	if !(config.Informer.Enabled == false || config.Kubernetes.DisableInformer) {
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
