package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/natefinch/lumberjack"
	"org.donghyuns.com/mqtt/listner/configs"
)

const (
	INFO = iota // 0
	DEBUG
	TRACE
)

func LogInitialize(logInfo configs.LogConfig) error {
	logLevel := ConvLogLevel(logInfo.Level)

	var logWriter io.Writer
	fileWriter := &lumberjack.Logger{
		Filename:   logInfo.Path,
		MaxSize:    logInfo.MaxSize, // megabytes
		MaxBackups: 10,
		MaxAge:     10,    //days
		Compress:   false, // disabled by default
	}
	// 기본적으로 stdout 도 출력.
	logWriter = io.MultiWriter(os.Stdout, fileWriter)

	// 로그 디렉토리 permission 문제
	err := checkPermission(logInfo.Path)
	if err != nil {
		return err
	}

	handler := slog.NewJSONHandler(logWriter, &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
		ReplaceAttr: func(group []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				source := a.Value.Any().(*slog.Source)
				// path 가 너무 길게 나와서 줄이기
				source.File = filepath.Base(source.File)
				return slog.Any(a.Key, source)
			}
			return a
		},
	})

	logger := slog.New(handler)

	slog.SetDefault(logger)
	return nil
}

func ConvLogLevel(level string) slog.Level {
	// 사실상 warn, error 은 잘 안 쓰긴 하지만 구현은 해두기로
	switch strings.ToUpper(level) {
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	case "INFO":
		return slog.LevelInfo
	case "DEBUG":
		return slog.LevelDebug
	default:
		fmt.Println("LogLevel is not entered normally, so it works as 'INFO'")
		return slog.LevelInfo
	}
}

func checkPermission(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			dir, _ := filepath.Split(path)
			err = os.MkdirAll(dir, 0755)
			if err != nil {
				return err
			}
		} else {
			fmt.Printf("Error checking log directory permissions: %s\n", err)
			return err
		}
	}
	return nil
}
