package sugar

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var sugaredLogger *zap.SugaredLogger

const (
	jsonEncoding    = "json"
	consoleEncoding = "console"
)

func init() {

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder

	// build zap.config
	logConfig := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
		Encoding:         jsonEncoding,
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		DisableCaller:    true,
	}

	logger, err := logConfig.Build()
	if err != nil {
		panic(fmt.Sprintf("logger init error: %v", err))
	}

	// 我们使用sugar log
	sugaredLogger = logger.Sugar()
}

func Info(args ...any) {
	sugaredLogger.Infoln(args...)
}

func Infof(tmpl string, args ...any) {
	sugaredLogger.Infof(tmpl, args...)
}

func Warn(args ...any) {
	sugaredLogger.Warnln(args...)
}

func Error(args ...any) {
	sugaredLogger.Errorln(args...)
}

func Errorf(tmpl string, args ...any) {
	sugaredLogger.Errorf(tmpl, args...)
}

func Debug(args ...any) {
	sugaredLogger.Debugln(args...)
}

func Debugf(tmpl string, args ...any) {
	sugaredLogger.Debugf(tmpl, args...)
}
