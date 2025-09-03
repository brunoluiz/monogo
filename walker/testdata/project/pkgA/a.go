package pkgA

import (
	"go.uber.org/zap"
	"test/project/pkgB"
)

func PkgA() {
	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()
	sugar.Infow("failed to fetch URL",
		"url", "http://example.com",
		"attempt", 3,
	)

	pkgB.PkgB()
}
