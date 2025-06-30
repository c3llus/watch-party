package logger

import (
	"os"
	"watch-party/pkg/config"

	zl "github.com/rs/zerolog"
)

// log is a unexported package-level global variable that holds julo-go-library logger instance
var log *logger

type logger struct {
	engine *zl.Logger
}

type options struct {
}

// InitLogger initializes the logger with configuration
func InitLogger(cfg *config.Config) {
	logLvl := getLogLevel(cfg.Log.Level)

	opts := options{}

	zl.SetGlobalLevel(logLvl)
	setupCloudLoggingSeverity()
	engine := newGCPLogger(opts) // TODO: support other cloud prviders

	log = &logger{
		engine: &engine,
	}
}

// getLogLevel returns the log level based on the string input
func getLogLevel(level string) zl.Level {
	switch level {
	case DebugLevel:
		return zl.DebugLevel
	case InfoLevel:
		return zl.InfoLevel
	case WarnLevel:
		return zl.WarnLevel
	case ErrorLevel:
		return zl.ErrorLevel
	default:
		return zl.InfoLevel
	}
}

// setupCloudLoggingSeverity configures zerolog to use Cloud Logging severity levels
func setupCloudLoggingSeverity() {
	zl.LevelFieldMarshalFunc = func(l zl.Level) string {
		switch l {
		case zl.DebugLevel:
			return "DEBUG"
		case zl.InfoLevel:
			return "INFO"
		case zl.WarnLevel:
			return "WARNING"
		case zl.ErrorLevel:
			return "ERROR"
		case zl.FatalLevel:
			return "CRITICAL"
		case zl.PanicLevel:
			return "CRITICAL"
		default:
			return "DEFAULT"
		}
	}
}

// newGCPLogger creates a logger that outputs JSON format (better for cloud environments)
func newGCPLogger(opts options) zl.Logger {
	// for Google Cloud Loggin structured logging, we need to use specific field names
	zl.TimeFieldFormat = zl.TimeFormatUnix
	zl.TimestampFieldName = "timestamp"
	zl.LevelFieldName = "severity"
	zl.MessageFieldName = "message"

	// Add caller information for better debugging in Cloud Logging
	return zl.New(os.Stdout).With().
		Timestamp().
		Caller().
		Logger()
}
