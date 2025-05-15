package chartconfigmanager

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	// "encoding/json" // Not used in mock directly unless for meta file handling
	// "gopkg.in/yaml.v3" // Not used in mock directly unless for meta file handling
)

// VariableDefinition describes a variable found in a chart.
type VariableDefinition struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Default     string `json:"default,omitempty" yaml:"default,omitempty"`
}

// Product represents a pre-configured chart template.
type Product struct {
	Name        string               `json:"name" yaml:"name"`
	Description string               `json:"description,omitempty" yaml:"description,omitempty"`
	ChartPath   string               `json:"chartPath" yaml:"chartPath"`
	Variables   []VariableDefinition `json:"variables,omitempty" yaml:"variables,omitempty"`
}

// ChartInfo holds the contents of a Chart.yaml
type ChartInfo struct {
	APIVersion  string `yaml:"apiVersion" json:"apiVersion"`
	Name        string `yaml:"name" json:"name"`
	Version     string `yaml:"version" json:"version"`
	AppVersion  string `yaml:"appVersion,omitempty" json:"appVersion,omitempty"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// Manager defines the interface for managing chart products and variable extraction/replacement.
type Manager interface {
	ListProducts() ([]Product, error)
	GetProduct(productName string) (*Product, error)
	ExtractVariablesFromPath(path string) ([]VariableDefinition, error)
	InstantiateProduct(productNameOrPath string, variables map[string]interface{}, outputPath string, unassignedVarAction string) (string, error)
	ValidateChartFiles(chartPath string) error
	DefineProduct(productName string, baseChartPath string, productMetadata *Product) error
	GetChartInfo(productName string) (*ChartInfo, error)
}

// FileSystemProductManager is the mock implementation.
type FileSystemProductManager struct {
	baseProductsPath string
	log              *log.Logger

	ListProductsFunc             func() ([]Product, error)
	GetProductFunc               func(productName string) (*Product, error)
	ExtractVariablesFromPathFunc func(path string) ([]VariableDefinition, error)
	InstantiateProductFunc       func(productNameOrPath string, variables map[string]interface{}, outputPath string, unassignedVarAction string) (string, error)
	ValidateChartFilesFunc       func(chartPath string) error
	DefineProductFunc            func(productName string, baseChartPath string, productMetadata *Product) error
	GetChartInfoFunc             func(productName string) (*ChartInfo, error)
}

const (
	ProductMetaFilenameYAML = "product_meta.yaml"
	ProductMetaFilenameJSON = "product_meta.json"
	DefaultChartSubDir      = "chart"
	UnassignedVarError      = "error"
	UnassignedVarEmpty      = "empty"
	UnassignedVarKeep       = "keep"
	defaultLogDirName       = "data/logs"
	logFileName             = "chartconfigmanager.log"
)

var variableRegex = regexp.MustCompile(`@{([a-zA-Z0-9_.-]+)}`)

func NewFileSystemProductManager(baseProductsPath string, logDirectoryPath string) (*FileSystemProductManager, error) {
	if baseProductsPath == "" {
		return nil, fmt.Errorf("baseProductsPath cannot be empty")
	}
	logger := log.New(os.Stdout, "MOCK_CHARTCONFIGMAN: ", log.LstdFlags)

	return &FileSystemProductManager{
		baseProductsPath: baseProductsPath,
		log:              logger,
	}, nil
}

var _ Manager = &FileSystemProductManager{}

func (m *FileSystemProductManager) ListProducts() ([]Product, error) {
	if m.ListProductsFunc != nil {
		return m.ListProductsFunc()
	}
	return []Product{
		{Name: "mock-product-1", ChartPath: "/mock/path/product-1/chart", Description: "Mock Product 1"},
		{Name: "mock-product-2", ChartPath: "/mock/path/product-2/chart", Description: "Mock Product 2"},
	}, nil
}

func (m *FileSystemProductManager) GetProduct(productName string) (*Product, error) {
	if m.GetProductFunc != nil {
		return m.GetProductFunc(productName)
	}
	if productName == "non-existent-product" {
		return nil, fmt.Errorf("product '%s' not found", productName)
	}
	return &Product{
		Name:        productName,
		ChartPath:   filepath.Join(m.baseProductsPath, productName, "chart"),
		Description: "Mock product " + productName,
		Variables: []VariableDefinition{
			{Name: "image.tag", Default: "latest"},
		},
	}, nil
}

func (m *FileSystemProductManager) ExtractVariablesFromPath(path string) ([]VariableDefinition, error) {
	if m.ExtractVariablesFromPathFunc != nil {
		return m.ExtractVariablesFromPathFunc(path)
	}
	return []VariableDefinition{
		{Name: "replicaCount", Description: "Number of replicas"},
		{Name: "service.port", Default: "80"},
	}, nil
}

func (m *FileSystemProductManager) InstantiateProduct(productNameOrPath string, variables map[string]interface{}, outputPath string, unassignedVarAction string) (string, error) {
	if m.InstantiateProductFunc != nil {
		return m.InstantiateProductFunc(productNameOrPath, variables, outputPath, unassignedVarAction)
	}
	if outputPath == "" {
		return "", fmt.Errorf("outputPath cannot be empty")
	}
	instantiatedPath := filepath.Join(outputPath, filepath.Base(productNameOrPath))
	m.log.Printf("Mock InstantiateProduct: productNameOrPath=%s, outputPath=%s, instantiatedPath=%s", productNameOrPath, outputPath, instantiatedPath)
	return instantiatedPath, nil
}

func (m *FileSystemProductManager) ValidateChartFiles(chartPath string) error {
	if m.ValidateChartFilesFunc != nil {
		return m.ValidateChartFilesFunc(chartPath)
	}
	if strings.Contains(chartPath, "invalid-chart") {
		return fmt.Errorf("mock validation error: chart at %s is invalid", chartPath)
	}
	return nil
}

func (m *FileSystemProductManager) DefineProduct(productName string, baseChartPath string, productMetadata *Product) error {
	if m.DefineProductFunc != nil {
		return m.DefineProductFunc(productName, baseChartPath, productMetadata)
	}
	if productName == "" {
		return fmt.Errorf("productName cannot be empty")
	}
	m.log.Printf("Mock DefineProduct: productName=%s, baseChartPath=%s", productName, baseChartPath)
	return nil
}

func (m *FileSystemProductManager) GetChartInfo(productName string) (*ChartInfo, error) {
	if m.GetChartInfoFunc != nil {
		return m.GetChartInfoFunc(productName)
	}
	if productName == "product-without-chartinfo" {
		return nil, fmt.Errorf("Chart.yaml not found for product %s", productName)
	}
	return &ChartInfo{
		APIVersion:  "v2",
		Name:        productName + "-chart",
		Version:     "0.1.0",
		AppVersion:  "1.0.0",
		Description: "Mock chart info for " + productName,
	}, nil
}

// LoadVariables is a mock for a global function presumably used by tests.
// The test calls it with three string arguments.
func LoadVariables(filePath string, arg2 string, arg3 string) (map[string]interface{}, error) {
	// arg2 and arg3 are ignored for now as their purpose in the original or test is unknown.
	if filePath == "non-existent-vars.yaml" {
		return nil, fmt.Errorf("mock error: file not found %s", filePath)
	}
	// Return some mock variables based on filePath or other args if their meaning becomes clear.
	return map[string]interface{}{
		"globalVar1":   "globalValue1FromMock",
		"fromFilePath": filePath,
		"arg2_val":     arg2,
		"arg3_val":     arg3,
		"nested": map[string]interface{}{
			"key": "nestedValueFromMock",
		},
	}, nil
}
