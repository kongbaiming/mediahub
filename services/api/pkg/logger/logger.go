// Package logger 封装 zerolog，提供结构化日志
package logger

import (
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
)

var log zerolog.Logger

// Init 初始化全局 logger
//   - level: debug | info | warn | error
//   - pretty: 是否使用 console 编码（开发用）
func Init(level string, pretty bool) {
	zerolog.TimeFieldFormat = time.RFC3339Nano

	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)

	if pretty {
		log = zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}).With().Timestamp().Logger()
	} else {
		logDir := "logs"
		_ = os.MkdirAll(logDir, 0755)
		logFile := filepath.Join(logDir, "mediahub.log")
		f, _ := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)

		if f != nil {
			log = zerolog.New(f).With().Timestamp().Logger()
		} else {
			log = zerolog.New(os.Stdout).With().Timestamp().Logger()
		}
	}
}

// Sync 刷新缓冲（zerolog 默认无缓冲，保留接口）
func Sync() {
	// no-op
}

// fieldsToMap 把 key-value 切片转换成 map（用于 zerolog）
// 支持的 value 类型：string、int、int64、float64、bool、error、any
func fieldsToMap(fields []any) map[string]any {
	m := make(map[string]any, len(fields)/2)
	for i := 0; i+1 < len(fields); i += 2 {
		if k, ok := fields[i].(string); ok {
			m[k] = fields[i+1]
		}
	}
	return m
}

// Info 信息日志
// 用法：logger.Info("msg", "key1", val1, "key2", val2)
func Info(msg string, fields ...any) {
	log.Info().Fields(fieldsToMap(fields)).Msg(msg)
}

// Warn 警告日志
func Warn(msg string, fields ...any) {
	log.Warn().Fields(fieldsToMap(fields)).Msg(msg)
}

// Error 错误日志
func Error(msg string, fields ...any) {
	log.Error().Fields(fieldsToMap(fields)).Msg(msg)
}

// Fatal 致命错误
func Fatal(msg string, fields ...any) {
	log.Fatal().Fields(fieldsToMap(fields)).Msg(msg)
}

// Debug 调试日志
func Debug(msg string, fields ...any) {
	log.Debug().Fields(fieldsToMap(fields)).Msg(msg)
}
