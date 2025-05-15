package k8sutils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake" // Import fake clientset
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// K8sAuthChecker defines the interface for Kubernetes authentication and permission checks.
// This should match the original interface definition.
type K8sAuthChecker interface {
	GetKubeConfig() (*rest.Config, error)
	GetClientset() (kubernetes.Interface, error)
	IsRunningInCluster() bool
	GetCurrentNamespace() (string, error)
	CheckNamespacePermissions(ctx context.Context, namespace string, resource schema.GroupVersionResource, verbs []string) (map[string]bool, error)
	CanPerformClusterAction(ctx context.Context, resource schema.GroupVersionResource, verb string) (bool, error)
}

// AuthUtil is the mock implementation of K8sAuthChecker.
// It replaces the original AuthUtil for testing environments.
type AuthUtil struct {
	// Fields to mimic the original AuthUtil, allowing tests to set them up
	clientset kubernetes.Interface
	config    *rest.Config
	inCluster bool

	// Customizable function fields for fine-grained mocking
	GetKubeConfigFunc             func() (*rest.Config, error)
	GetClientsetFunc              func() (kubernetes.Interface, error)
	IsRunningInClusterFunc        func() bool
	GetCurrentNamespaceFunc       func() (string, error)
	CheckNamespacePermissionsFunc func(ctx context.Context, namespace string, resource schema.GroupVersionResource, verbs []string) (map[string]bool, error)
	CanPerformClusterActionFunc   func(ctx context.Context, resource schema.GroupVersionResource, verb string) (bool, error)
}

// NewAuthUtil is a mock constructor that returns an *AuthUtil instance.
// This matches the original NewAuthUtil signature if it returned a concrete type.
func NewAuthUtil() (K8sAuthChecker, error) {
	// Return a new AuthUtil, which now implements K8sAuthChecker
	// Tests can then further customize the func fields or the internal clientset/config if needed.
	return &AuthUtil{
		// Provide default mock values that are generally safe for tests
		config:    &rest.Config{Host: "mock-kube-api-server"}, // A non-nil config
		clientset: fake.NewSimpleClientset(),                  // A default fake clientset
		inCluster: false,                                      // Default to out-of-cluster
	}, nil
}

// GetKubeConfig mocks the GetKubeConfig method.
func (u *AuthUtil) GetKubeConfig() (*rest.Config, error) {
	if u.GetKubeConfigFunc != nil {
		return u.GetKubeConfigFunc()
	}
	if u.config == nil {
		return &rest.Config{Host: "mock-host-default"}, nil
	}
	return u.config, nil
}

// GetClientset mocks the GetClientset method.
func (u *AuthUtil) GetClientset() (kubernetes.Interface, error) {
	if u.GetClientsetFunc != nil {
		return u.GetClientsetFunc()
	}
	if u.clientset == nil {
		return fake.NewSimpleClientset(), nil
	}
	return u.clientset, nil
}

// IsRunningInCluster mocks the IsRunningInCluster method.
func (u *AuthUtil) IsRunningInCluster() bool {
	if u.IsRunningInClusterFunc != nil {
		return u.IsRunningInClusterFunc()
	}
	return u.inCluster
}

// GetCurrentNamespace mocks the GetCurrentNamespace method.
func (u *AuthUtil) GetCurrentNamespace() (string, error) {
	if u.GetCurrentNamespaceFunc != nil {
		return u.GetCurrentNamespaceFunc()
	}

	if u.IsRunningInCluster() {
		if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
			return "kube-system", nil
		}
	}

	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "default", fmt.Errorf("mock: failed to get user home directory: %w", err)
		}
		kubeconfigPath = filepath.Join(home, ".kube", "config")
	}

	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return "default", nil
	}

	configAccess := clientcmd.NewDefaultClientConfigLoadingRules()
	configAccess.ExplicitPath = kubeconfigPath

	kcfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		configAccess,
		&clientcmd.ConfigOverrides{},
	).RawConfig()

	if err != nil {
		return "default", fmt.Errorf("mock: error loading kubeconfig for namespace: %w, falling back to \"default\"", err)
	}

	if kcfg.CurrentContext == "" || kcfg.Contexts[kcfg.CurrentContext] == nil {
		return "default", nil
	}

	namespace := kcfg.Contexts[kcfg.CurrentContext].Namespace
	if namespace == "" {
		return "default", nil
	}
	return namespace, nil
}

// CheckNamespacePermissions mocks the CheckNamespacePermissions method.
func (u *AuthUtil) CheckNamespacePermissions(ctx context.Context, namespace string, resourceGV schema.GroupVersionResource, verbs []string) (map[string]bool, error) {
	if u.CheckNamespacePermissionsFunc != nil {
		return u.CheckNamespacePermissionsFunc(ctx, namespace, resourceGV, verbs)
	}

	cs, err := u.GetClientset()
	if err != nil {
		return nil, fmt.Errorf("mock AuthUtil: failed to get clientset for CheckNamespacePermissions: %w", err)
	}
	if cs == nil {
		return nil, fmt.Errorf("mock AuthUtil: clientset is nil in CheckNamespacePermissions")
	}

	results := make(map[string]bool)
	for _, verb := range verbs {
		sar := &authorizationv1.SelfSubjectAccessReview{
			Spec: authorizationv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Namespace: namespace,
					Verb:      verb,
					Group:     resourceGV.Group,
					Version:   resourceGV.Version,
					Resource:  resourceGV.Resource,
				},
			},
		}
		response, errAuth := cs.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
		if errAuth != nil {
			results[verb] = false
		} else {
			results[verb] = response.Status.Allowed
		}
	}
	return results, nil
}

// CanPerformClusterAction mocks the CanPerformClusterAction method.
func (u *AuthUtil) CanPerformClusterAction(ctx context.Context, resourceGV schema.GroupVersionResource, verb string) (bool, error) {
	if u.CanPerformClusterActionFunc != nil {
		return u.CanPerformClusterActionFunc(ctx, resourceGV, verb)
	}

	cs, err := u.GetClientset()
	if err != nil {
		return false, fmt.Errorf("mock AuthUtil: failed to get clientset for CanPerformClusterAction: %w", err)
	}
	if cs == nil {
		return false, fmt.Errorf("mock AuthUtil: clientset is nil in CanPerformClusterAction")
	}

	sar := &authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Verb:     verb,
				Group:    resourceGV.Group,
				Version:  resourceGV.Version,
				Resource: resourceGV.Resource,
			},
		},
	}
	response, errAuth := cs.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
	if errAuth != nil {
		return false, fmt.Errorf("mock: failed to perform SelfSubjectAccessReview for verb 	%s on cluster resource 	%s: %w", verb, resourceGV.Resource, errAuth)
	}
	return response.Status.Allowed, nil
}

// Ensure AuthUtil implements K8sAuthChecker
var _ K8sAuthChecker = &AuthUtil{}

// Global resource variables, matching the original package
var (
	ResourcePods         = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	ResourceServices     = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
	ResourceConfigMaps   = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	ResourceSecrets      = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
	ResourceNamespaces   = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}
	ResourceDeployments  = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	ResourceStatefulSets = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}
	ResourceDaemonSets   = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}
)

// DefaultCRUDVerbs, matching the original package
var DefaultCRUDVerbs = []string{"get", "list", "watch", "create", "update", "patch", "delete"}
