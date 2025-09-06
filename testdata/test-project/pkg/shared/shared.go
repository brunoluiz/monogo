package shared

import "go.uber.org/zap"

func Log(msg string) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	logger.Info(msg)
}
