package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Common variables
var (
	kubeconfig string
	namespace  string
)

// Variables for create command
var (
	deploymentName string
	image          string
	replicas       int32
	port           int32
)

// getKubeClient creates a Kubernetes clientset from the provided kubeconfig path
func getKubeClient(kubeconfigPath string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

// getKubeClientOrError creates a Kubernetes clientset and logs any error
func getKubeClientOrError(kubeconfigPath string) (*kubernetes.Clientset, error) {
	clientset, err := getKubeClient(kubeconfigPath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create Kubernetes client")
		return nil, err
	}
	return clientset, nil
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List Kubernetes deployments in the specified namespace",
	Run: func(cmd *cobra.Command, args []string) {
		clientset, err := getKubeClientOrError(kubeconfig)
		if err != nil {
			return
		}
		
		// Get deployments
		deployments, err := clientset.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			log.Error().Err(err).Str("namespace", namespace).Msg("Failed to list deployments")
			return
		}
		
		log.Info().Int("count", len(deployments.Items)).Str("namespace", namespace).Msg("Found deployments")
		if len(deployments.Items) == 0 {
			log.Info().Str("namespace", namespace).Msg("No deployments found")
			return
		}
		
		// Print header for table format
		fmt.Println("NAME                    READY   UP-TO-DATE   AVAILABLE   AGE")
		
		// Print each deployment
		for _, d := range deployments.Items {
			ready := fmt.Sprintf("%d/%d", d.Status.ReadyReplicas, d.Status.Replicas)
			upToDate := fmt.Sprintf("%d", d.Status.UpdatedReplicas)
			available := fmt.Sprintf("%d", d.Status.AvailableReplicas)
			age := "<unknown>"
			if !d.CreationTimestamp.IsZero() {
				age = fmt.Sprintf("%s", time.Since(d.CreationTimestamp.Time).Round(time.Second))
			}
			
			// Log detailed info
			log.Debug().
				Str("name", d.Name).
				Str("ready", ready).
				Str("upToDate", upToDate).
				Str("available", available).
				Str("age", age).
				Msg("Deployment details")
			
			// Print in table format for user
			fmt.Printf("%-20s   %-5s   %-10s   %-9s   %s\n", d.Name, ready, upToDate, available, age)
		}
	},
}

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Kubernetes deployment in the specified namespace",
	Run: func(cmd *cobra.Command, args []string) {
		clientset, err := getKubeClientOrError(kubeconfig)
		if err != nil {
			return
		}

		// Create deployment object
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: deploymentName,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": deploymentName,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": deploymentName,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  deploymentName,
								Image: image,
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: port,
									},
								},
							},
						},
					},
				},
			},
		}

		// Create the deployment
		logDeploymentAction("Creating", deploymentName, namespace, image, replicas, port)

		result, err := clientset.AppsV1().Deployments(namespace).Create(context.Background(), deployment, metav1.CreateOptions{})
		if err != nil {
			log.Error().Err(err).Str("name", deploymentName).Str("namespace", namespace).Msg("Failed to create deployment")
			return
		}

		log.Info().
			Str("name", result.GetName()).
			Str("namespace", result.GetNamespace()).
			Msg("Deployment created successfully")
	},
}

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete [deployment-name]",
	Short: "Delete a Kubernetes deployment in the specified namespace",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deploymentToDelete := args[0]
		clientset, err := getKubeClientOrError(kubeconfig)
		if err != nil {
			return
		}

		logDeploymentAction("Deleting", deploymentToDelete, namespace, "", 0, 0)

		// Delete the deployment
		err = clientset.AppsV1().Deployments(namespace).Delete(
			context.Background(),
			deploymentToDelete,
			metav1.DeleteOptions{},
		)
		
		if err != nil {
			log.Error().Err(err).
				Str("name", deploymentToDelete).
				Str("namespace", namespace).
				Msg("Failed to delete deployment")
			return
		}

		log.Info().
			Str("name", deploymentToDelete).
			Str("namespace", namespace).
			Msg("Deployment deleted successfully")
	},
}

// logDeploymentAction logs information about a deployment action
func logDeploymentAction(action, name, ns, img string, replicas, p int32) {
	logEvent := log.Info().Str("name", name).Str("namespace", ns)
	
	if img != "" {
		logEvent = logEvent.Str("image", img)
	}
	
	if replicas > 0 {
		logEvent = logEvent.Int32("replicas", replicas)
	}
	
	if p > 0 {
		logEvent = logEvent.Int32("port", p)
	}
	
	logEvent.Msgf("%s deployment", action)
}

// getDefaultKubeconfig returns the default kubeconfig path
func getDefaultKubeconfig() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Warn().Err(err).Msg("Could not determine user home directory")
		return ""
	}
	
	return fmt.Sprintf("%s/.kube/config", home)
}

func init() {
	defaultKubeconfig := getDefaultKubeconfig()

	// Add commands to root command
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(deleteCmd)

	// Common flags for all Kubernetes commands
	for _, cmd := range []*cobra.Command{listCmd, createCmd, deleteCmd} {
		cmd.Flags().StringVar(&kubeconfig, "kubeconfig", defaultKubeconfig, "Path to the kubeconfig file (default: ~/.kube/config)")
		cmd.Flags().StringVar(&namespace, "namespace", "default", "Kubernetes namespace")
	}

	// Flags specific to create command
	createCmd.Flags().StringVar(&deploymentName, "name", "", "Name for the deployment (required)")
	createCmd.Flags().StringVar(&image, "image", "", "Container image to use (required)")
	createCmd.Flags().Int32Var(&replicas, "replicas", 1, "Number of replicas")
	createCmd.Flags().Int32Var(&port, "port", 80, "Container port")
	
	// Mark required flags
	createCmd.MarkFlagRequired("name")
	createCmd.MarkFlagRequired("image")
}
