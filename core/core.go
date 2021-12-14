package core

import (
    "github.com/weitrue/log/encoder"
    "github.com/weitrue/log/level"
    "github.com/weitrue/log/writer"
    "go.uber.org/zap/zapcore"
)

// NewCore creates a Core that writes logs to a WriteSyncer.
func NewCore(enc encoder.Encoder, ws writer.WriteSyncer, enab level.LevelEnabler) Core {
    if enc == nil || ws == nil{
        return nil
    }
    return zapcore.NewCore(enc,ws,enab)
}

var NewNopCore = zapcore.NewNopCore
var NewTee = zapcore.NewTee
type Core = zapcore.Core
type CheckedEntry = zapcore.CheckedEntry
var RegisterHooks = zapcore.RegisterHooks
var WriteThenPanic = zapcore.WriteThenPanic
var NewEntryCaller = zapcore.NewEntryCaller

