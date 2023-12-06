package core

import (
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
)

var LocalDebug = false

func ParseVars() {
	var err error
	if os.Getenv("LOCAL_DEBUG") != "" {
		LocalDebug, err = strconv.ParseBool(os.Getenv("LOCAL_DEBUG"))
		if err != nil {
			logrus.Fatalf("Parse LOCAL_DEBUG error, reason, %s", err.Error())
		}
	}
}
