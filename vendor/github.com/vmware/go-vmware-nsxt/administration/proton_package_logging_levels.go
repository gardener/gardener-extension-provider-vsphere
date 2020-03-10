/* Copyright © 2017 VMware, Inc. All Rights Reserved.
   SPDX-License-Identifier: BSD-2-Clause

   Generated by: https://github.com/swagger-api/swagger-codegen.git */

package administration

type ProtonPackageLoggingLevels struct {

	// Logging levels per package
	LoggingLevel string `json:"logging_level,omitempty"`

	// Package name
	PackageName string `json:"package_name,omitempty"`
}