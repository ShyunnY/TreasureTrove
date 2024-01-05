package sugar

import (
	"fmt"
	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// TODO: 目前我们是使用全局日志, 我们也许要考虑, 是否将日志组件化
var sugaredLogger *zap.SugaredLogger

const (
	jsonEncoding    = "json"
	consoleEncoding = "console"
)

func InitLogger(level ...string) {

	var logLevel string
	if len(level) == 0 {
		logLevel = zap.DebugLevel.String()
	} else {
		logLevel = level[0]
	}

	lev, err := zapcore.ParseLevel(logLevel)
	if err != nil {
		fmt.Printf("parse loglevel error: %v", err)
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder

	// build zap.config
	logConfig := zap.Config{
		Level:             zap.NewAtomicLevelAt(lev),
		Encoding:          jsonEncoding,
		EncoderConfig:     encoderConfig,
		OutputPaths:       []string{"stdout"},
		ErrorOutputPaths:  []string{"stderr"},
		DisableCaller:     true,
		DisableStacktrace: true,
	}

	logger, err := logConfig.Build()
	if err != nil {
		panic(fmt.Sprintf("logger init error: %v", err))
	}

	// 我们使用sugar log
	sugaredLogger = logger.Sugar()

	// TODO: 我们需要实现一个controller-runtime log sink
	log.SetLogger(logr.Discard())
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
