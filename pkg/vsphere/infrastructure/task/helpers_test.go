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
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"

	vinfra "github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure"
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
	Describe("#containsTags", func() {
		It("should check for tags", func() {
			tags1 := []model.Tag{
				{Scope: sp("owner"), Tag: sp("o1")},
				{Scope: sp("cluster"), Tag: sp("c1")},
			}
			tags2 := []model.Tag{
				{Scope: sp("owner"), Tag: sp("o1")},
				{Scope: sp("cluster"), Tag: sp("c2")},
			}
			itemTags := []model.Tag{
				{Scope: sp("owner"), Tag: sp("o1")},
				{Scope: sp("cluster"), Tag: sp("c1")},
				{Scope: sp("foo"), Tag: sp("bla")},
			}
			Expect(containsTags(tags1, tags1)).To(Equal(true))
			Expect(containsTags(itemTags, tags1)).To(Equal(true))
			Expect(containsTags(itemTags, tags2)).To(Equal(false))
			Expect(containsTags(tags1, itemTags)).To(Equal(false))
		})
	})
	Describe("#mergeTags", func() {
		It("should merge tags", func() {
			tags1 := []model.Tag{
				{Scope: sp("owner"), Tag: sp("o1")},
				{Scope: sp("cluster"), Tag: sp("c1")},
			}
			tags2 := []model.Tag{
				{Scope: sp("owner"), Tag: sp("o1")},
				{Scope: sp("cluster"), Tag: sp("c2")},
			}
			itemTags := []model.Tag{
				{Scope: sp("owner"), Tag: sp("o1")},
				{Scope: sp("cluster"), Tag: sp("c1")},
				{Scope: sp("foo"), Tag: sp("bla")},
				{Scope: sp("bar"), Tag: sp("xxx")},
			}
			Expect(mergeTags(tags1, tags1)).To(Equal(tags1))
			Expect(mergeTags(tags1, tags2)).To(Equal(tags1))
			Expect(mergeTags(tags2, tags1)).To(Equal(tags2))
			Expect(mergeTags(itemTags, tags1)).To(Equal(itemTags))
			Expect(mergeTags(tags1, itemTags)).To(Equal(itemTags))
		})
	})
	Describe("#IdFromPath", func() {
		It("should extract id from path", func() {
			Expect(IdFromPath("/infra/lb-services/60b86e75-41a0-474b-ab5d-ef46d3e1b25b")).To(Equal("60b86e75-41a0-474b-ab5d-ef46d3e1b25b"))
		})
	})
})

var _ = Describe("TaskHelper", func() {
	Describe("#CheckShootAuthorizationByTags", func() {
		It("should handle shoot authorization value correctly", func() {
			tags := map[string]string{vinfra.ScopeAuthorizedShoots: "shoot--foo--bar1,shoot--myns--x*x", vinfra.ScopeGarden: "garden1"}
			err := CheckShootAuthorizationByTags(nil, "IP pool", "mypool", "shoot--foo--bar1", "garden1", tags)
			Expect(err).NotTo(HaveOccurred())
			err = CheckShootAuthorizationByTags(nil, "IP pool", "mypool", "shoot--myns--xfoox", "garden1", tags)
			Expect(err).NotTo(HaveOccurred())

			err = CheckShootAuthorizationByTags(nil, "IP pool", "mypool", "shoot--foo--bar1", "garden2", tags)
			Expect(err).To(HaveOccurred())
			err = CheckShootAuthorizationByTags(nil, "IP pool", "mypool", "shoot--foo--bar", "garden1", tags)
			Expect(err).To(HaveOccurred())
			err = CheckShootAuthorizationByTags(nil, "IP pool", "mypool", "shoot--foo--bar2", "garden1", tags)
			Expect(err).To(HaveOccurred())
			err = CheckShootAuthorizationByTags(nil, "IP pool", "mypool", "shoot--myns--foo", "garden1", tags)
			Expect(err).To(HaveOccurred())
			err = CheckShootAuthorizationByTags(nil, "IP pool", "mypool", "shoot--myns--xfoo", "garden1", tags)
			Expect(err).To(HaveOccurred())
			err = CheckShootAuthorizationByTags(nil, "IP pool", "mypool", "shoot--myns--xfooxz", "garden1", tags)
			Expect(err).To(HaveOccurred())
		})
	})
})

func sp(s string) *string {
	return &s
}
