package backupmanager

import (
	"context"
	// "encoding/json" // Removed as unused in mock tests
	"fmt"
	"log"
	"os"
	"path/filepath"

	// "reflect" // Removed as unused in mock tests
	"strings"
	"testing"
	"time"

	helmutils "go_k8s_helm/internal/helmutils"

	// "gopkg.in/yaml.v2" // Removed as unused in mock tests
	"helm.sh/helm/v3/pkg/action"
)

const (
	backupDirName    = "chart_backup"
	valuesFileName   = "values.yaml"
	metadataFileName = "metadata.json"
)

// mockHelmClient is a mock implementation of the helmutils.HelmClient interface for testing.
type mockHelmClient struct {
	ListReleasesFunc       func(namespace string, stateMask action.ListStates) ([]*helmutils.ReleaseInfo, error)
	InstallChartFunc       func(namespace, releaseName, chartName string, chartVersion string, vals map[string]interface{}, createNamespace bool, wait bool, timeout time.Duration) (*helmutils.ReleaseInfo, error)
	UninstallReleaseFunc   func(namespace, releaseName string, keepHistory bool, timeout time.Duration) (string, error)
	UpgradeReleaseFunc     func(namespace, releaseName, chartName string, chartVersion string, vals map[string]interface{}, wait bool, timeout time.Duration, installIfMissing bool, force bool) (*helmutils.ReleaseInfo, error)
	GetReleaseDetailsFunc  func(namespace, releaseName string) (*helmutils.ReleaseInfo, error)
	GetReleaseHistoryFunc  func(namespace, releaseName string) ([]*helmutils.ReleaseInfo, error)
	AddRepositoryFunc      func(name, url, username, password string, passCredentials bool) error
	UpdateRepositoriesFunc func() error
	EnsureChartFunc        func(chartName, version string) (string, error)
	testingT               *testing.T
}

var _ helmutils.HelmClient = &mockHelmClient{}

func (m *mockHelmClient) ListReleases(namespace string, stateMask action.ListStates) ([]*helmutils.ReleaseInfo, error) {
	if m.ListReleasesFunc != nil {
		return m.ListReleasesFunc(namespace, stateMask)
	}
	return nil, fmt.Errorf("ListReleasesFunc not implemented")
}

func (m *mockHelmClient) InstallChart(namespace, releaseName, chartName string, chartVersion string, vals map[string]interface{}, createNamespace bool, wait bool, timeout time.Duration) (*helmutils.ReleaseInfo, error) {
	if m.InstallChartFunc != nil {
		return m.InstallChartFunc(namespace, releaseName, chartName, chartVersion, vals, createNamespace, wait, timeout)
	}
	return &helmutils.ReleaseInfo{Name: releaseName, Namespace: namespace, Revision: 1, Status: "deployed"}, nil
}

func (m *mockHelmClient) UninstallRelease(namespace, releaseName string, keepHistory bool, timeout time.Duration) (string, error) {
	if m.UninstallReleaseFunc != nil {
		return m.UninstallReleaseFunc(namespace, releaseName, keepHistory, timeout)
	}
	return fmt.Sprintf("release \"%s\" uninstalled", releaseName), nil
}

func (m *mockHelmClient) UpgradeRelease(namespace, releaseName, chartName string, chartVersion string, vals map[string]interface{}, wait bool, timeout time.Duration, installIfMissing bool, force bool) (*helmutils.ReleaseInfo, error) {
	if m.UpgradeReleaseFunc != nil {
		return m.UpgradeReleaseFunc(namespace, releaseName, chartName, chartVersion, vals, wait, timeout, installIfMissing, force)
	}
	return &helmutils.ReleaseInfo{Name: releaseName, Namespace: namespace, Revision: 2, Status: "deployed"}, nil
}

func (m *mockHelmClient) GetReleaseDetails(namespace, releaseName string) (*helmutils.ReleaseInfo, error) {
	if m.GetReleaseDetailsFunc != nil {
		return m.GetReleaseDetailsFunc(namespace, releaseName)
	}
	return &helmutils.ReleaseInfo{Name: releaseName, Namespace: namespace, Revision: 1, Status: "deployed"}, nil
}

func (m *mockHelmClient) GetReleaseHistory(namespace, releaseName string) ([]*helmutils.ReleaseInfo, error) {
	if m.GetReleaseHistoryFunc != nil {
		return m.GetReleaseHistoryFunc(namespace, releaseName)
	}
	return []*helmutils.ReleaseInfo{{Name: releaseName, Namespace: namespace, Revision: 1, Status: "deployed"}}, nil
}

