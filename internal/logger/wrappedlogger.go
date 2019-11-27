package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// coreWithLevel is needed as zap does not allow overriding logLevel of existing logger
// see https://github.com/uber-go/zap/issues/581
type coreWithLevel struct {
	zapcore.Core
	level zapcore.Level
}

func (c *coreWithLevel) Enabled(level zapcore.Level) bool {
	return c.level.Enabled(level) && c.Core.Enabled(level)
}

func (c *coreWithLevel) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	// We only need to do the local level check because
	// c.Core will do its own level checking.
	if !c.level.Enabled(e.Level) {
		return ce
	}
	return c.Core.Check(e, ce)
}

func wrapCoreWithLevel(level zapcore.Level) zap.Option {
	return zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return &coreWithLevel{
			Core:  core,
			level: level,
		}
	})
}

func createWrappedLogger(level string) *zap.SugaredLogger {
	if logcore == nil {
		panic("please initialize main logger")
	}
	return logcore.WithOptions(wrapCoreWithLevel(parseLevel(level))).Sugar()
}
