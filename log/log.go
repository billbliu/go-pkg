package log

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/billbliu/go-pkg/constant"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger 默认日志组件使用zap
type Logger interface {
	Debug(v ...interface{})
	Info(v ...interface{})
	Warn(v ...interface{})
	Error(v ...interface{})
	Debugf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
	Warnf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	// Printf(format string, v ...interface{})
	Fatalf(format string, v ...interface{})
	// DebugContextf(ctx context.Context, format string, v ...interface{})
	// ErrorContextf(ctx context.Context, format string, v ...interface{})
	// WarnContextf(ctx context.Context, format string, v ...interface{})
	// InfoContextf(ctx context.Context, format string, v ...interface{})
}

var (
	logger Logger
)

func Init(opts ...Option) {
	// 初始化
	logger = newSugarLogger(newOptions(opts...))
}

// Options 选项配置
type Options struct {
	logPath    string // 日志路径
	fileName   string // 日志名称
	logLevel   string // 日志级别
	maxSize    int    // 日志保留大小，以 M 为单位
	maxBackups int    // 日志保留文件个数
	maxAge     int    // 日志最大保留天数
	console    bool   // 日志是否输出到控制台
}

// Option 选项方法
type Option func(*Options)

// newOptions 初始化
func newOptions(opts ...Option) Options {
	// 默认配置
	options := Options{
		fileName:   "bitstorm.log",
		logLevel:   "info",
		maxSize:    100,
		maxBackups: 3,
		maxAge:     30,
		console:    false, // 默认不会输出到控制台
	}
	for _, opt := range opts {
		opt(&options)
	}
	return options
}

// WithLogLevel 日志级别
func WithLogLevel(level string) Option {
	return func(o *Options) {
		o.logLevel = level
	}
}

func WithFileName(fileName string) Option {
	return func(o *Options) {
		o.fileName = fileName
	}
}

func WithLogPath(logPath string) Option {
	return func(o *Options) {
		o.logPath = logPath
	}
}

func WithMaxSize(maxSize int) Option {
	return func(o *Options) {
		o.maxSize = maxSize
	}
}

func WithMaxBackups(maxBackups int) Option {
	return func(o *Options) {
		o.maxBackups = maxBackups
	}
}

func WithMaxAge(maxAge int) Option {
	return func(o *Options) {
		o.maxAge = maxAge
	}
}

func WithConsole(console bool) Option {
	return func(o *Options) {
		o.console = console
	}
}

type zapLoggerWrapper struct {
	*zap.SugaredLogger
	options Options
}

func newSugarLogger(options Options) *zapLoggerWrapper {
	w := &zapLoggerWrapper{options: options}
	encoder := w.getEncoder()
	w.setSugaredLogger(encoder)
	return w
}
func (w *zapLoggerWrapper) setSugaredLogger(encoder zapcore.Encoder) {
	// core切片
	var cores []zapcore.Core

	errPriority := zap.LevelEnablerFunc(func(lev zapcore.Level) bool { // error级别
		// 大于等于 error 级别才会打印
		return lev >= zap.ErrorLevel
	})

	appPriority := zap.LevelEnablerFunc(func(lev zapcore.Level) bool {
		// 小于等于 fatalLevel 级别都会打印
		return lev <= zap.FatalLevel
	})
	appWriter := w.getLogWriter("app-")
	appLogFileCore := zapcore.NewCore(encoder, appWriter, appPriority)

	errorWriter := w.getLogWriter("error-")
	errorFileCore := zapcore.NewCore(encoder, errorWriter, errPriority)

	cores = append(cores, appLogFileCore)
	cores = append(cores, errorFileCore)
	zapLogger := zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddCallerSkip(1)) // zap.AddCaller()为显示文件名和行号，可省略
	w.SugaredLogger = zapLogger.Sugar()
}