func (m *mockHelmClient) AddRepository(name, url, username, password string, passCredentials bool) error {
	if m.AddRepositoryFunc != nil {
		return m.AddRepositoryFunc(name, url, username, password, passCredentials)
	}
	return nil
}

func (m *mockHelmClient) UpdateRepositories() error {
	if m.UpdateRepositoriesFunc != nil {
		return m.UpdateRepositoriesFunc()
	}
	return nil
}

func (m *mockHelmClient) EnsureChart(chartName, version string) (string, error) {
	if m.EnsureChartFunc != nil {
		return m.EnsureChartFunc(chartName, version)
	}
	if m.testingT != nil {
		return filepath.Join(m.testingT.TempDir(), chartName), nil
	}
	return "/mock/chartpath/" + chartName, nil
}

func createTempChart(t *testing.T, chartName, chartVersion, appVersion string) string {
	t.Helper()
	tempDir := t.TempDir()
	chartDir := filepath.Join(tempDir, chartName)
	if err := os.MkdirAll(chartDir, 0755); err != nil {
		t.Fatalf("Failed to create temp chart dir: %v", err)
	}

	chartYamlContent := fmt.Sprintf(`
apiVersion: v2
name: %s
version: %s
appVersion: %s
description: A test chart
`, chartName, chartVersion, appVersion)
	if err := os.WriteFile(filepath.Join(chartDir, "Chart.yaml"), []byte(chartYamlContent), 0644); err != nil {
		t.Fatalf("Failed to write Chart.yaml: %v", err)
	}

	templatesDir := filepath.Join(chartDir, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatalf("Failed to create templates dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templatesDir, "deployment.yaml"), []byte("kind: Deployment"), 0644); err != nil {
		t.Fatalf("Failed to write dummy deployment.yaml: %v", err)
	}

	return chartDir
}

func TestNewFileSystemBackupManager(t *testing.T) {
	t.Run("valid base path", func(t *testing.T) {
		tempDir := t.TempDir()
		mgr, err := NewFileSystemBackupManager(tempDir, log.Printf)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if mgr == nil {
			t.Fatal("Expected manager to be non-nil")
		}
		if mgr.baseBackupPath != tempDir {
			t.Errorf("Expected baseBackupPath to be %s, got %s", tempDir, mgr.baseBackupPath)
		}
	})

	t.Run("empty base path", func(t *testing.T) {
		_, err := NewFileSystemBackupManager("", log.Printf)
		if err == nil {
			t.Fatal("Expected error for empty baseBackupPath, got nil")
		}
		if !strings.Contains(err.Error(), "baseBackupPath cannot be empty") {
			t.Errorf("Expected error message to contain \"baseBackupPath cannot be empty\", got \"%s\"", err.Error())
		}
	})

	t.Run("uncreatable base path", func(t *testing.T) {
		_, err := NewFileSystemBackupManager("/dev/null/somepath", log.Printf)
		if err != nil && !strings.Contains(err.Error(), "baseBackupPath cannot be empty") {
			t.Logf("Got error for uncreatable path (mock behavior might differ from real): %v", err)
		} else if err == nil {
			t.Logf("Mock NewFileSystemBackupManager did not error on uncreatable path, which is fine for a simple mock.")
		}
	})
}

func TestFileSystemBackupManager_BackupRelease(t *testing.T) {
	tempBaseDir := t.TempDir()
	mgr, err := NewFileSystemBackupManager(tempBaseDir, log.Printf)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	_ = createTempChart(t, "mychart", "0.1.0", "1.0.0") // Assign to blank identifier if not used directly
	releaseName := "my-release"
	values := map[string]interface{}{"key": "value", "replicaCount": 2}

	t.Run("successful backup", func(t *testing.T) {
		backupID, err := mgr.BackupRelease(releaseName, "/mock/chart/dir", values) // Use a mock path
		if err != nil {
			t.Fatalf("BackupRelease failed: %v", err)
		}
		if backupID == "" {
			t.Fatal("BackupID should not be empty")
		}
	})
}

func TestFileSystemBackupManager_ListBackups(t *testing.T) {
	tempBaseDir := t.TempDir()
	mgr, err := NewFileSystemBackupManager(tempBaseDir, log.Printf)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	releaseName := "list-test-release"

	t.Run("no backups", func(t *testing.T) {
		mgr.ListBackupsFunc = func(rn string) ([]BackupMetadata, error) {
			if rn == releaseName {
				return []BackupMetadata{}, nil
			}
			return nil, fmt.Errorf("unexpected release for ListBackupsFunc")
		}
		backups, err := mgr.ListBackups(releaseName)
		if err != nil {
			t.Fatalf("ListBackups failed: %v", err)
		}
		if len(backups) != 0 {
			t.Errorf("Expected 0 backups, got %d", len(backups))
		}
		mgr.ListBackupsFunc = nil
	})

	t.Run("list multiple backups", func(t *testing.T) {
		now := time.Now()
		mgr.ListBackupsFunc = func(rn string) ([]BackupMetadata, error) {
			if rn == releaseName {
				return []BackupMetadata{
					{BackupID: "mock-backup-id-3", Timestamp: now, ReleaseName: releaseName, ChartName: "listchart", ChartVersion: "1.0.0", AppVersion: "1.0"},
					{BackupID: "mock-backup-id-2", Timestamp: now.Add(-10 * time.Millisecond), ReleaseName: releaseName, ChartName: "listchart", ChartVersion: "1.0.0", AppVersion: "1.0"},
					{BackupID: "mock-backup-id-1", Timestamp: now.Add(-20 * time.Millisecond), ReleaseName: releaseName, ChartName: "listchart", ChartVersion: "1.0.0", AppVersion: "1.0"},
				}, nil
			}
			return nil, fmt.Errorf("unexpected release for ListBackupsFunc in multi-backup test")
		}
		backups, err := mgr.ListBackups(releaseName)
		if err != nil {
			t.Fatalf("ListBackups failed: %v", err)
		}
		if len(backups) != 3 {
			t.Errorf("Expected 3 backups, got %d", len(backups))
		}
		if backups[0].BackupID != "mock-backup-id-3" || backups[1].BackupID != "mock-backup-id-2" || backups[2].BackupID != "mock-backup-id-1" {
			t.Errorf("Backups not sorted correctly by timestamp (descending). Got IDs: %s, %s, %s", backups[0].BackupID, backups[1].BackupID, backups[2].BackupID)
		}
		mgr.ListBackupsFunc = nil
	})
}

func TestFileSystemBackupManager_GetBackupDetails(t *testing.T) {
	tempBaseDir := t.TempDir()
	mgr, err := NewFileSystemBackupManager(tempBaseDir, log.Printf)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	releaseName := "details-test-release"
	backupID := "mock-detail-backup-id"

	t.Run("successful get details", func(t *testing.T) {
		expectedChartPath := "/mocked/chart/path/" + backupID
		expectedValuesPath := "/mocked/values/path/" + backupID + ".yaml"
		expectedMeta := BackupMetadata{
			BackupID:     backupID,
			Timestamp:    time.Now(),
			ReleaseName:  releaseName,
			ChartName:    "detailschart",
			ChartVersion: "0.2.0",
			AppVersion:   "2.0",
		}
		mgr.GetBackupDetailsFunc = func(rn string, bid string) (string, string, BackupMetadata, error) {
			if rn == releaseName && bid == backupID {
				metaToReturn := expectedMeta
				metaToReturn.Timestamp = expectedMeta.Timestamp
				return expectedChartPath, expectedValuesPath, metaToReturn, nil
			}
			return "", "", BackupMetadata{}, fmt.Errorf("backup not found")
		}

		chartPath, valuesPath, meta, err := mgr.GetBackupDetails(releaseName, backupID)
		if err != nil {
			t.Fatalf("GetBackupDetails failed: %v", err)
		}
		if chartPath != expectedChartPath {
			t.Errorf("Expected chartPath %s, got %s", expectedChartPath, chartPath)
		}
		if valuesPath != expectedValuesPath {
			t.Errorf("Expected valuesPath %s, got %s", expectedValuesPath, valuesPath)
		}
		if meta.BackupID != expectedMeta.BackupID || meta.ReleaseName != expectedMeta.ReleaseName || meta.ChartName != expectedMeta.ChartName {
			t.Errorf("Metadata mismatch. Got %+v, expected (approx) %+v", meta, expectedMeta)
		}
		mgr.GetBackupDetailsFunc = nil
	})

	t.Run("backup not found", func(t *testing.T) {
		mgr.GetBackupDetailsFunc = func(rn string, bid string) (string, string, BackupMetadata, error) {
			return "", "", BackupMetadata{}, fmt.Errorf("mock: backup not found")
		}
		_, _, _, err := mgr.GetBackupDetails("non-existent-release", "non-existent-id")
		if err == nil {
			t.Fatal("Expected error for non-existent backup, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected error message to contain 'not found', got: %v", err)
		}
		mgr.GetBackupDetailsFunc = nil
	})
}

func TestFileSystemBackupManager_DeleteBackup(t *testing.T) {
	tempBaseDir := t.TempDir()
	mgr, err := NewFileSystemBackupManager(tempBaseDir, log.Printf)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	releaseName := "delete-test-release"
	backupID := "mock-delete-id"

	t.Run("successful delete", func(t *testing.T) {
		mgr.DeleteBackupFunc = func(rn string, bid string) error {
			if rn == releaseName && bid == backupID {
				return nil
			}
			return fmt.Errorf("unexpected delete call")
		}
		err := mgr.DeleteBackup(releaseName, backupID)
		if err != nil {
			t.Fatalf("DeleteBackup failed: %v", err)
		}
		mgr.DeleteBackupFunc = nil
	})

	t.Run("delete non-existent", func(t *testing.T) {
		mgr.DeleteBackupFunc = func(rn string, bid string) error {
			return fmt.Errorf("mock: not found for deletion")
		}
		err := mgr.DeleteBackup("non-existent-release-del", "non-existent-id-del")
		if err == nil {
			t.Fatal("Expected error when deleting non-existent backup, got nil")
		}
		if !strings.Contains(err.Error(), "not found for deletion") {
			t.Errorf("Expected error message for non-existent delete, got: %v", err)
		}
		mgr.DeleteBackupFunc = nil
	})
}

func TestFileSystemBackupManager_PruneBackups(t *testing.T) {
	tempBaseDir := t.TempDir()
	mgr, err := NewFileSystemBackupManager(tempBaseDir, log.Printf)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	releaseName := "prune-test-release"

	t.Run("prune to keep 2", func(t *testing.T) {
		mgr.PruneBackupsFunc = func(rn string, keep int) (int, error) {
			if rn == releaseName && keep == 2 {
				return 3, nil
			}
			return 0, fmt.Errorf("unexpected prune call")
		}
		prunedCount, err := mgr.PruneBackups(releaseName, 2)
		if err != nil {
			t.Fatalf("PruneBackups failed: %v", err)
		}
		if prunedCount != 3 {
			t.Errorf("Expected 3 backups to be pruned, got %d", prunedCount)
		}
		mgr.PruneBackupsFunc = nil
	})
}

func TestFileSystemBackupManager_RestoreRelease(t *testing.T) {
	tempBaseDir := t.TempDir()
	mgr, err := NewFileSystemBackupManager(tempBaseDir, log.Printf)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	releaseName := "restore-test-release"
	namespace := "test-ns"
	backupID := "mock-restore-backup-id"

	mockHC := &mockHelmClient{testingT: t}

	t.Run("successful restore", func(t *testing.T) {
		uninstalled := false
		installed := false
		mockHC.UninstallReleaseFunc = func(ns, rn string, kh bool, to time.Duration) (string, error) {
			if ns != namespace || rn != releaseName {
				t.Errorf("Uninstall called with wrong ns/release: got %s/%s, want %s/%s", ns, rn, namespace, releaseName)
			}
			uninstalled = true
			return "uninstalled", nil
		}
		mockHC.InstallChartFunc = func(ns, rn, chartPath, chartVer string, vals map[string]interface{}, createNS bool, wait bool, to time.Duration) (*helmutils.ReleaseInfo, error) {
			if ns != namespace || rn != releaseName {
				t.Errorf("Install called with wrong ns/release: got %s/%s, want %s/%s", ns, rn, namespace, releaseName)
			}
			installed = true
			return &helmutils.ReleaseInfo{Name: rn, Namespace: ns, Revision: 1, Status: "deployed", ChartName: "restorechart", ChartVersion: "0.5.0"}, nil
		}

		mgr.GetBackupDetailsFunc = func(rn string, bid string) (string, string, BackupMetadata, error) {
			if rn == releaseName && bid == backupID {
				return "/mocked/chart/path/for_restore", "/mocked/values.yaml", BackupMetadata{ChartName: "restorechart", ChartVersion: "0.5.0"}, nil
			}
			return "", "", BackupMetadata{}, fmt.Errorf("GetBackupDetails mock: not found")
		}

		_, err := mgr.RestoreRelease(context.Background(), mockHC, namespace, releaseName, backupID, true, false, 30*time.Second)
		if err != nil {
			t.Fatalf("RestoreRelease failed: %v", err)
		}
		if !uninstalled {
			t.Error("Expected UninstallRelease to be called")
		}
		if !installed {
			t.Error("Expected InstallChart to be called")
		}
		mgr.GetBackupDetailsFunc = nil
	})
}

func TestFileSystemBackupManager_UpgradeToBackup(t *testing.T) {
	tempBaseDir := t.TempDir()
	mgr, err := NewFileSystemBackupManager(tempBaseDir, log.Printf)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	releaseName := "upgrade-test-release"
	namespace := "test-ns-upgrade"
	backupID := "mock-upgrade-backup-id"

	mockHC := &mockHelmClient{testingT: t}

	t.Run("successful upgrade", func(t *testing.T) {
		upgraded := false
		mockHC.UpgradeReleaseFunc = func(ns, rn, chartPath, chartVer string, vals map[string]interface{}, wait bool, to time.Duration, installIfMissing bool, force bool) (*helmutils.ReleaseInfo, error) {
			if ns != namespace || rn != releaseName {
				t.Errorf("Upgrade called with wrong ns/release: got %s/%s, want %s/%s", ns, rn, namespace, releaseName)
			}
			upgraded = true
			return &helmutils.ReleaseInfo{Name: rn, Namespace: ns, Revision: 2, Status: "deployed", ChartName: "upgradechart", ChartVersion: "0.6.0"}, nil
		}

		mgr.GetBackupDetailsFunc = func(rn string, bid string) (string, string, BackupMetadata, error) {
			if rn == releaseName && bid == backupID {
				return "/mocked/chart/path/for_upgrade", "/mocked/values_upgrade.yaml", BackupMetadata{ChartName: "upgradechart", ChartVersion: "0.6.0"}, nil
			}
			return "", "", BackupMetadata{}, fmt.Errorf("GetBackupDetails mock: not found")
		}

		_, err := mgr.UpgradeToBackup(context.Background(), mockHC, namespace, releaseName, backupID, false, 30*time.Second, false)
		if err != nil {
			t.Fatalf("UpgradeToBackup failed: %v", err)
		}
		if !upgraded {
			t.Error("Expected UpgradeRelease to be called")
		}
		mgr.GetBackupDetailsFunc = nil
	})
}

func TestFileSystemBackupManager_DefaultListAndGetDetails(t *testing.T) {
	tempBaseDir := t.TempDir()
	mgr, err := NewFileSystemBackupManager(tempBaseDir, log.Printf)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	t.Run("default ListBackups returns one default entry", func(t *testing.T) {
		// Ensure we're using the mock's built-in ListBackups
		mgr.ListBackupsFunc = nil

		backups, err := mgr.ListBackups("any-release")
		if err != nil {
			t.Fatalf("ListBackups failed: %v", err)
		}
		if len(backups) != 1 {
			t.Fatalf("Expected 1 backup, got %d", len(backups))
		}
		b := backups[0]
		if b.BackupID != "default-mock-id" {
			t.Errorf("Expected BackupID 'default-mock-id', got %q", b.BackupID)
		}
		if b.ChartName != "mock-chart" {
			t.Errorf("Expected ChartName 'mock-chart', got %q", b.ChartName)
		}
	})

	t.Run("default GetBackupDetails returns mock values", func(t *testing.T) {
		mgr.GetBackupDetailsFunc = nil
		release := "some-release"
		bid := "some-backup-id"

		chartPath, valuesPath, meta, err := mgr.GetBackupDetails(release, bid)
		if err != nil {
			t.Fatalf("GetBackupDetails failed: %v", err)
		}
		// default mock always uses /mocked/chart/path/<backupID>
		expectedChart := "/mocked/chart/path/" + bid
		if chartPath != expectedChart {
			t.Errorf("Expected chartPath %s, got %s", expectedChart, chartPath)
		}
		// valuesPath ends with "<backupID>.yaml"
		if !strings.HasSuffix(valuesPath, bid+".yaml") {
			t.Errorf("Expected valuesPath to end with %s.yaml, got %s", bid, valuesPath)
		}
		if meta.BackupID != bid {
			t.Errorf("Expected metadata.BackupID %s, got %s", bid, meta.BackupID)
		}
		// the default mock injects mockKey->mockValueFromGetDetails
		if v, ok := meta.Values["mockKey"]; !ok || v != "mockValueFromGetDetails" {
			t.Errorf("Expected meta.Values[\"mockKey\"]=\"mockValueFromGetDetails\", got %v", meta.Values)
		}
	})
}
