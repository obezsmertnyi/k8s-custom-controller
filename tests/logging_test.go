package tests

import (
	"bytes"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

// TestLoggingLevels verifies that log levels are respected
func TestLoggingLevels(t *testing.T) {
	// Save the original logger and restore it after the test
	originalLogger := log.Logger
	defer func() {
		log.Logger = originalLogger
	}()

	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Test different log levels
	testCases := []struct {
		level          string
		expectedDebug  bool
		expectedInfo   bool
		expectedWarn   bool
		expectedError  bool
	}{
		{"debug", true, true, true, true},
		{"info", false, true, true, true},
		{"warn", false, false, true, true},
		{"error", false, false, false, true},
	}

	for _, tc := range testCases {
		t.Run("LogLevel_"+tc.level, func(t *testing.T) {
			// Reset the buffer
			buf.Reset()

			// Set the log level
			level, err := zerolog.ParseLevel(tc.level)
			assert.NoError(t, err, "Should parse log level")

			// Configure the logger to write to our buffer
			logger := zerolog.New(&buf).Level(level).With().Timestamp().Logger()
			log.Logger = logger

			// Generate logs at different levels
			log.Debug().Msg("Debug message")
			log.Info().Msg("Info message")
			log.Warn().Msg("Warning message")
			log.Error().Msg("Error message")

			// Check if the expected messages are in the output
			output := buf.String()
			
			// Debug messages should only appear when level is debug
			if tc.expectedDebug {
				assert.Contains(t, output, "Debug message", "Debug message should be logged at level "+tc.level)
			} else {
				assert.NotContains(t, output, "Debug message", "Debug message should not be logged at level "+tc.level)
			}

			// Info messages should appear when level is debug or info
			if tc.expectedInfo {
				assert.Contains(t, output, "Info message", "Info message should be logged at level "+tc.level)
			} else {
				assert.NotContains(t, output, "Info message", "Info message should not be logged at level "+tc.level)
			}

			// Warning messages should appear when level is debug, info, or warn
			if tc.expectedWarn {
				assert.Contains(t, output, "Warning message", "Warning message should be logged at level "+tc.level)
			} else {
				assert.NotContains(t, output, "Warning message", "Warning message should not be logged at level "+tc.level)
			}

			// Error messages should always appear
			assert.Contains(t, output, "Error message", "Error message should always be logged")
		})
	}
}

// TestLoggingFormat verifies that log format (text/json) is respected
func TestLoggingFormat(t *testing.T) {
	// Save the original logger and restore it after the test
	originalLogger := log.Logger
	defer func() {
		log.Logger = originalLogger
	}()

	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Test text format
	t.Run("TextFormat", func(t *testing.T) {
		buf.Reset()
		logger := zerolog.New(&buf).With().Timestamp().Logger()
		log.Logger = logger

		// Set console writer for text format
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: &buf, NoColor: true})
		
		log.Info().Msg("Test message")
		
		output := buf.String()
		// Text format should not contain JSON markers
		assert.NotContains(t, output, "\"message\":", "Text format should not contain JSON markers")
		assert.Contains(t, output, "Test message", "Text format should contain the message")
	})

	// Test JSON format
	t.Run("JSONFormat", func(t *testing.T) {
		buf.Reset()
		logger := zerolog.New(&buf).With().Timestamp().Logger()
		log.Logger = logger
		
		log.Info().Msg("Test message")
		
		output := buf.String()
		// JSON format should contain JSON markers
		assert.Contains(t, output, "\"message\"", "JSON format should contain JSON markers")
		assert.Contains(t, output, "Test message", "JSON format should contain the message")
	})
}
