package logger

import (
	"fmt"

	"go.uber.org/zap"
)

type Logger interface {
	Error(val interface{})
	Errorw(val interface{}, keysAndValues ...interface{})
	Debug(val interface{})
	Debugw(msg string, keysAndValues ...interface{})
	Info(val interface{})
	Infow(msg string, keysAndValues ...interface{})
	Warn(val interface{})
	Warnw(msg string, keysAndValues ...interface{})
	WithMap(m map[string]string) Logger
	Fatal(val interface{})

	zapcore() *zap.Logger
}

type loggerOpts struct {
	encoding string
	level    string
}

type LoggerOption func(opt *loggerOpts)

func WithLevel(level string) LoggerOption {
	return func(opt *loggerOpts) {
		opt.level = level
	}
}

func WithEncoding(encoding string) LoggerOption {
	return func(opt *loggerOpts) {
		opt.encoding = encoding
	}
}

func NewLogger(opts ...LoggerOption) Logger {
	lo := &loggerOpts{}

	for _, opt := range opts {
		opt(lo)
	}

	logger := createLogger(lo.level, lo.encoding)

	return &loggerImpl{
		logcore: logger,
		log:     logger.Sugar(),
	}
}

type loggerImpl struct {
	logcore *zap.Logger
	log     *zap.SugaredLogger
}

func (l *loggerImpl) WithMap(m map[string]string) Logger {
	if len(m) > 0 {
		fields := map2fields(m)
		return &loggerImpl{
			log: l.log.With(fields...),
		}
	}

	return l
}

func (l *loggerImpl) Error(val interface{}) {
	l.Errorw(val)
}

func (l *loggerImpl) Errorw(val interface{}, keysAndValues ...interface{}) {
	msg := ""
	switch v := val.(type) {
	case error:
		msg = fmt.Sprintf("%+v", v)
	case string:
		msg = v
	default:
		msg = fmt.Sprint(v)
	}

	l.log.Errorw(msg, keysAndValues...)
}

func (l *loggerImpl) Debug(val interface{}) {
	l.log.Debug(val)
}

func (l *loggerImpl) Debugw(msg string, keysAndValues ...interface{}) {
	l.log.Debugw(msg, keysAndValues...)
}

func (l *loggerImpl) Info(val interface{}) {
	l.log.Info(val)
}

func (l *loggerImpl) Infow(msg string, keysAndValues ...interface{}) {
	l.log.Infow(msg, keysAndValues...)
}

func (l *loggerImpl) Warn(val interface{}) {
	l.log.Warn(val)
}

func (l *loggerImpl) Warnw(msg string, keysAndValues ...interface{}) {
	l.log.Warnw(msg, keysAndValues...)
}

func (l *loggerImpl) Fatal(val interface{}) {
	l.log.Fatal(val)
}

func (l *loggerImpl) zapcore() *zap.Logger {
	return l.logcore
}

func map2fields(m map[string]string) []interface{} {
	fields := make([]interface{}, 0, len(m))
	for k, v := range m {
		fields = append(fields, zap.Any(k, v))
	}
	return fields
}
