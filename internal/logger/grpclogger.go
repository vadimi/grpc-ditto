package logger

import (
	"go.uber.org/zap"
	"google.golang.org/grpc/grpclog"
)

const (
	// internal grpc transport logger logs at verbosity level 2
	// so we need to lower that a bit to avoid showing grpc errors after health check
	verbosityLevel = 1
)

type grpcLogger struct {
	l *zap.SugaredLogger
}

// NewGrpcLogger creates new instance of grpc LoggerV2
func NewGrpcLogger(level string) grpclog.LoggerV2 {
	return &grpcLogger{
		l: createWrappedLogger(level),
	}
}

func (g *grpcLogger) Info(args ...interface{}) {
	g.l.Info(args...)
}

func (g *grpcLogger) Infoln(args ...interface{}) {
	g.l.Info(args...)
}

func (g *grpcLogger) Infof(format string, args ...interface{}) {
	g.l.Infof(format, args...)
}

func (g *grpcLogger) Warning(args ...interface{}) {
	g.l.Warn(args...)
}

func (g *grpcLogger) Warningln(args ...interface{}) {
	g.l.Warn(args...)
}

func (g *grpcLogger) Warningf(format string, args ...interface{}) {
	g.l.Warnf(format, args...)
}

func (g *grpcLogger) Error(args ...interface{}) {
	g.l.Error(args...)
}

func (g *grpcLogger) Errorln(args ...interface{}) {
	g.l.Error(args...)
}

func (g *grpcLogger) Errorf(format string, args ...interface{}) {
	g.l.Errorf(format, args...)
}

func (g *grpcLogger) Fatal(args ...interface{}) {
	g.l.Fatal(args...)
	// No need to call os.Exit() again because log.Logger.Fatal() calls os.Exit().
}

func (g *grpcLogger) Fatalln(args ...interface{}) {
	g.l.Fatal(args...)
}

func (g *grpcLogger) Fatalf(format string, args ...interface{}) {
	g.l.Fatalf(format, args...)
}

func (g *grpcLogger) V(l int) bool {
	return l <= verbosityLevel
}
