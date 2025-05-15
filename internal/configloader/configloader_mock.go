package configloader

import (
	"encoding/json" // Keep for ToJSON
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	// "sort" // Removed as it seems unused in the mock
	"strings"
	"sync"
	"time"
)

// Options defines the parameters for loading configurations.
type Options struct {
	BasePath               string
	CustomFilePaths        []string
	Environment            string
	EnableDatabaseGrouping bool
}

// LoadedConfig represents the fully parsed and resolved configuration.
type LoadedConfig struct {
	Main               map[string]string            `json:"main"`
	DatabaseConfigs    map[string]map[string]string `json:"database_configs,omitempty"`
	Metadata           map[string]interface{}       `json:"metadata"`
	rawMainConfig      map[string]string
	rawDatabaseConfigs map[string]map[string]string
	opts               Options
}

var dbFileRegex = regexp.MustCompile(`^database_(\w+)\.conf$`)

// newLoadedConfig initializes a LoadedConfig structure.
func newLoadedConfig(opts Options) *LoadedConfig {
	return &LoadedConfig{
		Main:               make(map[string]string),
		DatabaseConfigs:    make(map[string]map[string]string),
		Metadata:           make(map[string]interface{}),
		rawMainConfig:      make(map[string]string),
		rawDatabaseConfigs: make(map[string]map[string]string),
		opts:               opts,
	}
}

// Load simulates the Load function.
func Load(opts Options) (*LoadedConfig, error) {
	lc := newLoadedConfig(opts)
	lc.Main["mock_key_main"] = "mock_value_main"
	lc.Main["another_key"] = "another_value"
	lc.Main["var1"] = "value1"
	lc.Main["var2"] = "${var1}_value2"
	if opts.EnableDatabaseGrouping {
		lc.DatabaseConfigs["mockdb"] = map[string]string{
			"host": "localhost",
			"port": "5432",
		}
	}
	lc.Metadata["source_type"] = "mock_load"
	lc.Metadata["parsed_files"] = []string{"/fake/path/mock.conf"}
	lc.Metadata["database_grouping_enabled"] = opts.EnableDatabaseGrouping
	lc.Metadata["extraction_date"] = time.Now().UTC().Format(time.RFC3339)
	lc.Main = resolveConfigMap(lc.rawMainConfig, lc.Main, "MAIN_RESOLVED_MOCK", "")
	if opts.EnableDatabaseGrouping {
		for dbType, rawDbConf := range lc.rawDatabaseConfigs {
			lc.DatabaseConfigs[dbType] = resolveConfigMap(rawDbConf, lc.Main, "DB_"+strings.ToUpper(dbType)+"_RESOLVED_MOCK", "")
		}
	}
	return lc, nil
}

// LoadWithDefaults simulates the LoadWithDefaults function.
func LoadWithDefaults(basePath string, env string, enableDBGrouping bool) (*LoadedConfig, error) {
	if basePath == "" {
		basePath = "/mock/base"
	}
	opts := Options{
		BasePath:               basePath,
		Environment:            env,
		EnableDatabaseGrouping: enableDBGrouping,
	}
	return Load(opts)
}

// DiscoverDefaultPaths simulates DiscoverDefaultPaths.
func DiscoverDefaultPaths(basePath string, env string) (string, string, error) {
	if basePath == "" {
		basePath = "/mock/base"
	}
	if env != "" {
		return filepath.Join(basePath, fmt.Sprintf("install-%s.conf", env)), filepath.Join(basePath, fmt.Sprintf("conf-%s", env)), nil
	}
	return filepath.Join(basePath, "install.conf"), filepath.Join(basePath, "conf"), nil
}

func parseConfFile(filePath string, targetConfig map[string]string, sectionName string, logPrefix string) error {
	targetConfig["mock_parsed_key_from_"+filepath.Base(filePath)] = "mock_value"
	targetConfig["input_var"] = "hello"
	targetConfig["another_input"] = "world_${input_var}"
	targetConfig["to_be_resolved"] = "original_value"
	return nil
}

func resolveValue(rawValue string, context map[string]string) string {
	re := regexp.MustCompile(`\${(.*?)}`)
	resolvedValue := re.ReplaceAllStringFunc(rawValue, func(match string) string {
		varName := strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")
		if val, ok := context[varName]; ok {
			return val
		}
		return match
	})
	return resolvedValue
}

func resolveConfigMap(rawConfig map[string]string, primaryContext map[string]string, sectionName string, logPrefix string) map[string]string {
	resolved := make(map[string]string)
	combinedContext := make(map[string]string)
	for k, v := range primaryContext {
		combinedContext[k] = v
	}
	for k, v := range rawConfig {
		combinedContext[k] = v
	}
	for k, v := range rawConfig {
		resolved[k] = resolveValue(v, combinedContext)
	}
	return resolved
}

// ToJSON converts the LoadedConfig into a JSON byte array.
func (lc *LoadedConfig) ToJSON() ([]byte, error) {
	// Create a map that matches the JSON structure, omitting internal fields.
	outputMap := make(map[string]interface{})
	outputMap["main"] = lc.Main
	if lc.opts.EnableDatabaseGrouping && len(lc.DatabaseConfigs) > 0 {
		outputMap["database_configs"] = lc.DatabaseConfigs
	}
	outputMap["metadata"] = lc.Metadata
	return json.MarshalIndent(outputMap, "", "  ")
}

// SaveAsJSON writes the LoadedConfig as JSON to the given file path.
func (lc *LoadedConfig) SaveAsJSON(filePath string) error {
	data, err := lc.ToJSON()
	if err != nil {
		return err
	}
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

// ToMap converts the LoadedConfig into a map[string]interface{},
// mirroring the structure of the JSON output.
func (lc *LoadedConfig) ToMap() map[string]interface{} {
	outputMap := make(map[string]interface{})
	outputMap["main"] = lc.Main
	if lc.opts.EnableDatabaseGrouping && len(lc.DatabaseConfigs) > 0 {
		outputMap["database_configs"] = lc.DatabaseConfigs
	}
	outputMap["metadata"] = lc.Metadata
	return outputMap
}

var (
	logFileHandle *os.File
	logMutex      sync.Mutex
	logInitErr    error
	logInitOnce   sync.Once
)

const logPrefixDefault = "[configloader_mock] "

func ensureLoggingInitialized(basePath string) { /* mock */ }
func logMessage(msg string)                    { fmt.Println(logPrefixDefault + msg) }
func logWarning(msg string)                    { fmt.Println(logPrefixDefault + "[WARNING] " + msg) }
func logError(msg string)                      { fmt.Println(logPrefixDefault + "[ERROR] " + msg) }
