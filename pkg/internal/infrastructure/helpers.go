/*
 * Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package infrastructure

import (
	"fmt"
	"net"

	vapi_errors "github.com/vmware/vsphere-automation-sdk-go/lib/vapi/std/errors"
)

func strptr(s string) *string {
	return &s
}

func isNotFoundError(err error) bool {
	if _, ok := err.(vapi_errors.NotFound); ok {
		return true
	}

	return false
}

func boolptr(b bool) *bool {
	return &b
}

func int64ptr(i int64) *int64 {
	return &i
}

func safeEquals(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func cidrHost(cidr string, index int) (string, error) {
	addr, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", err
	}
	if addr.To4() == nil {
		return "", fmt.Errorf("Not an IPv4 cidr: %s", cidr)
	}
	nip := network.IP
	m := network.Mask

	invert := false
	n := index
	if index < 0 {
		invert = true
		n = -n - 1
	}
	delta := make(net.IP, 4)
	for i := 3; i >= 0; i-- {
		delta[i] = byte(n % 256)
		if delta[i]&m[i] != 0 {
			return "", fmt.Errorf("Index %d out of CIDR range %s", index, cidr)
		}
		n = n / 256
	}
	if invert {
		for i := range delta {
			delta[i] = (delta[i] ^ 255) & (m[i] ^ 255)
		}
	}

	for i := range delta {
		delta[i] = (nip[i] & m[i]) + delta[i]
	}

	return delta.String(), nil
}
