package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

// ANSI color codes for log levels
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"   // Error, Fatal, Panic
	ColorYellow = "\033[33m"   // Warning
	ColorYellowBold = "\033[1;33m" // Bold Yellow for performance metrics
	ColorPink   = "\033[38;5;206m" // Pink for error levels
	ColorBlue   = "\033[34m"   // Info
	ColorCyan   = "\033[36m"   // Debug
	ColorGray   = "\033[37m"   // Trace
)

// getColorForLevel returns the appropriate color for a log level
func getColorForLevel(level logrus.Level) string {
	switch level {
	case logrus.PanicLevel, logrus.FatalLevel, logrus.ErrorLevel:
		return ColorPink // Pink for error levels as requested
	case logrus.WarnLevel:
		return ColorYellow
	case logrus.InfoLevel:
		return ColorBlue
	case logrus.DebugLevel:
		return ColorCyan
	case logrus.TraceLevel:
		return ColorGray
	default:
		return ColorReset
	}
}

// isPerformanceMetricField checks if a field name is a key performance metric that should be highlighted
func isPerformanceMetricField(key string) bool {
	performanceMetrics := []string{
		"recognitionTimeMs",
		"denoiserTimeMs",
		"vadProcessingTimeMs",
	}

	for _, metric := range performanceMetrics {
		if key == metric {
			return true
		}
	}
	return false
}

// highlightPerformanceMetric adds yellow highlighting to performance metric values
func highlightPerformanceMetric(value interface{}) string {
	return ColorYellowBold + fmt.Sprintf("%v", value) + ColorReset
}

// CustomFormatter is a custom log formatter that ensures the "level" field appears first
// and maintains consistent field ordering for better readability
type CustomFormatter struct {
	TimestampFormat string
	ForceColors     bool // Force color output even when not writing to terminal
}

// Format implements the logrus.Formatter interface
func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	data := make(logrus.Fields, len(entry.Data)+4)

	// Define the preferred field order
	preferredOrder := []string{"level", "time", "component", "action", "msg"}

	// Add standard fields with controlled ordering
	// Add color to level field for error levels
	levelValue := entry.Level.String()
	if f.ForceColors || (entry.Level >= logrus.ErrorLevel) {
		color := getColorForLevel(entry.Level)
		levelValue = color + levelValue + ColorReset
	}
	data["level"] = levelValue
	data["time"] = entry.Time.Format(f.TimestampFormat)
	data["msg"] = entry.Message

	// Add custom fields from entry.Data
	for k, v := range entry.Data {
		// Skip standard fields that we've already added
		if k == "level" || k == "time" || k == "msg" {
			continue
		}
		data[k] = v
	}

	// Create a new map with controlled field ordering
	orderedData := make(map[string]interface{})

	// First, add fields in preferred order
	for _, key := range preferredOrder {
		if value, exists := data[key]; exists {
			orderedData[key] = value
		}
	}

	// Then, add remaining fields in alphabetical order
	var remainingKeys []string
	for key := range data {
		// Skip keys that are already in preferred order
		found := false
		for _, preferredKey := range preferredOrder {
			if key == preferredKey {
				found = true
				break
			}
		}
		if !found {
			remainingKeys = append(remainingKeys, key)
		}
	}

	// Sort remaining keys alphabetically
	sort.Strings(remainingKeys)

	// Add remaining fields
	for _, key := range remainingKeys {
		orderedData[key] = data[key]
	}

	// Create JSON manually to maintain field order
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	b.WriteString("{")

	// Add fields in preferred order first
	firstField := true
	for _, key := range preferredOrder {
		if value, exists := orderedData[key]; exists {
			if !firstField {
				b.WriteString(",")
			}
			firstField = false

			// Special handling for level field to preserve ANSI color codes
			if key == "level" {
				// Escape the level value manually to preserve ANSI color codes
				levelStr := fmt.Sprintf("%v", value)
				fmt.Fprintf(b, "\"%s\":\"%s\"", key, levelStr)
			} else if isPerformanceMetricField(key) {
				// Apply yellow highlighting to performance metrics
				highlightedValue := highlightPerformanceMetric(value)
				fmt.Fprintf(b, "\"%s\":\"%s\"", key, highlightedValue)
			} else {
				jsonValue, _ := json.Marshal(value)
				fmt.Fprintf(b, "\"%s\":%s", key, string(jsonValue))
			}
		}
	}

	// Add remaining fields
	for _, key := range remainingKeys {
		if value, exists := orderedData[key]; exists {
			if !firstField {
				b.WriteString(",")
			}
			firstField = false

			if isPerformanceMetricField(key) {
				// Apply yellow highlighting to performance metrics
				highlightedValue := highlightPerformanceMetric(value)
				fmt.Fprintf(b, "\"%s\":\"%s\"", key, highlightedValue)
			} else {
				jsonValue, _ := json.Marshal(value)
				fmt.Fprintf(b, "\"%s\":%s", key, string(jsonValue))
			}
		}
	}

	b.WriteString("}\n")

	return b.Bytes(), nil
}

