package logger

import (
	"os"
	"time"

	zaplogfmt "github.com/jsternberg/zap-logfmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.SugaredLogger
var logcore *zap.Logger

func init() {
	logcore = createLogger("info")
	logger = logcore.Sugar()
}

// Init initializes logger with specified level
func Init(logLevel string) {
	config := zap.NewProductionEncoderConfig()
	config.TimeKey = "time"
	config.EncodeTime = rfc3339NanoTimeEncoder
	logcore = createLogger(logLevel)
	logger = logcore.Sugar()
}

func createLogger(level string) *zap.Logger {
	config := zap.NewProductionEncoderConfig()
	config.TimeKey = "time"
	config.EncodeTime = rfc3339NanoTimeEncoder
	return zap.New(zapcore.NewCore(
		zaplogfmt.NewEncoder(config),
		os.Stdout,
		parseLevel(level),
	))
}

func rfc3339NanoTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format(time.RFC3339Nano))
}

func parseLevel(level string) zapcore.Level {
	var l zapcore.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		return zapcore.InfoLevel
	}

	return l
}

func Close() {
	if logger != nil {
		defer logger.Sync()
	}
}
