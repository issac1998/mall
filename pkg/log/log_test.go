package log

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogConfig(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() {
		logger = originalLogger
	}()

	t.Run("InitWithDefaultConfig", func(t *testing.T) {
		cfg := Config{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		}

		err := Init(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, logger)
		assert.Equal(t, logrus.InfoLevel, logger.Level)
	})

	t.Run("InitWithJSONFormat", func(t *testing.T) {
		cfg := Config{
			Level:  "debug",
			Format: "json",
			Output: "stdout",
		}

		err := Init(cfg)
		assert.NoError(t, err)
		
		// Check if formatter is JSON
		_, ok := logger.Formatter.(*logrus.JSONFormatter)
		assert.True(t, ok)
	})

	t.Run("InitWithTextFormat", func(t *testing.T) {
		cfg := Config{
			Level:  "warn",
			Format: "text",
			Output: "stdout",
		}

		err := Init(cfg)
		assert.NoError(t, err)
		
		// Check if formatter is Text
		_, ok := logger.Formatter.(*logrus.TextFormatter)
		assert.True(t, ok)
	})

	t.Run("InitWithFileOutput", func(t *testing.T) {
		tempDir := t.TempDir()
		logFile := filepath.Join(tempDir, "test.log")

		cfg := Config{
			Level:      "error",
			Format:     "json",
			Output:     "file",
			Filename:   logFile,
			MaxSize:    10,
			MaxAge:     7,
			MaxBackups: 3,
			Compress:   true,
		}

		err := Init(cfg)
		assert.NoError(t, err)
		
		// Test that we can write to the log
		Error("test error message")
		
		// Check if file exists
		_, err = os.Stat(logFile)
		assert.NoError(t, err)
	})

	t.Run("InitWithInvalidLevel", func(t *testing.T) {
		cfg := Config{
			Level:  "invalid",
			Format: "text",
			Output: "stdout",
		}

		err := Init(cfg)
		assert.NoError(t, err)
		// Should default to InfoLevel
		assert.Equal(t, logrus.InfoLevel, logger.Level)
	})
}

func TestLogLevels(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	
	// Save original logger
	originalLogger := logger
	defer func() {
		logger = originalLogger
	}()

	// Initialize with debug level and text format
	logger = logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	logger.SetOutput(&buf)

	t.Run("DebugLevel", func(t *testing.T) {
		buf.Reset()
		Debug("debug message")
		assert.Contains(t, buf.String(), "debug message")
		assert.Contains(t, buf.String(), "level=debug")

		buf.Reset()
		Debugf("debug %s", "formatted")
		assert.Contains(t, buf.String(), "debug formatted")
	})

	t.Run("InfoLevel", func(t *testing.T) {
		buf.Reset()
		Info("info message")
		assert.Contains(t, buf.String(), "info message")
		assert.Contains(t, buf.String(), "level=info")

		buf.Reset()
		Infof("info %d", 123)
		assert.Contains(t, buf.String(), "info 123")
	})

	t.Run("WarnLevel", func(t *testing.T) {
		buf.Reset()
		Warn("warn message")
		assert.Contains(t, buf.String(), "warn message")
		assert.Contains(t, buf.String(), "level=warning")

		buf.Reset()
		Warnf("warn %s", "test")
		assert.Contains(t, buf.String(), "warn test")
	})

	t.Run("ErrorLevel", func(t *testing.T) {
		buf.Reset()
		Error("error message")
		assert.Contains(t, buf.String(), "error message")
		assert.Contains(t, buf.String(), "level=error")

		buf.Reset()
		Errorf("error %v", 404)
		assert.Contains(t, buf.String(), "error 404")
	})
}

func TestLogFields(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	
	// Save original logger
	originalLogger := logger
	defer func() {
		logger = originalLogger
	}()

	// Initialize with JSON format for easier parsing
	logger = logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(&buf)

	t.Run("WithField", func(t *testing.T) {
		buf.Reset()
		WithField("user_id", 123).Info("user action")
		
		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)
		
		assert.Equal(t, "user action", logEntry["msg"])
		assert.Equal(t, float64(123), logEntry["user_id"])
	})

	t.Run("WithFields", func(t *testing.T) {
		buf.Reset()
		WithFields(logrus.Fields{
			"user_id": 456,
			"action":  "login",
		}).Info("user logged in")
		
		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)
		
		assert.Equal(t, "user logged in", logEntry["msg"])
		assert.Equal(t, float64(456), logEntry["user_id"])
		assert.Equal(t, "login", logEntry["action"])
	})

	t.Run("WithError", func(t *testing.T) {
		buf.Reset()
		testErr := assert.AnError
		WithError(testErr).Error("operation failed")
		
		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)
		
		assert.Equal(t, "operation failed", logEntry["msg"])
		assert.Equal(t, testErr.Error(), logEntry["error"])
	})
}

