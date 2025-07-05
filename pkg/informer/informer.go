package informer

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

// CreateClientset creates a Kubernetes clientset from kubeconfig or in-cluster config
func CreateClientset(kubeconfig string, inCluster bool, opts *InformerOptions) (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error

	if inCluster {
		log.Info().Msg("Using in-cluster configuration")
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		log.Info().Str("kubeconfig", kubeconfig).Msg("Using kubeconfig file")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	// Apply client configuration from options
	if opts != nil {
		config.QPS = opts.QPS
		config.Burst = opts.Burst
		config.Timeout = opts.Timeout
	}

	return kubernetes.NewForConfig(config)
}

// StartDeploymentInformer starts a shared informer for Deployments with configuration options
func StartDeploymentInformer(ctx context.Context, clientset *kubernetes.Clientset, opts *InformerOptions) error {
	if opts == nil {
		// Use defaults if no options provided
		opts = DefaultInformerOptions()
	}

	// Skip if informer is disabled in config
	if opts.DisableInformer {
		log.Info().Msg("Deployment informer is disabled via configuration")
		return nil
	}

	// Prepare informer factory options
	factoryOptions := []informers.SharedInformerOption{
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			// Apply field selector if provided
			if opts.FieldSelector != "" {
				options.FieldSelector = opts.FieldSelector
			} else {
				options.FieldSelector = fields.Everything().String()
			}

			// Apply label selector if provided
			if opts.LabelSelector != "" {
				options.LabelSelector = opts.LabelSelector
			}
		}),
	}

	// Set namespace if specified
	if opts.Namespace != "" {
		factoryOptions = append(factoryOptions, informers.WithNamespace(opts.Namespace))
	}

	// Create factory with options
	factory := informers.NewSharedInformerFactoryWithOptions(
		clientset,
		opts.ResyncPeriod,
		factoryOptions...,
	)

	// Get informer
	informer := factory.Apps().V1().Deployments().Informer()

	// Add event handlers
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			deployment, ok := obj.(*appsv1.Deployment)
			if !ok {
				log.Warn().Msg("Received non-deployment object in add event")
				return
			}

			// Custom add event handling logic
			processDeploymentAdd(deployment)

			// Log if enabled
			if opts.EnableEventLogging {
				log.Info().Str("name", deployment.Name).Str("namespace", deployment.Namespace).Msg("Deployment added")
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldDeployment, ok1 := oldObj.(*appsv1.Deployment)
			newDeployment, ok2 := newObj.(*appsv1.Deployment)
			if !ok1 || !ok2 {
				log.Warn().Msg("Received non-deployment object in update event")
				return
			}

			// Skip processing if resourceVersion hasn't changed
			if oldDeployment.ResourceVersion == newDeployment.ResourceVersion {
				return
			}

			// Custom update event handling logic
			processDeploymentUpdate(oldDeployment, newDeployment)

			// Log if enabled
			if opts.EnableEventLogging {
				log.Info().Str("name", newDeployment.Name).Str("namespace", newDeployment.Namespace).Msg("Deployment updated")
			}
		},
		DeleteFunc: func(obj interface{}) {
			// When a delete is observed, obj could be a DeletedFinalStateUnknown object
			deployment, ok := obj.(*appsv1.Deployment)
			if !ok {
				// Try to recover the deployment from DeletedFinalStateUnknown
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					log.Warn().Msg("Received non-deployment object in delete event")
					return
				}
				deployment, ok = tombstone.Obj.(*appsv1.Deployment)
				if !ok {
					log.Warn().Str("type", tombstone.Key).Msg("Tombstone contained non-deployment object")
					return
				}
			}

			// Custom delete event handling logic
			processDeploymentDelete(deployment)

			// Log if enabled
			if opts.EnableEventLogging {
				log.Info().Str("name", deployment.Name).Str("namespace", deployment.Namespace).Msg("Deployment deleted")
			}
		},
	})

	// Start informer and wait for cache sync
	log.Info().Msg("Starting deployment informer...")
	factory.Start(ctx.Done())

	// Wait for cache sync with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	if !cache.WaitForCacheSync(timeoutCtx.Done(), informer.HasSynced) {
		return fmt.Errorf("timed out waiting for deployment informer cache to sync")
	}

	log.Info().Msg("Deployment informer cache synced. Watching for events...")
	<-ctx.Done() // Block until context is cancelled
	return nil
}

