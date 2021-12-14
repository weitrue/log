package log

import (
    "os"
    "sync/atomic"

    "github.com/weitrue/log/field"
)
var     DefaultName  = "root"
var g atomic.Value
func init(){
    cfg := NewDevelopmentConfig(os.Stdout)
    cfg.Name = DefaultName
    cfg.ID = DefaultName
    // 创建默认的 全局 日志调用对象，root
    // 默认配置为 Production
    cfg.Name = DefaultName
    cfg.ID = DefaultName
    // 一般来说不会失败
    l,_ := New(cfg)

    UpdateGlobalLog(l)
}

func getGlobalLog()ILogger{
    return g.Load().(ILogger)
}

// UpdateGlobalLogWithILogger 通过接口对象更新
func UpdateGlobalLogWithILogger(l ILogger) {
    g.Store(l)
}

// UpdateGlobalLog 更新 全局 默认 logger
func UpdateGlobalLog(l *Logger) {
    UpdateGlobalLogWithILogger(l)
}
type logFunc func(msg string, fields ...field.Field)

func Debug(msg string, fields ...field.Field){
    getGlobalLog().Debug(msg,fields...)
}
func Info(msg string, fields ...field.Field){
    getGlobalLog().Info(msg,fields...)
}
func Warn(msg string, fields ...field.Field){
    getGlobalLog().Warn(msg,fields...)
}
func Error(msg string, fields ...field.Field){
    getGlobalLog().Error(msg,fields...)
}
func Critical(msg string, fields ...field.Field){
    getGlobalLog().Critical(msg,fields...)
}
func Fixed(msg string, fields ...field.Field){
    getGlobalLog().Fixed(msg,fields...)
}

func Sync() error{
    return getGlobalLog().Sync()
}
func With(fields ...field.Field) *Logger{
    return getGlobalLog().With(fields...)
}