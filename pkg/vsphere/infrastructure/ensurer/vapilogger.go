/*
 * Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package ensurer

import (
	"errors"
	"fmt"

	"github.com/go-logr/logr"
)

type logrBridge struct {
	logger logr.Logger
}

func NewLogrBridge(logger logr.Logger) logrBridge {
	return logrBridge{logger: logger}
}

func (d logrBridge) Error(args ...interface{}) {
	d.logger.Error(errors.New(fmt.Sprint(args...)), "")

}

func (d logrBridge) Errorf(a string, args ...interface{}) {
	d.logger.Error(fmt.Errorf(a, args...), "")
}

func (d logrBridge) Info(args ...interface{}) {
	d.logger.Info(fmt.Sprint(args...))
}

func (d logrBridge) Infof(a string, args ...interface{}) {
	d.logger.Info(fmt.Sprintf(a, args...))
}

func (d logrBridge) Debug(args ...interface{}) {
	d.logger.V(4).Info(fmt.Sprint(args...))
}

func (d logrBridge) Debugf(a string, args ...interface{}) {
	d.logger.V(4).Info(fmt.Sprintf(a, args...))
}
