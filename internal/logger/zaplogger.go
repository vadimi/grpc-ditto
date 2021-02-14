package logger

import (
	"strings"

	_ "github.com/jsternberg/zap-logfmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func createLogger(level, encoding string) *zap.Logger {
	zc := zap.NewProductionConfig()
	zc.Encoding = "logfmt"
	if strings.EqualFold(encoding, "json") {
		zc.Encoding = "json"
	}
	zc.DisableCaller = true
	zc.EncoderConfig.TimeKey = "time"
	zc.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	zc.OutputPaths = []string{"stdout"}
	zc.Level = parseLevel(level)
	l, _ := zc.Build()
	return l
}

func parseLevel(level string) zap.AtomicLevel {
	var l zapcore.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		return zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	return zap.NewAtomicLevelAt(l)
}
