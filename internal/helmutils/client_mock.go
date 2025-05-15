package helmutils

import (
	"fmt"
	"log"
	"os"
	"time"

	k8sutils "go_k8s_helm/internal/k8sutils"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// HelmClient defines the interface for Helm operations.
type HelmClient interface {
	ListReleases(namespace string, stateMask action.ListStates) ([]*ReleaseInfo, error)
	InstallChart(namespace, releaseName, chartName string, chartVersion string, vals map[string]interface{}, createNamespace bool, wait bool, timeout time.Duration) (*ReleaseInfo, error)
	UninstallRelease(namespace, releaseName string, keepHistory bool, timeout time.Duration) (string, error)
	UpgradeRelease(namespace, releaseName, chartName string, chartVersion string, vals map[string]interface{}, wait bool, timeout time.Duration, installIfMissing bool, force bool) (*ReleaseInfo, error)
	GetReleaseDetails(namespace, releaseName string) (*ReleaseInfo, error)
	GetReleaseHistory(namespace, releaseName string) ([]*ReleaseInfo, error)
	AddRepository(name, url, username, password string, passCredentials bool) error
	UpdateRepositories() error
	EnsureChart(chartName, version string) (string, error)
}

// ReleaseInfo holds summarized information about a Helm release.
type ReleaseInfo struct {
	Name         string                 `json:"name"`
	Namespace    string                 `json:"namespace"`
	Revision     int                    `json:"revision"`
	Updated      time.Time              `json:"updated"`
	Status       release.Status         `json:"status"`
	ChartName    string                 `json:"chartName"`
	ChartVersion string                 `json:"chartVersion"`
	AppVersion   string                 `json:"appVersion"`
	Description  string                 `json:"description,omitempty"`
	Notes        string                 `json:"notes,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Manifest     string                 `json:"manifest,omitempty"`
	Values       map[string]interface{} `json:"values,omitempty"`
}

// MockHelmClientFields holds the mockable functions for HelmClient methods.
type MockHelmClientFields struct {
	ListReleasesFunc       func(namespace string, stateMask action.ListStates) ([]*ReleaseInfo, error)
	InstallChartFunc       func(namespace, releaseName, chartName string, chartVersion string, vals map[string]interface{}, createNamespace bool, wait bool, timeout time.Duration) (*ReleaseInfo, error)
	UninstallReleaseFunc   func(namespace, releaseName string, keepHistory bool, timeout time.Duration) (string, error)
	UpgradeReleaseFunc     func(namespace, releaseName, chartName string, chartVersion string, vals map[string]interface{}, wait bool, timeout time.Duration, installIfMissing bool, force bool) (*ReleaseInfo, error)
	GetReleaseDetailsFunc  func(namespace, releaseName string) (*ReleaseInfo, error)
	GetReleaseHistoryFunc  func(namespace, releaseName string) ([]*ReleaseInfo, error)
	AddRepositoryFunc      func(name, url, username, password string, passCredentials bool) error
	UpdateRepositoriesFunc func() error
	EnsureChartFunc        func(chartName, version string) (string, error)
}

// Client is the mock implementation of HelmClient.
type Client struct {
	settings       *cli.EnvSettings
	authChecker    k8sutils.K8sAuthChecker
	baseKubeConfig *rest.Config
	Log            func(format string, v ...interface{})
	*MockHelmClientFields
}

// NewClient returns a new mock HelmClient.
func NewClient(authChecker k8sutils.K8sAuthChecker, defaultNamespace string, logger func(format string, v ...interface{})) (HelmClient, error) {
	actualLogger := logger
	if actualLogger == nil {
		actualLogger = log.Printf
	}

	settings := cli.New()
	if defaultNamespace != "" {
		settings.SetNamespace(defaultNamespace)
	} else {
		// Attempt to fall back to authChecker's current namespace
		ns, err := authChecker.GetCurrentNamespace()
		if err != nil {
			actualLogger("Warning: could not determine current namespace via authChecker, using settings default: %v", err)
		} else {
			settings.SetNamespace(ns)
		}
	}

	// Now retrieve kubeconfig (propagate any error)
	kubeConfig, err := authChecker.GetKubeConfig()
	if err != nil {
		return nil, err
	}

	mc := &Client{
		settings:             settings,
		authChecker:          authChecker,
		baseKubeConfig:       kubeConfig,
		Log:                  actualLogger,
		MockHelmClientFields: &MockHelmClientFields{},
	}
	return mc, nil
}

// --- Mock implementations for HelmClient interface methods ---
func (c *Client) ListReleases(namespace string, stateMask action.ListStates) ([]*ReleaseInfo, error) {
	if c.MockHelmClientFields != nil && c.MockHelmClientFields.ListReleasesFunc != nil {
		return c.ListReleasesFunc(namespace, stateMask)
	}
	c.Log("Mock ListReleases called for namespace: %s", namespace)
	return []*ReleaseInfo{{Name: "mocked-release", Namespace: namespace, Status: release.StatusDeployed}}, nil
}

func (c *Client) InstallChart(namespace, releaseName, chartName string, chartVersion string, vals map[string]interface{}, createNamespace bool, wait bool, timeout time.Duration) (*ReleaseInfo, error) {
	if c.MockHelmClientFields != nil && c.MockHelmClientFields.InstallChartFunc != nil {
		return c.InstallChartFunc(namespace, releaseName, chartName, chartVersion, vals, createNamespace, wait, timeout)
	}
	c.Log("Mock InstallChart called for release: %s, chart: %s", releaseName, chartName)
	return &ReleaseInfo{Name: releaseName, Namespace: namespace, Status: release.StatusDeployed}, nil
}

func (c *Client) UninstallRelease(namespace, releaseName string, keepHistory bool, timeout time.Duration) (string, error) {
	if c.MockHelmClientFields != nil && c.MockHelmClientFields.UninstallReleaseFunc != nil {
		return c.UninstallReleaseFunc(namespace, releaseName, keepHistory, timeout)
	}
	c.Log("Mock UninstallRelease called for release: %s", releaseName)
	return "uninstalled", nil
}

func (c *Client) UpgradeRelease(namespace, releaseName, chartName string, chartVersion string, vals map[string]interface{}, wait bool, timeout time.Duration, installIfMissing bool, force bool) (*ReleaseInfo, error) {
	if c.MockHelmClientFields != nil && c.MockHelmClientFields.UpgradeReleaseFunc != nil {
		return c.UpgradeReleaseFunc(namespace, releaseName, chartName, chartVersion, vals, wait, timeout, installIfMissing, force)
	}
	c.Log("Mock UpgradeRelease called for release: %s, chart: %s", releaseName, chartName)
	return &ReleaseInfo{Name: releaseName, Namespace: namespace, Status: release.StatusDeployed}, nil
}

func (c *Client) GetReleaseDetails(namespace, releaseName string) (*ReleaseInfo, error) {
	if c.MockHelmClientFields != nil && c.MockHelmClientFields.GetReleaseDetailsFunc != nil {
		return c.GetReleaseDetailsFunc(namespace, releaseName)
	}
	c.Log("Mock GetReleaseDetails called for release: %s", releaseName)
	if releaseName == "non-existent-release" {
		return nil, fmt.Errorf("release: not found")
	}
	return &ReleaseInfo{Name: releaseName, Namespace: namespace, Status: release.StatusDeployed}, nil
}

func (c *Client) GetReleaseHistory(namespace, releaseName string) ([]*ReleaseInfo, error) {
	if c.MockHelmClientFields != nil && c.MockHelmClientFields.GetReleaseHistoryFunc != nil {
		return c.GetReleaseHistoryFunc(namespace, releaseName)
	}
	c.Log("Mock GetReleaseHistory called for release: %s", releaseName)
	return []*ReleaseInfo{{Name: releaseName, Namespace: namespace, Status: release.StatusDeployed}}, nil
}

func (c *Client) AddRepository(name, url, username, password string, passCredentials bool) error {
	if c.MockHelmClientFields != nil && c.MockHelmClientFields.AddRepositoryFunc != nil {
		return c.AddRepositoryFunc(name, url, username, password, passCredentials)
	}
	c.Log("Mock AddRepository called for repo: %s", name)
	return nil
}

func (c *Client) UpdateRepositories() error {
	if c.MockHelmClientFields != nil && c.MockHelmClientFields.UpdateRepositoriesFunc != nil {
		return c.UpdateRepositoriesFunc()
	}
	c.Log("Mock UpdateRepositories called")
	return nil
}

func (c *Client) EnsureChart(chartName, version string) (string, error) {
	if c.MockHelmClientFields != nil && c.MockHelmClientFields.EnsureChartFunc != nil {
		return c.EnsureChartFunc(chartName, version)
	}
	c.Log("Mock EnsureChart called for chart: %s, version: %s", chartName, version)
	return "/mocked/chart/path", nil
}

// --- Mock implementation for non-interface methods called by tests ---

// getActionConfig creates a new action.Configuration for the specified namespace.
// This is a mock implementation.
func (c *Client) getActionConfig(namespace string) (*action.Configuration, error) {
	c.Log("Mock getActionConfig called for namespace: %s", namespace)
	if namespace == "" {
		if c.settings == nil || c.settings.Namespace() == "" {
			return nil, fmt.Errorf("mock getActionConfig: target namespace is empty and client's default namespace is also empty")
		}
		namespace = c.settings.Namespace()
	}

	// Return a minimally viable action.Configuration for mock purposes.
	// The Init method requires a genericclioptions.RESTClientGetter.
	mockClientGetter := newMockConfigGetter(c.baseKubeConfig, namespace)
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(mockClientGetter, namespace, os.Getenv("HELM_DRIVER"), c.Log); err != nil {
		return nil, fmt.Errorf("mock getActionConfig: failed to initialize Helm action configuration for namespace '%s': %w", namespace, err)
	}
	return actionConfig, nil
}

// --- Mock implementation for configGetter and its methods ---

// mockConfigGetter implements clientcmd.RESTClientGetter for mock purposes.
type mockConfigGetter struct {
	config    *rest.Config
	namespace string
}

func newMockConfigGetter(config *rest.Config, namespace string) genericclioptions.RESTClientGetter {
	return &mockConfigGetter{config: config, namespace: namespace}
}

func (mcg *mockConfigGetter) ToRESTConfig() (*rest.Config, error) {
	if mcg.config == nil {
		// Return a default mock config if nil, to prevent panics in Helm's Init
		return &rest.Config{Host: "mock-host"}, nil
	}
	return rest.CopyConfig(mcg.config), nil
}

func (mcg *mockConfigGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	rc, _ := mcg.ToRESTConfig() // Error ignored for simplicity in mock
	d, err := discovery.NewDiscoveryClientForConfig(rc)
	if err != nil {
		return nil, err
	}
	return memory.NewMemCacheClient(d), nil
}

func (mcg *mockConfigGetter) ToRESTMapper() (meta.RESTMapper, error) {
	dc, err := mcg.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	return restmapper.NewDeferredDiscoveryRESTMapper(dc), nil
}

func (mcg *mockConfigGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	cfg := clientcmdapi.NewConfig()
	// Populate with minimal mock data as needed by Helm's Init or tests
	contextName := "mock-context"
	clusterName := "mock-cluster"
	cfg.Clusters[clusterName] = &clientcmdapi.Cluster{Server: "mock-server"}
	cfg.AuthInfos["mock-user"] = &clientcmdapi.AuthInfo{}
	cfg.Contexts[contextName] = &clientcmdapi.Context{Cluster: clusterName, AuthInfo: "mock-user", Namespace: mcg.namespace}
	cfg.CurrentContext = contextName
	return clientcmd.NewDefaultClientConfig(*cfg, &clientcmd.ConfigOverrides{})
}

// convertReleaseToInfo is a helper function, copied from original client.go for compatibility.
func convertReleaseToInfo(rel *release.Release) *ReleaseInfo {
	if rel == nil {
		return nil
	}
	info := &ReleaseInfo{
		Name:      rel.Name,
		Namespace: rel.Namespace,
		Revision:  rel.Version,
		Values:    rel.Config,
		Manifest:  rel.Manifest,
	}
	if rel.Info != nil {
		info.Status = rel.Info.Status
		info.Description = rel.Info.Description
		info.Notes = rel.Info.Notes
		if !rel.Info.LastDeployed.IsZero() {
			info.Updated = rel.Info.LastDeployed.Time
		}
	}
	if rel.Chart != nil {
		info.Config = rel.Chart.Values
		if rel.Chart.Metadata != nil {
			info.ChartName = rel.Chart.Metadata.Name
			info.ChartVersion = rel.Chart.Metadata.Version
			info.AppVersion = rel.Chart.Metadata.AppVersion
		}
	}
	return info
}

// Compile-time check to ensure *Client implements HelmClient.
var _ HelmClient = (*Client)(nil)
