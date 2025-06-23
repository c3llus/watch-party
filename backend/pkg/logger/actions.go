package logger

import (
	"fmt"

	"watch-party/pkg/utils"
)

// Debug logs a debug message
func Debug(message string) {
	log.engine.Debug().Str(lineOfCode, utils.GetFileAndLoC(1)).Msg(message)
}

// Debugf logs a debug message given a template and arguments
func Debugf(template string, args ...interface{}) {
	log.engine.Debug().Str(lineOfCode, utils.GetFileAndLoC(1)).Msg(
		fmt.Sprintf(template, args...),
	)
}

// Info logs an info message
func Info(message string) {
	log.engine.Info().Str(lineOfCode, utils.GetFileAndLoC(1)).Msg(message)
}

// Infof logs an info message given a template and arguments
func Infof(template string, args ...interface{}) {
	log.engine.Info().Str(lineOfCode, utils.GetFileAndLoC(1)).Msg(
		fmt.Sprintf(template, args...),
	)
}

// Warn logs a warning message
func Warn(message string) {
	log.engine.Warn().Str(lineOfCode, utils.GetFileAndLoC(1)).Msg(message)
}

// Warnf logs a warning message given a template and arguments
func Warnf(template string, args ...interface{}) {
	log.engine.Warn().Str(lineOfCode, utils.GetFileAndLoC(1)).Msg(
		fmt.Sprintf(template, args...),
	)
}

// Error logs an error message with the line of code where the log is called
func Error(err error, message string) {
	if err != nil {
		log.engine.Err(err).Str(lineOfCode, utils.GetFileAndLoC(1)).Msg(message)
		return
	}

	log.engine.Error().Str(lineOfCode, utils.GetFileAndLoC(1)).Msg(message)
}

func Errorf(err error, template string, args ...interface{}) {
	if err != nil {
		log.engine.Err(err).Str(lineOfCode, utils.GetFileAndLoC(1)).Msg(
			fmt.Sprintf(template, args...),
		)
		return
	}

	log.engine.Error().Str(lineOfCode, utils.GetFileAndLoC(1)).Msg(
		fmt.Sprintf(template, args...),
	)
}

func Fatalf(template string, args ...interface{}) {
	log.engine.Fatal().Str(lineOfCode, utils.GetFileAndLoC(1)).Msg(
		fmt.Sprintf(template, args...),
	)
}
