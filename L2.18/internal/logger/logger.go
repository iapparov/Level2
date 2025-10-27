package logger

import (
	"calendar/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"path/filepath"
)

func ProvideLogger(cfg *config.Config) (*zap.Logger, error) {
	switch cfg.Env {
	case "prod":
		// путь до файла логов
		logDir := "logs"
		logFile := filepath.Join(logDir, "app.log")

		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, err
		}

		file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}

		writer := zapcore.AddSync(file)

		encoderCfg := zap.NewProductionEncoderConfig()
		encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			writer,
			zap.InfoLevel,
		)

		logger := zap.New(core)
		return logger, nil

	default:
		zapCfg := zap.NewDevelopmentConfig()
		zapCfg.Encoding = "console"
		return zapCfg.Build()
	}
}