// Custom event processing functions
func processDeploymentAdd(deployment *appsv1.Deployment) {
	// Custom logic for deployment add events
	log.Debug().
		Str("name", deployment.Name).
		Str("namespace", deployment.Namespace).
		Int32("replicas", *deployment.Spec.Replicas).
		Msg("Processing add event")

	// Example: Check for specific annotations
	if val, exists := deployment.Annotations["custom-controller/monitored"]; exists && val == "true" {
		log.Info().Str("name", deployment.Name).Msg("Found monitored deployment")
	}
}

func processDeploymentUpdate(oldDeployment, newDeployment *appsv1.Deployment) {
	// Custom logic for deployment update events
	log.Debug().
		Str("name", newDeployment.Name).
		Str("namespace", newDeployment.Namespace).
		Str("oldVersion", oldDeployment.ResourceVersion).
		Str("newVersion", newDeployment.ResourceVersion).
		Msg("Processing update event")

	// Example: Detect scaling events
	if *oldDeployment.Spec.Replicas != *newDeployment.Spec.Replicas {
		log.Info().
			Str("name", newDeployment.Name).
			Int32("oldReplicas", *oldDeployment.Spec.Replicas).
			Int32("newReplicas", *newDeployment.Spec.Replicas).
			Msg("Deployment scaled")
	}

	// Example: Detect image changes
	oldImage := getContainerImage(oldDeployment)
	newImage := getContainerImage(newDeployment)
	if oldImage != newImage {
		log.Info().
			Str("name", newDeployment.Name).
			Str("oldImage", oldImage).
			Str("newImage", newImage).
			Msg("Deployment image updated")
	}
}

func processDeploymentDelete(deployment *appsv1.Deployment) {
	// Custom logic for deployment delete events
	log.Debug().
		Str("name", deployment.Name).
		Str("namespace", deployment.Namespace).
		Msg("Processing delete event")

	// Example: Check if this was a monitored deployment
	if val, exists := deployment.Annotations["custom-controller/monitored"]; exists && val == "true" {
		log.Info().Str("name", deployment.Name).Msg("Monitored deployment was deleted")
	}
}

// Helper function to get the first container image from a deployment
func getContainerImage(deployment *appsv1.Deployment) string {
	if deployment == nil || len(deployment.Spec.Template.Spec.Containers) == 0 {
		return "<unknown>"
	}
	return deployment.Spec.Template.Spec.Containers[0].Image
}

// FindDeploymentInCache searches for a deployment in the informer's cache by name and namespace
func FindDeploymentInCache(informer cache.SharedIndexInformer, namespace, name string) (*appsv1.Deployment, error) {
	key := namespace + "/" + name
	item, exists, err := informer.GetIndexer().GetByKey(key)
	if err != nil {
		return nil, fmt.Errorf("error finding deployment %s in cache: %w", key, err)
	}
	if !exists {
		return nil, fmt.Errorf("deployment %s not found in cache", key)
	}

	deployment, ok := item.(*appsv1.Deployment)
	if !ok {
		return nil, fmt.Errorf("found object is not a deployment")
	}

	return deployment, nil
}

// FindDeploymentsByLabelInCache searches for deployments in the informer's cache by label selector
func FindDeploymentsByLabelInCache(informer cache.SharedIndexInformer, labelSelector string) ([]*appsv1.Deployment, error) {
	selector, err := labels.Parse(labelSelector)
	if err != nil {
		return nil, fmt.Errorf("invalid label selector %s: %w", labelSelector, err)
	}

	var deployments []*appsv1.Deployment
	for _, obj := range informer.GetIndexer().List() {
		deployment, ok := obj.(*appsv1.Deployment)
		if !ok {
			continue
		}

		if selector.Matches(labels.Set(deployment.Labels)) {
			deployments = append(deployments, deployment)
		}
	}

	return deployments, nil
}

// ListDeploymentsInCache returns all deployments in the informer's cache
func ListDeploymentsInCache(informer cache.SharedIndexInformer, namespace string) ([]*appsv1.Deployment, error) {
	var deployments []*appsv1.Deployment

	for _, obj := range informer.GetIndexer().List() {
		deployment, ok := obj.(*appsv1.Deployment)
		if !ok {
			continue
		}

		// Filter by namespace if provided
		if namespace != "" && deployment.Namespace != namespace {
			continue
		}

		deployments = append(deployments, deployment)
	}

	return deployments, nil
}

// Utility function for getting deployment name from any object
func getDeploymentName(obj any) string {
	if d, ok := obj.(metav1.Object); ok {
		return d.GetName()
	}
	return "unknown"
}
