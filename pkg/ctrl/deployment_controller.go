// Package ctrl contains Kubernetes controller implementations
package ctrl

import (
	context "context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// DeploymentReconciler handles basic deployment reconciliation
type DeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// DeploymentController watches Kubernetes deployments and logs events
type DeploymentController struct {
	client    client.Client
	clientset *kubernetes.Clientset
	clusterID string
}

// ClusterConfig holds configuration for a Kubernetes cluster
type ClusterConfig struct {
	Name        string
	KubeConfig  string // Path to kubeconfig file
	Context     string // Context in the kubeconfig file
	InCluster   bool   // Use in-cluster config
	Namespace   string // Namespace to watch (empty for all)
	ClusterID   string // Unique ID for this cluster
	APIEndpoint string // API server endpoint
	
	// Leader election settings
	LeaderElection struct {
		Enabled   bool   // Enable leader election
		Namespace string // Namespace for leader election resources
		ID        string // Unique ID for leader election
	}
	
	// Metrics settings
	MetricsBindAddress string // Address for metrics server, empty to disable
}

// MultiClusterManager manages controllers for multiple Kubernetes clusters
type MultiClusterManager struct {
	managers    map[string]manager.Manager
	configs     map[string]ClusterConfig
	controllers map[string]controller.Controller
}

// Reconcile handles reconciliation of Deployment objects for basic reconciler
func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.Info().Msgf("Reconciling Deployment: %s/%s", req.Namespace, req.Name)
	return ctrl.Result{}, nil
}

// Reconcile handles reconciliation of Deployment objects for the enhanced controller
func (r *DeploymentController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Get the deployment
	var deployment appsv1.Deployment
	if err := r.client.Get(ctx, req.NamespacedName, &deployment); err != nil {
		// We'll ignore not-found errors
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Log that we're reconciling this deployment
	log.Debug().
		Str("cluster_id", r.clusterID).
		Str("namespace", deployment.Namespace).
		Str("name", deployment.Name).
		Msg("Reconciling deployment")

	// No need to do anything else since our goal is just to log events

	return ctrl.Result{}, nil
}

// NewManager creates a new controller manager for a specific cluster
func NewManager(cfg ClusterConfig) (manager.Manager, error) {
	var config *rest.Config
	var err error

	// Get Kubernetes REST config
	if cfg.InCluster {
		// In-cluster configuration
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("error creating in-cluster config: %w", err)
		}
	} else {
		// External cluster configuration
		config, err = clientcmd.BuildConfigFromFlags("", cfg.KubeConfig)
		if err != nil {
			return nil, fmt.Errorf("error building kubeconfig: %w", err)
		}

		// Use specific context if provided
		if cfg.Context != "" {
			loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
			loadingRules.ExplicitPath = cfg.KubeConfig
			context := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
				loadingRules,
				&clientcmd.ConfigOverrides{CurrentContext: cfg.Context},
			)
			clientConfig, err := context.ClientConfig()
			if err != nil {
				return nil, fmt.Errorf("failed to create client config with context %s: %w", cfg.Context, err)
			}
			config = clientConfig
		}
	}

	// Create manager options
	options := ctrl.Options{
		Scheme: Scheme(),
		// Leader election settings
		LeaderElection:          cfg.LeaderElection.Enabled,
		LeaderElectionNamespace: cfg.LeaderElection.Namespace,
		LeaderElectionID:        cfg.LeaderElection.ID,
	}
	
	// Add metrics server if configured
	if cfg.MetricsBindAddress != "" {
		options.Metrics.BindAddress = cfg.MetricsBindAddress
	}

	// Apply namespace filter if specified
	if cfg.Namespace != "" {
		options.Cache.DefaultNamespaces = map[string]cache.Config{
			cfg.Namespace: {},
		}
	}

	// Create manager
	mgr, err := ctrl.NewManager(config, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %w", err)
	}

	return mgr, nil
}

// Scheme creates and returns a new scheme with required types registered
func Scheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)
	return scheme
}

// Basic version for backward compatibility
func AddDeploymentController(mgr manager.Manager) error {
	r := &DeploymentReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}

