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

package worker

import "context"

func (w *workerDelegate) DeployMachineDependencies(_ context.Context) error {
	return nil
}

func (w *workerDelegate) CleanupMachineDependencies(ctx context.Context) error {
	return nil
}

func (w *workerDelegate) PreReconcileHook(context.Context) error {
	return nil
}

func (w *workerDelegate) PostReconcileHook(context.Context) error {
	return nil
}

func (w *workerDelegate) PreDeleteHook(context.Context) error {
	return nil
}

func (w *workerDelegate) PostDeleteHook(context.Context) error {
	return nil
}
