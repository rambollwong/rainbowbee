package log

import "github.com/rambollwong/rainbowlog"

var Logger *rainbowlog.Logger

func InitLogger(rootLogger *rainbowlog.Logger) {
	Logger = rootLogger.SubLogger(rainbowlog.WithLabel("RAINBOW-BEE"))
}
