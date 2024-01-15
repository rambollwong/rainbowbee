package log

import "github.com/rambollwong/rainbowlog"

var (
	DefaultLoggerLabel = "RAINBOW-BEE"
	Logger             *rainbowlog.Logger
)

func InitLogger(rootLogger *rainbowlog.Logger) {
	Logger = rootLogger.SubLogger(rainbowlog.WithLabels(DefaultLoggerLabel))
}
