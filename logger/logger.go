// Copyright 2024. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var globalLogger *zap.Logger

// Init initializes the global logger
func Init() error {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	logger, err := config.Build()
	if err != nil {
		return err
	}

	globalLogger = logger
	return nil
}

// GetLogger returns the global logger instance
func GetLogger() *zap.Logger {
	if globalLogger == nil {
		// Fallback to development logger if not initialized
		devLogger, _ := zap.NewDevelopment()
		return devLogger
	}
	return globalLogger
}

// LogChannelEvent logs an event with channel, payload, message, and success fields
func LogChannelEvent(channel, payload, message string, success bool) {
	fields := []zap.Field{
		zap.String("channel", channel),
		zap.String("payload", payload),
		zap.String("message", message),
		zap.Bool("success", success),
	}

	if success {
		GetLogger().Info(message, fields...)
	} else {
		GetLogger().Error(message, fields...)
	}
}

// Info logs an info message with optional fields
func Info(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

// Error logs an error message with optional fields
func Error(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

// ErrorWithErr logs an error message with an error and optional fields
func ErrorWithErr(msg string, err error, fields ...zap.Field) {
	allFields := make([]zap.Field, 0, len(fields)+1)
	allFields = append(allFields, fields...)
	allFields = append(allFields, zap.Error(err))
	GetLogger().Error(msg, allFields...)
}

// Debug logs a debug message with optional fields
func Debug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

// Warn logs a warning message with optional fields
func Warn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...zap.Field) {
	GetLogger().Fatal(msg, fields...)
}
