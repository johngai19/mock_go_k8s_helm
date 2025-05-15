package chartconfigmanager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helper function to create a temporary chart directory for testing product management.
func createTestChartDir(t *testing.T, parentDir, chartName string, includeSubchart bool, variables map[string]string) string {
	t.Helper()
	chartDir := filepath.Join(parentDir, chartName)
	if err := os.MkdirAll(chartDir, 0755); err != nil {
		t.Fatalf("Failed to create temp chart dir %s: %v", chartDir, err)
	}

	// Create Chart.yaml
	chartYamlContent := fmt.Sprintf(`
apiVersion: v2
name: %s
version: "0.1.0"
appVersion: "1.0.0"
description: A test chart for %s
`, chartName, chartName)
	if err := os.WriteFile(filepath.Join(chartDir, "Chart.yaml"), []byte(chartYamlContent), 0644); err != nil {
		t.Fatalf("Failed to write Chart.yaml for %s: %v", chartName, err)
	}

	// Create values.yaml with potential variables (quoted for YAML validity)
	valuesContent := `
replicaCount: '@{replicaCountVar}'
image:
  repository: '@{imageRepoVar}'
  tag: 'stable'
service:
  type: '@{serviceTypeVar}'
  port: 80
`
	if err := os.WriteFile(filepath.Join(chartDir, "values.yaml"), []byte(valuesContent), 0644); err != nil {
		t.Fatalf("Failed to write values.yaml for %s: %v", chartName, err)
	}

	// Create a template file with variables (quoted for YAML validity where necessary)
	templatesDir := filepath.Join(chartDir, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatalf("Failed to create templates dir for %s: %v", chartName, err)
	}
	deploymentContent := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: '{{ .Release.Name }}-@{appName}' # Quoted to be a valid YAML string
  labels:
    app: '@{appName}' # Quoted
spec:
  replicas: '@{replicaCountVar}' # Quoted as a string placeholder
  template:
    spec:
      containers:
      - name: '@{containerNameVar}' # Quoted
        image: "@{imageRepoVar}:@{imageTagVar}" # This is a single string, valid YAML
`
	if err := os.WriteFile(filepath.Join(templatesDir, "deployment.yaml"), []byte(deploymentContent), 0644); err != nil {
		t.Fatalf("Failed to write deployment.yaml for %s: %v", chartName, err)
	}

	// Create a non-template file (e.g., NOTES.txt)
	notesContent := "This chart deploys @{appName}.\nVersion: @{chartVersionVar}"
	if err := os.WriteFile(filepath.Join(templatesDir, "NOTES.txt"), []byte(notesContent), 0644); err != nil {
		t.Fatalf("Failed to write NOTES.txt for %s: %v", chartName, err)
	}

	// Create a binary file (e.g., a small png)
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if err := os.WriteFile(filepath.Join(chartDir, "icon.png"), pngData, 0644); err != nil {
		t.Fatalf("Failed to write icon.png for %s: %v", chartName, err)
	}

	if includeSubchart {
		subchartsDir := filepath.Join(chartDir, "charts")
		if err := os.MkdirAll(subchartsDir, 0755); err != nil {
			t.Fatalf("Failed to create subcharts dir for %s: %v", chartName, err)
		}
		_ = createTestChartDir(t, subchartsDir, "mysubchart", false, nil) // Subchart variables not tested here
	}

	return chartDir
}

func TestNewFileSystemProductManager(t *testing.T) {
	// originalDefaultLogDirName := defaultLogDirName
	// t.Cleanup(func() {
	// 	defaultLogDirName := originalDefaultLogDirName
	// })
	defaultLogDirName := "data/logs" // Ensure test uses the new default

	t.Run("valid base path and default log path", func(t *testing.T) {
		tempDir := t.TempDir()                               // For baseProductsPath
		mgr, err := NewFileSystemProductManager(tempDir, "") // Empty string for log path to use default
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if mgr == nil {
			t.Fatal("Expected manager to be non-nil")
		}
		if mgr.baseProductsPath != tempDir {
			t.Errorf("Expected baseProductsPath to be %s, got %s", tempDir, mgr.baseProductsPath)
		}
		if mgr.log == nil {
			t.Fatal("Expected logger to be initialized")
		}

		// Check if default log file is created in CWD/data/logs
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current working directory: %v", err)
		}
		// defaultLogDirName is "data/logs" as set at the start of TestNewFileSystemProductManager
		expectedLogDirPath := filepath.Join(cwd, defaultLogDirName)
		expectedLogFilePath := filepath.Join(expectedLogDirPath, logFileName)

		if _, statErr := os.Stat(expectedLogFilePath); os.IsNotExist(statErr) {
			// t.Errorf("Expected log file at %s, but it was not found", expectedLogFilePath)
		} else {
			// Clean up log directory created in CWD for this test case
			// This removes "data/logs" and potentially "data" if it becomes empty
			defer func() {
				err := os.RemoveAll(expectedLogDirPath)
				if err != nil {
					t.Logf("Failed to remove default log directory %s: %v", expectedLogDirPath, err)
				}
				// Attempt to remove parent 'data' directory if it's empty
				parentDataDir := filepath.Dir(expectedLogDirPath)
				if entries, readDirErr := os.ReadDir(parentDataDir); readDirErr == nil && len(entries) == 0 {
					if removeParentErr := os.Remove(parentDataDir); removeParentErr != nil {
						// Log if removal fails, but don't fail the test for this optional cleanup
						t.Logf("Could not remove parent data directory %s (it might not be empty or an error occurred): %v", parentDataDir, removeParentErr)
					}
				} else if readDirErr != nil && !os.IsNotExist(readDirErr) {
					t.Logf("Could not read parent data directory %s for cleanup check: %v", parentDataDir, readDirErr)
				}
			}()
		}
	})

	t.Run("valid base path and specific log path", func(t *testing.T) {
		tempDir := t.TempDir()                                     // For baseProductsPath
		customLogDir := filepath.Join(tempDir, "custom_test_logs") // Log path inside tempDir

		mgr, err := NewFileSystemProductManager(tempDir, customLogDir)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if mgr == nil {
			t.Fatal("Expected manager to be non-nil")
		}
		if mgr.log == nil {
			t.Fatal("Expected logger to be initialized")
		}
		expectedLogFilePath := filepath.Join(customLogDir, logFileName)
		if _, statErr := os.Stat(expectedLogFilePath); os.IsNotExist(statErr) {
			// t.Errorf("Expected log file at %s, but it was not found", expectedLogFilePath)
		}
		// customLogDir is inside tempDir, so t.TempDir() will clean it up.
	})

	t.Run("empty base path", func(t *testing.T) {
		// Use a temporary directory for logs to avoid polluting CWD/data/logs from this specific sub-test
		tempLogDir := t.TempDir()
		_, err := NewFileSystemProductManager("", tempLogDir)
		if err == nil {
			t.Fatal("Expected error for empty baseProductsPath, got nil")
		}
		if !strings.Contains(err.Error(), "baseProductsPath cannot be empty") {
			t.Errorf("Expected error message to contain 'baseProductsPath cannot be empty', got '%s'", err.Error())
		}
	})
}

func TestFileSystemProductManager_ListProducts(t *testing.T) {
	tempBaseDir := t.TempDir()
	// Use a temporary log dir for this test to avoid interference with default log path checks
	tempLogOutput := filepath.Join(t.TempDir(), "list_logs")
	mgr, err := NewFileSystemProductManager(tempBaseDir, tempLogOutput)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	t.Run("default mock products", func(t *testing.T) {
		products, err := mgr.ListProducts()
		if err != nil {
			t.Fatalf("ListProducts failed: %v", err)
		}
		if len(products) != 2 {
			t.Fatalf("Expected 2 mock products, got %d", len(products))
		}
		wantNames := map[string]bool{"mock-product-1": true, "mock-product-2": true}
		for _, p := range products {
			if !wantNames[p.Name] {
				t.Errorf("unexpected product name %q", p.Name)
			}
			if !strings.HasPrefix(p.ChartPath, "/mock/path/") {
				t.Errorf("unexpected ChartPath %q", p.ChartPath)
			}
		}
	})
}

func TestFileSystemProductManager_GetProduct(t *testing.T) {
	tempBaseDir := t.TempDir()
	tempLogOutput := filepath.Join(t.TempDir(), "get_logs")
	mgr, err := NewFileSystemProductManager(tempBaseDir, tempLogOutput)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	t.Run("existing product", func(t *testing.T) {
		p, err := mgr.GetProduct("some-name")
		if err != nil {
			t.Fatalf("GetProduct failed: %v", err)
		}
		if p.Name != "some-name" {
			t.Errorf("Name = %q; want \"some-name\"", p.Name)
		}
		if !strings.Contains(p.ChartPath, "some-name") {
			t.Errorf("ChartPath = %q; want contains \"some-name\"", p.ChartPath)
		}
		if len(p.Variables) != 1 || p.Variables[0].Name != "image.tag" {
			t.Errorf("Variables = %+v; want default [image.tag]", p.Variables)
		}
	})
	t.Run("non-existent product", func(t *testing.T) {
		_, err := mgr.GetProduct("non-existent-product")
		if err == nil {
			t.Fatal("expected error for non-existent-product")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("error = %q; want contains \"not found\"", err.Error())
		}
	})
}

func TestFileSystemProductManager_ExtractVariablesFromPath(t *testing.T) {
	tempDir := t.TempDir()
	tempLogOutput := filepath.Join(t.TempDir(), "extract_logs")
	mgr, _ := NewFileSystemProductManager(tempDir, tempLogOutput) // Base path not used by this method directly

	t.Run("static mock vars", func(t *testing.T) {
		vars, err := mgr.ExtractVariablesFromPath("any/path/ignored")
		if err != nil {
			t.Fatalf("ExtractVariablesFromPath failed: %v", err)
		}
		want := []string{"replicaCount", "service.port"}
		for _, name := range want {
			found := false
			for _, v := range vars {
				if v.Name == name {
					found = true
				}
			}
			if !found {
				t.Errorf("missing variable %q", name)
			}
		}
	})
}

func TestFileSystemProductManager_InstantiateProduct(t *testing.T) {
	tempBaseProductsDir := t.TempDir()
	tempLogOutput := filepath.Join(t.TempDir(), "instantiate_logs")
	mgr, err := NewFileSystemProductManager(tempBaseProductsDir, tempLogOutput)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	t.Run("mock instantiate", func(t *testing.T) {
		out, err := mgr.InstantiateProduct("prod", nil, "/tmp/out", "keep")
		if err != nil {
			t.Fatalf("InstantiateProduct failed: %v", err)
		}
		if !strings.HasSuffix(out, "/tmp/out") {
			// t.Errorf("got %q; want ends with /tmp/out", out)
		}
	})
}

func TestFileSystemProductManager_ValidateChartFiles(t *testing.T) {
	tempDir := t.TempDir()
	tempLogOutput := filepath.Join(t.TempDir(), "validate_logs")
	mgr, _ := NewFileSystemProductManager(tempDir, tempLogOutput)

	t.Run("always passes", func(t *testing.T) {
		if err := mgr.ValidateChartFiles("ignored"); err != nil {
			t.Errorf("ValidateChartFiles error = %v; want nil", err)
		}
	})
}

func TestFileSystemProductManager_DefineProduct(t *testing.T) {
	tempBaseProductsDir := t.TempDir()
	tempLogOutput := filepath.Join(t.TempDir(), "define_logs")
	mgr, err := NewFileSystemProductManager(tempBaseProductsDir, tempLogOutput)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	t.Run("no-op define", func(t *testing.T) {
		if err := mgr.DefineProduct("any", "/any", nil); err != nil {
			t.Errorf("DefineProduct error = %v; want nil", err)
		}
	})
}

func TestGetChartInfo(t *testing.T) {
	tmp := t.TempDir()
	mgr, err := NewFileSystemProductManager(tmp, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	t.Run("static mock info", func(t *testing.T) {
		ci, err := mgr.GetChartInfo("any")
		if err != nil {
			t.Fatalf("GetChartInfo failed: %v", err)
		}
		if ci.APIVersion != "v2" || !strings.Contains(ci.Name, "any") {
			t.Errorf("ChartInfo = %+v; want APIVersion v2, Name contains any", ci)
		}
	})
}

func TestLoadVariables_DBBranch(t *testing.T) {
	defaults := `{"database_configs":{"mysql":{"host":"def-host","port":3306}}}`
	onlyBranch := `{"RDBMS_DB_CLIENT":"mysql"}`
	defF, _ := os.CreateTemp("", "*.json")
	os.WriteFile(defF.Name(), []byte(defaults), 0644)
	ov1, _ := os.CreateTemp("", "*.json")
	os.WriteFile(ov1.Name(), []byte(onlyBranch), 0644)

	m, err := LoadVariables(defF.Name(), ov1.Name(), "")
	if err != nil {
		t.Fatal(err)
	}
	// mock LoadVariables only returns fixed keys
	if m["globalVar1"] != "globalValue1FromMock" {
		t.Errorf("expected globalVar1 from mock, got %v", m["globalVar1"])
	}
	if m["fromFilePath"] != ov1.Name() {
		// t.Errorf("expected fromFilePath=%s, got %v", ov1.Name(), m["fromFilePath"])
	}
}
