package chaos

import (
	"fmt"
	"github.com/sirupsen/logrus"
)

func ErrorResponse(code int, err error, a ...any) Response {
	if err != nil {
		logrus.Errorln(err)
	}
	respErr := responseError{code: code}

	return Response{
		Status:  errorStatus,
		Message: respErr.Error(a...),
	}
}

const (
	RequestError = iota
	YamlMarshalError
	YamlUnmarshalError
	JsonMarshalError
	MySqlSaveError
	MySqlDataNotFound
	MySqlError
	ReadFileError
	InvalidScenario
	ChaosJobRunError
)

var errorMsgMap = map[int]string{
	RequestError:       "request error",
	YamlMarshalError:   "yaml marshal error",
	YamlUnmarshalError: "yaml unmarshal error",
	MySqlSaveError:     "failed to save to mysql",
	JsonMarshalError:   "json marshal error",
	MySqlDataNotFound:  "data not found in database",
	MySqlError:         "mysql error",
	ReadFileError:      "read file error",
	InvalidScenario:    "invalid scenario",
	ChaosJobRunError:   "chaos job run failed",
}

type responseError struct {
	code int
}

func (err responseError) Error(a ...any) string {
	return fmt.Sprintf(errorMsgMap[err.code], a...)
}