// CustomFormatterText is a custom log formatter for text format that ensures the "level" field appears first
type CustomFormatterText struct {
	TimestampFormat string
	ForceColors     bool // Force color output even when not writing to terminal
}

// Format implements the logrus.Formatter interface for text output
func (f *CustomFormatterText) Format(entry *logrus.Entry) ([]byte, error) {
	data := make(logrus.Fields, len(entry.Data)+3)

	// Define the preferred field order
	preferredOrder := []string{"level", "time", "component", "action", "msg"}

	// Add standard fields with controlled ordering
	// Add color to level field for all levels in text format
	levelValue := strings.ToUpper(entry.Level.String())
	if f.ForceColors || entry.Level >= logrus.WarnLevel { // Show colors for warn and above in text format
		color := getColorForLevel(entry.Level)
		levelValue = color + levelValue + ColorReset
	}
	data["level"] = levelValue
	data["time"] = entry.Time.Format(f.TimestampFormat)
	data["msg"] = entry.Message

	// Add custom fields from entry.Data
	for k, v := range entry.Data {
		// Skip standard fields that we've already added
		if k == "level" || k == "time" || k == "msg" {
			continue
		}
		data[k] = v
	}

	// Create a new map with controlled field ordering
	orderedData := make(map[string]interface{})

	// First, add fields in preferred order
	for _, key := range preferredOrder {
		if value, exists := data[key]; exists {
			orderedData[key] = value
		}
	}

	// Then, add remaining fields in alphabetical order
	var remainingKeys []string
	for key := range data {
		// Skip keys that are already in preferred order
		found := false
		for _, preferredKey := range preferredOrder {
			if key == preferredKey {
				found = true
				break
			}
		}
		if !found {
			remainingKeys = append(remainingKeys, key)
		}
	}

	// Sort remaining keys alphabetically
	sort.Strings(remainingKeys)

	// Add remaining fields
	for _, key := range remainingKeys {
		orderedData[key] = data[key]
	}

	// Build text output with controlled ordering
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	// Format: time=... level=INFO msg="..." key1=value1 key2=value2
	var parts []string

	// Add fields in preferred order first
	for _, key := range preferredOrder {
		if value, exists := orderedData[key]; exists {
			if isPerformanceMetricField(key) {
				// Apply yellow highlighting to performance metrics
				highlightedValue := highlightPerformanceMetric(value)
				parts = append(parts, fmt.Sprintf("%s=%v", key, highlightedValue))
			} else {
				parts = append(parts, fmt.Sprintf("%s=%v", key, value))
			}
		}
	}

	// Add remaining fields
	for _, key := range remainingKeys {
		if value, exists := orderedData[key]; exists {
			if isPerformanceMetricField(key) {
				// Apply yellow highlighting to performance metrics
				highlightedValue := highlightPerformanceMetric(value)
				parts = append(parts, fmt.Sprintf("%s=%v", key, highlightedValue))
			} else {
				parts = append(parts, fmt.Sprintf("%s=%v", key, value))
			}
		}
	}

	b.WriteString(strings.Join(parts, " "))
	b.WriteString("\n")

	return b.Bytes(), nil
}