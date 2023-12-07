package core

import (
	"github.com/sirupsen/logrus"
	"path"
	"runtime"
	"strconv"
)

func InitLogrus() {
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	customFormatter.CallerPrettyfier = func(frame *runtime.Frame) (function string, file string) {
		fileName := path.Base(frame.File) + ":" + strconv.Itoa(frame.Line)
		return "", fileName
	}
	logrus.SetFormatter(customFormatter)
	logrus.SetReportCaller(true)
}
