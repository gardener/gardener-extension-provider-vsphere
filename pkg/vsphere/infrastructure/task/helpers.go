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

package task

import (
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	vapi_errors "github.com/vmware/vsphere-automation-sdk-go/lib/vapi/std/errors"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"
)

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func RandomStringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func RandomString(length int) string {
	return RandomStringWithCharset(length, charset)
}

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

func cidrHostAndPrefix(cidr string, index int) (string, error) {
	host, err := cidrHost(cidr, index)
	if err != nil {
		return "", err
	}
	parts := strings.Split(cidr, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("splitting cidr failed: %s", cidr)
	}
	return fmt.Sprintf("%s/%s", host, parts[1]), nil
}

func containsTags(itemTags []model.Tag, tags []model.Tag) bool {
outer:
	for _, tag := range tags {
		for _, t := range itemTags {
			if *t.Scope == *tag.Scope {
				if *t.Tag == *tag.Tag {
					continue outer
				} else {
					return false
				}
			}
		}
		return false
	}
	return true
}

// mergeTags merges two tag arrays using scope as key
func mergeTags(a []model.Tag, b []model.Tag) []model.Tag {
	result := make([]model.Tag, len(a))
	copy(result, a)
outer:
	for _, tag := range b {
		for _, t := range a {
			if *t.Scope == *tag.Scope {
				continue outer
			}
		}
		result = append(result, tag)
	}
	return result
}

func IdFromPath(path string) string {
	parts := strings.Split(path, "/")
	id := parts[len(parts)-1]
	return id
}

func TagsToMap(tags []model.Tag) map[string]string {
	tagmap := map[string]string{}
	for _, tag := range tags {
		tagmap[*tag.Scope] = *tag.Tag
	}
	return tagmap
}
