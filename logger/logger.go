package logger

import (
	"context"
	"os"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger *zap.Logger
)

func SetupLog() {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "time"
	encoderCfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02T15:04:05")
	encoderCfg.MessageKey = "message"
	encoderCfg.CallerKey = zapcore.OmitKey

	fileEncoder := zapcore.NewJSONEncoder(encoderCfg)
	consoleEncoder := zapcore.NewConsoleEncoder(encoderCfg)

	logFile, _ := os.OpenFile("application.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	writer := zapcore.AddSync(logFile)
	defaultLogLevel := zapcore.DebugLevel
	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, writer, defaultLogLevel),
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), defaultLogLevel),
	)
	logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
}

func init() {
	SetupLog()
}

type LoggerWithCtx struct {
	*zap.Logger
	context *context.Context
}

func Ctx(ctx context.Context) *LoggerWithCtx {
	return &LoggerWithCtx{
		Logger:  logger,
		context: &ctx,
	}
}

func (l *LoggerWithCtx) logFields(
	ctx context.Context, fields []zap.Field,
) []zap.Field {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		context := span.SpanContext()
		spanField := zap.String("span_id", context.SpanID().String())
		traceField := zap.String("trace_id", context.TraceID().String())
		traceFlags := zap.Int("trace_flags", int(context.TraceFlags()))
		fields = append(fields, []zap.Field{spanField, traceField, traceFlags}...)
	}

	return fields
}

func (log *LoggerWithCtx) Info(msg string, fields ...zap.Field) {
	fieldsWithTraceCtx := log.logFields(*log.context, fields)
	log.Logger.Info(msg, fieldsWithTraceCtx...)
}

func (log *LoggerWithCtx) Warn(msg string, fields ...zap.Field) {
	fieldsWithTraceCtx := log.logFields(*log.context, fields)
	log.Logger.Warn(msg, fieldsWithTraceCtx...)
}

func (log *LoggerWithCtx) Error(msg string, fields ...zap.Field) {
	fieldsWithTraceCtx := log.logFields(*log.context, fields)
	log.Logger.Error(msg, fieldsWithTraceCtx...)
}

func (log *LoggerWithCtx) Fatal(msg string, fields ...zap.Field) {
	fieldsWithTraceCtx := log.logFields(*log.context, fields)
	log.Logger.Fatal(msg, fieldsWithTraceCtx...)
}