// AddDeploymentControllerWithLogging adds a deployment controller to the manager with event logging
func AddDeploymentControllerWithLogging(mgr manager.Manager, clusterID string) error {
	// Create clientset from the manager's rest config
	clientset, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}

	// Create controller instance
	r := &DeploymentController{
		client:    mgr.GetClient(),
		clientset: clientset,
		clusterID: clusterID,
	}

	// Define event handlers that will log all events
	eventLogger := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			if deployment, ok := e.Object.(*appsv1.Deployment); ok {
				eventID := uuid.New().String()
				log.Info().
					Str("event_id", eventID).
					Str("cluster_id", r.clusterID).
					Str("event_type", "CREATE").
					Str("resource_type", "Deployment").
					Str("namespace", deployment.Namespace).
					Str("name", deployment.Name).
					Int32("replicas", *deployment.Spec.Replicas).
					Msg("Deployment created")
			}
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			if deployment, ok := e.ObjectNew.(*appsv1.Deployment); ok {
				oldDeployment, _ := e.ObjectOld.(*appsv1.Deployment)
				eventID := uuid.New().String()

				// Check for specific changes
				if oldDeployment != nil && *oldDeployment.Spec.Replicas != *deployment.Spec.Replicas {
					log.Info().
						Str("event_id", eventID).
						Str("cluster_id", r.clusterID).
						Str("event_type", "UPDATE").
						Str("resource_type", "Deployment").
						Str("namespace", deployment.Namespace).
						Str("name", deployment.Name).
						Int32("old_replicas", *oldDeployment.Spec.Replicas).
						Int32("new_replicas", *deployment.Spec.Replicas).
						Msg("Deployment replicas changed")
				} else {
					log.Info().
						Str("event_id", eventID).
						Str("cluster_id", r.clusterID).
						Str("event_type", "UPDATE").
						Str("resource_type", "Deployment").
						Str("namespace", deployment.Namespace).
						Str("name", deployment.Name).
						Int32("replicas", *deployment.Spec.Replicas).
						Msg("Deployment updated")
				}
			}
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			if deployment, ok := e.Object.(*appsv1.Deployment); ok {
				eventID := uuid.New().String()
				log.Info().
					Str("event_id", eventID).
					Str("cluster_id", r.clusterID).
					Str("event_type", "DELETE").
					Str("resource_type", "Deployment").
					Str("namespace", deployment.Namespace).
					Str("name", deployment.Name).
					Msg("Deployment deleted")
			}
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			if deployment, ok := e.Object.(*appsv1.Deployment); ok {
				eventID := uuid.New().String()
				log.Info().
					Str("event_id", eventID).
					Str("cluster_id", r.clusterID).
					Str("event_type", "GENERIC").
					Str("resource_type", "Deployment").
					Str("namespace", deployment.Namespace).
					Str("name", deployment.Name).
					Msg("Generic deployment event received")
			}
			return true
		},
	}

	// Watch deployments with our event logger using builder pattern
	err = ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		WithEventFilter(eventLogger).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to watch deployments: %w", err)
	}

	log.Info().
		Str("cluster_id", clusterID).
		Msg("Deployment controller added to manager")
	return nil
}

// NewMultiClusterManager creates a new manager for multiple Kubernetes clusters
func NewMultiClusterManager() *MultiClusterManager {
	return &MultiClusterManager{
		managers:    make(map[string]manager.Manager),
		configs:     make(map[string]ClusterConfig),
		controllers: make(map[string]controller.Controller),
	}
}

// AddCluster adds a new cluster to be managed
func (m *MultiClusterManager) AddCluster(ctx context.Context, config ClusterConfig) error {
	// Check if cluster with this ID already exists
	if _, exists := m.configs[config.ClusterID]; exists {
		return fmt.Errorf("cluster with ID %s already exists", config.ClusterID)
	}

	// Create manager for this cluster
	mgr, err := NewManager(config)
	if err != nil {
		return fmt.Errorf("failed to create manager for cluster %s: %w", config.ClusterID, err)
	}

	// Add deployment controller with event logging
	err = AddDeploymentControllerWithLogging(mgr, config.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to add deployment controller for cluster %s: %w", config.ClusterID, err)
	}

	// Store manager and config
	m.managers[config.ClusterID] = mgr
	m.configs[config.ClusterID] = config

	log.Info().
		Str("cluster_id", config.ClusterID).
		Str("api_endpoint", config.APIEndpoint).
		Bool("in_cluster", config.InCluster).
		Str("namespace", config.Namespace).
		Msg("Added cluster to multi-cluster manager")

	return nil
}

