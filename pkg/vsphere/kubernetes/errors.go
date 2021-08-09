/*
 * Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package kubernetes

var _ error = &NotFoundError{}
var _ error = &AlreadyExistsError{}

// NewNotFoundError creates a NotFoundError
func NewNotFoundError(msg string) error {
	return &NotFoundError{msg: msg}
}

// NotFoundError is a not found error
type NotFoundError struct {
	msg string
}

// Error implements error message
func (e *NotFoundError) Error() string {
	return e.msg
}

// IsNotFoundError checks if err is a NotFoundError
func IsNotFoundError(err error) bool {
	_, ok := err.(*NotFoundError)
	return ok
}

// NewAlreadyExistsError creates a AlreadyExistsError
func NewAlreadyExistsError(msg string) error {
	return &AlreadyExistsError{msg: msg}
}

// AlreadyExistsError is a already exists error
type AlreadyExistsError struct {
	msg string
}

// Error implements error message
func (e *AlreadyExistsError) Error() string {
	return e.msg
}

// IsAlreadyExistsError checks if err is a AlreadyExistsError
func IsAlreadyExistsError(err error) bool {
	_, ok := err.(*AlreadyExistsError)
	return ok
}