func (w *zapLoggerWrapper) getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	// 如何编码时间戳
	// encoderConfig.EncodeTime = CustomTimeEncoder
	// encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	// 在日志文件中使用大写字母记录日志级别
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	// NewConsoleEncoder 打印更符合人们观察的方式
	return zapcore.NewConsoleEncoder(encoderConfig)
}
func (w *zapLoggerWrapper) getLogWriter(typeName string) zapcore.WriteSyncer {
	// 设置日志文件轮转配置
	logFile := &lumberjack.Logger{
		Filename:   fmt.Sprintf("%s/%s%s", w.options.logPath, typeName, w.options.fileName), // 日志文件路径
		MaxSize:    w.options.maxSize,                                                       // 每个日志文件的最大大小（以MB为单位）
		MaxBackups: w.options.maxBackups,                                                    // 保留旧日志文件的最大数量
		MaxAge:     w.options.maxAge,                                                        // 保留旧日志文件的最大天数
		Compress:   true,                                                                    // 是否压缩旧的日志文件
		LocalTime:  true,                                                                    // 日志轮换文件的时间戳是否使用本地时间(默认UTC时间)
	}

	if w.options.console {
		// 既输出到控制台又存到日志文件
		return zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(logFile))
	}

	return zapcore.NewMultiWriteSyncer(zapcore.AddSync(logFile))
}

// GetDefaultLogger 获取默认日志实现（注意，先初始化Log）
func GetDefaultLogger() Logger {
	return logger

}

func Debug(v ...interface{}) {
	GetDefaultLogger().Debug(v...)
}
func Info(v ...interface{}) {
	GetDefaultLogger().Info(v...)
}
func Warn(v ...interface{}) {
	GetDefaultLogger().Warn(v...)
}
func Error(v ...interface{}) {
	GetDefaultLogger().Error(v...)
}
func Debugf(format string, v ...interface{}) {
	GetDefaultLogger().Debugf(format, v...)
}
func Errorf(format string, v ...interface{}) {
	GetDefaultLogger().Errorf(format, v...)
}
func Warnf(format string, v ...interface{}) {
	GetDefaultLogger().Warnf(format, v...)
}
func Infof(format string, v ...interface{}) {
	GetDefaultLogger().Infof(format, v...)
}
func Fatalf(format string, v ...interface{}) {
	GetDefaultLogger().Fatalf(format, v...)
}

func DebugContextf(ctx context.Context, format string, v ...interface{}) {
	traceID := ""
	if traceValue := ctx.Value(constant.TraceID); traceValue != nil {
		traceID = traceValue.(string)
	}
	v = append([]interface{}{traceID}, v...)
	GetDefaultLogger().Debugf(constant.TraceID+":%s, "+format, v...)

}
func ErrorContextf(ctx context.Context, format string, v ...interface{}) {
	traceID := ""
	if traceValue := ctx.Value(constant.TraceID); traceValue != nil {
		traceID = traceValue.(string)
	}
	v = append([]interface{}{traceID}, v...)
	// 获取调用者的信息
	_, file, _, _ := runtime.Caller(1)
	callerFile := filepath.Base(file)

	// 判断是否为 GORM 调用者的文件
	if callerFile == "gormlog.go" {
		// 获取 GORM 调用者的信息
		_, file, line, _ := runtime.Caller(4)
		callerFile = filepath.Base(file)
		GetDefaultLogger().Infof("caller is %v:%v", callerFile, line)
	}
	GetDefaultLogger().Errorf(constant.TraceID+":%s, "+format, v...)
}
func WarnContextf(ctx context.Context, format string, v ...interface{}) {
	traceID := ""
	if traceValue := ctx.Value(constant.TraceID); traceValue != nil {
		traceID = traceValue.(string)
	}
	v = append([]interface{}{traceID}, v...)
	GetDefaultLogger().Warnf(constant.TraceID+":%s, "+format, v...)
}
func InfoContextf(ctx context.Context, format string, v ...interface{}) {
	traceID := ""
	if traceValue := ctx.Value(constant.TraceID); traceValue != nil {
		traceID = traceValue.(string)
	}
	v = append([]interface{}{traceID}, v...)

	// 获取调用者的信息
	_, file, _, _ := runtime.Caller(1)
	callerFile := filepath.Base(file)

	// 判断是否为 GORM 调用者的文件
	if callerFile == "gormlog.go" {
		// 获取 GORM 调用者的信息
		_, file, line, _ := runtime.Caller(4)
		callerFile = filepath.Base(file)
		GetDefaultLogger().Infof("caller is %v:%v", callerFile, line)
	}

	GetDefaultLogger().Infof(constant.TraceID+":%s, "+format, v...)
}

// 自定义日志输出时间格式
func CustomTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}
