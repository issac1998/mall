package log

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logger *logrus.Logger
)

// Config log configuration
type Config struct {
	Level      string `json:"level"`      // debug, info, warn, error
	Format     string `json:"format"`     // json, text
	Output     string `json:"output"`     // stdout, file
	Filename   string `json:"filename"`   // log file path
	MaxSize    int    `json:"max_size"`   // maximum size of a single file (MB)
	MaxAge     int    `json:"max_age"`    // maximum number of days to keep files
	MaxBackups int    `json:"max_backups"` // maximum number of backup files
	Compress   bool   `json:"compress"`   // whether to compress
}

// Init initialize logger
func Init(cfg Config) error {
	logger = logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set log format
	if cfg.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}

	// Set output
	var output io.Writer = os.Stdout
	if cfg.Output == "file" && cfg.Filename != "" {
		// Ensure log directory exists
		if err := os.MkdirAll(filepath.Dir(cfg.Filename), 0755); err != nil {
			return err
		}

		// Use lumberjack for log rotation
		output = &lumberjack.Logger{
			Filename:   cfg.Filename,
			MaxSize:    cfg.MaxSize,
			MaxAge:     cfg.MaxAge,
			MaxBackups: cfg.MaxBackups,
			Compress:   cfg.Compress,
		}
	}
	logger.SetOutput(output)

	return nil
}

// GetLogger get logger instance
func GetLogger() *logrus.Logger {
	if logger == nil {
		logger = logrus.New()
	}
	return logger
}

// Debug output debug log
func Debug(args ...interface{}) {
	GetLogger().Debug(args...)
}

// Debugf formatted output debug log
func Debugf(format string, args ...interface{}) {
	GetLogger().Debugf(format, args...)
}

// Info output info log
func Info(args ...interface{}) {
	GetLogger().Info(args...)
}

// Infof formatted output info log
func Infof(format string, args ...interface{}) {
	GetLogger().Infof(format, args...)
}

// Warn output warning log
func Warn(args ...interface{}) {
	GetLogger().Warn(args...)
}

// Warnf formatted output warning log
func Warnf(format string, args ...interface{}) {
	GetLogger().Warnf(format, args...)
}

// Error output error log
func Error(args ...interface{}) {
	GetLogger().Error(args...)
}

// Errorf formatted output error log
func Errorf(format string, args ...interface{}) {
	GetLogger().Errorf(format, args...)
}

// Fatal output fatal error log and exit program
func Fatal(args ...interface{}) {
	GetLogger().Fatal(args...)
}

// Fatalf formatted output fatal error log and exit program
func Fatalf(format string, args ...interface{}) {
	GetLogger().Fatalf(format, args...)
}

// Panic output panic log and trigger panic
func Panic(args ...interface{}) {
	GetLogger().Panic(args...)
}

// Panicf formatted output panic log and trigger panic
func Panicf(format string, args ...interface{}) {
	GetLogger().Panicf(format, args...)
}

// WithField add field
func WithField(key string, value interface{}) *logrus.Entry {
	return GetLogger().WithField(key, value)
}

// WithFields add multiple fields
func WithFields(fields logrus.Fields) *logrus.Entry {
	return GetLogger().WithFields(fields)
}

// WithError add error field
func WithError(err error) *logrus.Entry {
	return GetLogger().WithError(err)
}