func TestGetLogger(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() {
		logger = originalLogger
	}()

	t.Run("GetLoggerWhenNotInitialized", func(t *testing.T) {
		logger = nil
		l := GetLogger()
		assert.NotNil(t, l)
		assert.IsType(t, &logrus.Logger{}, l)
	})

	t.Run("GetLoggerWhenInitialized", func(t *testing.T) {
		cfg := Config{
			Level:  "debug",
			Format: "json",
			Output: "stdout",
		}
		
		err := Init(cfg)
		require.NoError(t, err)
		
		l := GetLogger()
		assert.NotNil(t, l)
		assert.Equal(t, logger, l)
		assert.Equal(t, logrus.DebugLevel, l.Level)
	})
}

func TestLogOutput(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() {
		logger = originalLogger
	}()

	t.Run("StdoutOutput", func(t *testing.T) {
		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		cfg := Config{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		}

		err := Init(cfg)
		require.NoError(t, err)

		Info("test stdout message")

		// Restore stdout
		w.Close()
		os.Stdout = oldStdout

		// Read captured output
		var buf bytes.Buffer
		io.Copy(&buf, r)
		r.Close()

		assert.Contains(t, buf.String(), "test stdout message")
	})

	t.Run("FileOutput", func(t *testing.T) {
		tempDir := t.TempDir()
		logFile := filepath.Join(tempDir, "output_test.log")

		cfg := Config{
			Level:    "info",
			Format:   "text",
			Output:   "file",
			Filename: logFile,
		}

		err := Init(cfg)
		require.NoError(t, err)

		Info("test file message")

		// Read log file
		content, err := os.ReadFile(logFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "test file message")
	})
}

func TestLogFormats(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() {
		logger = originalLogger
	}()

	t.Run("JSONFormat", func(t *testing.T) {
		var buf bytes.Buffer
		
		cfg := Config{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		}

		err := Init(cfg)
		require.NoError(t, err)
		
		logger.SetOutput(&buf)
		Info("json test message")

		// Should be valid JSON
		var logEntry map[string]interface{}
		err = json.Unmarshal(buf.Bytes(), &logEntry)
		assert.NoError(t, err)
		assert.Equal(t, "json test message", logEntry["msg"])
	})

	t.Run("TextFormat", func(t *testing.T) {
		var buf bytes.Buffer
		
		cfg := Config{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		}

		err := Init(cfg)
		require.NoError(t, err)
		
		logger.SetOutput(&buf)
		Info("text test message")

		output := buf.String()
		assert.Contains(t, output, "text test message")
		assert.Contains(t, output, "level=info")
		// Should not be JSON
		var logEntry map[string]interface{}
		err = json.Unmarshal(buf.Bytes(), &logEntry)
		assert.Error(t, err)
	})
}

func TestLogLevelFiltering(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() {
		logger = originalLogger
	}()

	var buf bytes.Buffer

	t.Run("ErrorLevelFiltering", func(t *testing.T) {
		cfg := Config{
			Level:  "error",
			Format: "text",
			Output: "stdout",
		}

		err := Init(cfg)
		require.NoError(t, err)
		logger.SetOutput(&buf)

		buf.Reset()
		Debug("debug message")
		assert.Empty(t, strings.TrimSpace(buf.String()))

		buf.Reset()
		Info("info message")
		assert.Empty(t, strings.TrimSpace(buf.String()))

		buf.Reset()
		Warn("warn message")
		assert.Empty(t, strings.TrimSpace(buf.String()))

		buf.Reset()
		Error("error message")
		assert.Contains(t, buf.String(), "error message")
	})

	t.Run("DebugLevelFiltering", func(t *testing.T) {
		cfg := Config{
			Level:  "debug",
			Format: "text",
			Output: "stdout",
		}

		err := Init(cfg)
		require.NoError(t, err)
		logger.SetOutput(&buf)

		// All levels should be logged
		buf.Reset()
		Debug("debug message")
		assert.Contains(t, buf.String(), "debug message")

		buf.Reset()
		Info("info message")
		assert.Contains(t, buf.String(), "info message")

		buf.Reset()
		Warn("warn message")
		assert.Contains(t, buf.String(), "warn message")

		buf.Reset()
		Error("error message")
		assert.Contains(t, buf.String(), "error message")
	})
}