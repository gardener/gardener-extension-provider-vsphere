/* Copyright © 2017 VMware, Inc. All Rights Reserved.
   SPDX-License-Identifier: BSD-2-Clause

   Generated by: https://github.com/swagger-api/swagger-codegen.git */

package manager

import (
	"github.com/vmware/go-vmware-nsxt/common"
)

type NsGroupSimpleExpression struct {
	ResourceType string `json:"resource_type"`

	// Operator of the expression
	Op string `json:"op"`

	// Field of the resource on which this expression is evaluated
	TargetProperty string `json:"target_property"`

	// Reference of the target. Will be populated when the property is a resource id, the op (operator) is EQUALS and populate_references is set to be true.
	TargetResource *common.ResourceReference `json:"target_resource,omitempty"`

	// Type of the resource on which this expression is evaluated
	TargetType string `json:"target_type"`

	// Value that satisfies this expression
	Value string `json:"value"`
}

// List of NSGroupSimpleExpressions
type NsGroupSimpleExpressionList struct {

	// List of NSGroupSimpleExpressions to be passed to add and remove APIs
	Members []NsGroupSimpleExpression `json:"members"`
}