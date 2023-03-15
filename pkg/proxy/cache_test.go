/*
 * Copyright (c) 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *
 *  You may obtain a copy of the License at
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package proxy

import (
	"context"
	"log"
	"testing"
	"time"
)

func TestListClusters(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	creds, err := GetCredentials("")
	if err != nil {
		t.Fatal(err)
	}
	cache, err := NewCache(ctx, creds.ProjectID, creds, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	clusters := cache.GetMap()
	if len(*clusters) == 0 {
		t.Fatalf("expected at least 1 cluster, found none")
	}

	for _, cluster := range *clusters {
		log.Printf("%v", cluster)
	}
}
