package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// TrySetupClientset attempts to create a Kubernetes clientset using envtest.
// It returns the clientset or an error if the setup fails.
func TrySetupClientset() (*kubernetes.Clientset, error) {
	env := &envtest.Environment{}

	cfg, err := env.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start envtest: %w", err)
	}

	// Write kubeconfig to /tmp/envtest.kubeconfig
	kubeconfig := clientcmdapi.NewConfig()
	kubeconfig.Clusters["envtest"] = &clientcmdapi.Cluster{
		Server:                   cfg.Host,
		CertificateAuthorityData: cfg.CAData,
	}
	kubeconfig.AuthInfos["envtest-user"] = &clientcmdapi.AuthInfo{
		ClientCertificateData: cfg.CertData,
		ClientKeyData:         cfg.KeyData,
	}
	kubeconfig.Contexts["envtest-context"] = &clientcmdapi.Context{
		Cluster:  "envtest",
		AuthInfo: "envtest-user",
	}
	kubeconfig.CurrentContext = "envtest-context"

	kubeconfigBytes, err := clientcmd.Write(*kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to write kubeconfig: %w", err)
	}

	err = os.WriteFile("/tmp/envtest.kubeconfig", kubeconfigBytes, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write kubeconfig file: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return clientset, nil
}

// CreateTestDeployments creates sample deployments in the given clientset
func CreateTestDeployments(t *testing.T, clientset *kubernetes.Clientset) {
	t.Helper()
	ctx := context.Background()

	// Create sample Deployments
	for i := 1; i <= 2; i++ {
		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("sample-deployment-%d", i),
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "test"},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test"}},
					Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "nginx", Image: "nginx"}}},
				},
			},
		}
		_, err := clientset.AppsV1().Deployments("default").Create(ctx, dep, metav1.CreateOptions{})
		if err != nil {
			t.Logf("Failed to create deployment %s: %v", dep.Name, err)
		}
	}
}

func int32Ptr(i int32) *int32 { return &i }

// CreateTempKubeconfig creates a temporary kubeconfig file for testing and returns its path.
// This is useful for testing functions that need to read a kubeconfig file.
func CreateTempKubeconfig() (string, error) {
	// Create a minimal kubeconfig structure
	kubeconfig := clientcmdapi.NewConfig()
	kubeconfig.Clusters["test-cluster"] = &clientcmdapi.Cluster{
		Server: "https://localhost:8443",
	}
	kubeconfig.AuthInfos["test-user"] = &clientcmdapi.AuthInfo{}
	kubeconfig.Contexts["test-context"] = &clientcmdapi.Context{
		Cluster:  "test-cluster",
		AuthInfo: "test-user",
	}
	kubeconfig.CurrentContext = "test-context"

	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "kubeconfig-*.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	filePath := tmpfile.Name()

	// Write the kubeconfig to the file
	kubeconfigBytes, err := clientcmd.Write(*kubeconfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal kubeconfig: %w", err)
	}

	if _, err := tmpfile.Write(kubeconfigBytes); err != nil {
		return "", fmt.Errorf("failed to write to temp file: %w", err)
	}

	if err := tmpfile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	return filePath, nil
}
