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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Helpers", func() {
	cidr := "10.0.96.0/20"
	DescribeTable("#cidrHost", func(index int, expectedHost string, expectedSuccess bool) {
		host, err := cidrHost(cidr, index)
		if expectedSuccess {
			Expect(err).To(BeNil())
			Expect(host).To(Equal(expectedHost))
		} else {
			Expect(err).NotTo(BeNil())
		}
	},
		Entry("0", 0, "10.0.96.0", true),
		Entry("1", 1, "10.0.96.1", true),
		Entry("2", 2, "10.0.96.2", true),
		Entry("256", 256, "10.0.97.0", true),
		Entry("second last", -2, "10.0.111.254", true),
		Entry("last-neg", -1, "10.0.111.255", true),
		Entry("last-pos", 4095, "10.0.111.255", true),
		Entry("out of bounds", 4096, "", false),
	)
	Describe("#cidrHostAndPrefix", func() {
		It("should build host with prefix", func() {
			result, err := cidrHostAndPrefix("10.0.96.0/19", 1)
			Expect(err).To(BeNil())
			Expect(result).To(Equal("10.0.96.1/19"))
		})
	})
	Describe("#RandomString", func() {
		It("should generate random strings", func() {
			s1 := RandomString(16)
			s2 := RandomString(16)
			Expect(len(s1)).To(Equal(16))
			Expect(s1).NotTo(Equal(s2))
		})
	})
})
