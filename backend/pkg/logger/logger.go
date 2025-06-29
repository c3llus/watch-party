package logger

import (
	"io"
	"os"
	"watch-party/pkg/config"

	zl "github.com/rs/zerolog"
	zll "github.com/rs/zerolog/log"
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

	var engine zl.Logger
	if cfg.Log.Format == "json" {
		engine = newJSONLogger(opts)
	} else {
		engine = newLogger(opts)
	}

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

func newLogger(opts options) zl.Logger {
	var wr []io.Writer
	zll.Logger = zll.Output(zl.Logger{})

	wr = append(wr, zl.ConsoleWriter{Out: os.Stdout})
	mw := io.MultiWriter(wr...)

	return zl.New(mw).With().Timestamp().Logger()
}

// newJSONLogger creates a logger that outputs JSON format (better for cloud environments)
func newJSONLogger(opts options) zl.Logger {
	return zl.New(os.Stdout).With().Timestamp().Logger()
}
