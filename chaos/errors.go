/*
 *
 *  * Licensed to the Apache Software Foundation (ASF) under one
 *  * or more contributor license agreements.  See the NOTICE file
 *  * distributed with this work for additional information
 *  * regarding copyright ownership.  The ASF licenses this file
 *  * to you under the Apache License, Version 2.0 (the
 *  * "License"); you may not use this file except in compliance
 *  * with the License.  You may obtain a copy of the License at
 *  *
 *  *     http://www.apache.org/licenses/LICENSE-2.0
 *  *
 *  * Unless required by applicable law or agreed to in writing, software
 *  * distributed under the License is distributed on an "AS IS" BASIS,
 *  * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  * See the License for the specific language governing permissions and
 *  * limitations under the License.
 *
 *
 */

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
	ScenarioNotFound
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
	ScenarioNotFound:   "scenario not found",
}

type responseError struct {
	code int
}

func (err responseError) Error(a ...any) string {
	return fmt.Sprintf(errorMsgMap[err.code], a...)
}
