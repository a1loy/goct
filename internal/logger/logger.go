package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var ZapLog *zap.Logger
var SugarLog *zap.SugaredLogger
var enableDebug = os.Getenv("DEBUG") == "true"

func init() {
	var err error
	level := zapcore.InfoLevel
	if enableDebug {
		level = zapcore.DebugLevel
	}
	config := zap.Config{
		Encoding:    "console",
		Level:       zap.NewAtomicLevelAt(level),
		OutputPaths: []string{"stdout"},
		// TODO: change to ProductionDevelopmentConfig ?
		EncoderConfig: zap.NewDevelopmentEncoderConfig(),
	}
	ZapLog, err = config.Build()
	defer func() {
		err = ZapLog.Sync()
	}()
	if err != nil {
		panic(err)
	}
	SugarLog = ZapLog.Sugar()
}

func Infof(message string, args ...interface{}) {
	SugarLog.Infof(message, args...)
}

func Debugf(message string, args ...interface{}) {
	SugarLog.Debugf(message, args...)
}

func Errorf(message string, args ...interface{}) {
	SugarLog.Errorf(message, args...)
}

func Fatalf(message string, args ...interface{}) {
	SugarLog.Fatalf(message, args...)
}
