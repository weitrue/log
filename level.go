package log

import "github.com/weitrue/log/level"

var DEBUG = level.DebugLevel
var INFO = level.InfoLevel
var WARN = level.WarnLevel
var ERROR = level.ErrorLevel
var CRITICAL = level.CriticalLevel
var FIXED = level.FixedLevel

var Name2Level = level.Name2Level
var Level2Name = level.Level2Name