// RemoveCluster removes a cluster from management
func (m *MultiClusterManager) RemoveCluster(clusterID string) error {
	// Check if cluster exists
	_, exists := m.managers[clusterID]
	if !exists {
		return fmt.Errorf("cluster with ID %s does not exist", clusterID)
	}

	// Clean up resources
	delete(m.managers, clusterID)
	delete(m.configs, clusterID)
	delete(m.controllers, clusterID)

	log.Info().
		Str("cluster_id", clusterID).
		Msg("Removed cluster from multi-cluster manager")

	return nil
}

// StartAll starts all cluster managers
func (m *MultiClusterManager) StartAll(ctx context.Context) error {
	if len(m.managers) == 0 {
		log.Warn().Msg("No cluster managers to start")
		return nil
	}

	// Log the number of clusters we're managing
	log.Info().
		Int("cluster_count", len(m.managers)).
		Msg("Starting multi-cluster manager")

	// Create wait group to track all manager goroutines
	wg := &sync.WaitGroup{}
	errorCh := make(chan error, len(m.managers))
	doneCh := make(chan struct{})

	// Start each manager in its own goroutine
	for clusterID, mgr := range m.managers {
		wg.Add(1)
		go func(id string, manager manager.Manager) {
			defer wg.Done()
			log.Info().Str("cluster_id", id).Msg("Starting manager for cluster")

			if err := manager.Start(ctx); err != nil && ctx.Err() == nil {
				log.Error().
					Str("cluster_id", id).
					Err(err).
					Msg("Manager failed to start")

				// Only report error if it wasn't caused by context cancellation
				errorCh <- fmt.Errorf("failed to start manager for cluster %s: %w", id, err)
				return
			}

			log.Info().Str("cluster_id", id).Msg("Manager stopped")
		}(clusterID, mgr)
	}

	// Wait for all managers to complete in a separate goroutine
	go func() {
		wg.Wait()
		close(doneCh)
		close(errorCh)
	}()

	// Handle two cases:
	// 1. Context canceled - we wait for all managers to stop
	// 2. Error from a manager - we return it immediately
	select {
	case <-ctx.Done():
		log.Info().Msg("Context canceled, waiting for managers to stop")
		<-doneCh // Wait for all managers to stop after context cancellation
		return ctx.Err()

	case err := <-errorCh:
		if err != nil {
			return err
		}
		// Wait for all managers to finish if there was no error
		<-doneCh
		return nil

	case <-doneCh:
		// All managers stopped without errors
		return nil
	}
}

// StopAll gracefully shuts down all cluster managers
func (m *MultiClusterManager) StopAll(ctx context.Context) {
	clusterCount := len(m.managers)
	if clusterCount == 0 {
		log.Debug().Msg("No cluster managers to stop")
		return
	}

	log.Info().Int("cluster_count", clusterCount).Msg("Shutting down all cluster managers")

	// The ctx parameter is expected to be a cancelable context
	// When it's canceled, all managers will gracefully shut down
	// We could wait for the context to be canceled, but that's typically handled by the caller

	// For each cluster, log that we're stopping it
	for clusterID := range m.managers {
		log.Debug().
			Str("cluster_id", clusterID).
			Msg("Stopping manager for cluster")
	}

	// Note: The actual stopping happens when the context is canceled
	// and is handled by the StartAll method
}

// GetClusters returns a list of all configured clusters
func (m *MultiClusterManager) GetClusters() []ClusterConfig {
	configs := make([]ClusterConfig, 0, len(m.configs))
	for _, cfg := range m.configs {
		configs = append(configs, cfg)
	}
	return configs
}

// GetClusterCount returns the number of configured clusters
func (m *MultiClusterManager) GetClusterCount() int {
	return len(m.configs)
}
