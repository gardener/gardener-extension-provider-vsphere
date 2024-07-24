/*
 * Copyright 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 *
 */

package app

import (
	"context"
	"fmt"
	"os"

	druidv1alpha1 "github.com/gardener/etcd-druid/api/v1alpha1"
	"github.com/gardener/gardener/extensions/pkg/controller"
	controllercmd "github.com/gardener/gardener/extensions/pkg/controller/cmd"
	"github.com/gardener/gardener/extensions/pkg/controller/controlplane/genericactuator"
	"github.com/gardener/gardener/extensions/pkg/controller/heartbeat"
	heartbeatcmd "github.com/gardener/gardener/extensions/pkg/controller/heartbeat/cmd"
	"github.com/gardener/gardener/extensions/pkg/util"
	webhookcmd "github.com/gardener/gardener/extensions/pkg/webhook/cmd"
	gardenerhealthz "github.com/gardener/gardener/pkg/healthz"
	machinev1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	autoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/component-base/version/verflag"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	vsphereinstall "github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/install"
	vspherecmd "github.com/gardener/gardener-extension-provider-vsphere/pkg/cmd"
	vspherecontrolplane "github.com/gardener/gardener-extension-provider-vsphere/pkg/controller/controlplane"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/controller/healthcheck"
	vsphereinfrastructure "github.com/gardener/gardener-extension-provider-vsphere/pkg/controller/infrastructure"
	vsphereworker "github.com/gardener/gardener-extension-provider-vsphere/pkg/controller/worker"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere"
	providervsphere "github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere"
	vspherecontrolplaneexposure "github.com/gardener/gardener-extension-provider-vsphere/pkg/webhook/controlplaneexposure"
)

// NewControllerManagerCommand creates a new command for running a vSphere provider controller.
func NewControllerManagerCommand(ctx context.Context) *cobra.Command {
	var (
		generalOpts = &controllercmd.GeneralOptions{}
		restOpts    = &controllercmd.RESTOptions{}
		mgrOpts     = &controllercmd.ManagerOptions{
			LeaderElection:          true,
			LeaderElectionID:        controllercmd.LeaderElectionNameID(vsphere.Name),
			LeaderElectionNamespace: os.Getenv("LEADER_ELECTION_NAMESPACE"),
			WebhookServerPort:       443,
			MetricsBindAddress:      ":8080",
			HealthBindAddress:       ":8081",
		}
		configFileOpts = &vspherecmd.ConfigOptions{}

		// options for the infrastructure controller
		infraCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}
		reconcileOpts = &controllercmd.ReconcilerOptions{}

		// options for the health care controller
		healthCareCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}

		// options for the heartbeat controller
		heartbeatCtrlOpts = &heartbeatcmd.Options{
			ExtensionName:        vsphere.Name,
			RenewIntervalSeconds: 30,
			Namespace:            os.Getenv("LEADER_ELECTION_NAMESPACE"),
		}

		// options for the control plane controller
		controlPlaneCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}

		// options for the worker controller
		workerCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}

		// options for the webhook server
		webhookServerOptions = &webhookcmd.ServerOptions{
			Namespace: os.Getenv("WEBHOOK_CONFIG_NAMESPACE"),
		}

		controllerSwitches = vspherecmd.ControllerSwitchOptions()
		webhookSwitches    = vspherecmd.WebhookSwitchOptions()
		webhookOptions     = webhookcmd.NewAddToManagerOptions(
			vsphere.Name,
			genericactuator.ShootWebhooksResourceName,
			genericactuator.ShootWebhookNamespaceSelector(vsphere.Type),
			webhookServerOptions,
			webhookSwitches,
		)

		aggOption = controllercmd.NewOptionAggregator(
			generalOpts,
			restOpts,
			mgrOpts,
			controllercmd.PrefixOption("controlplane-", controlPlaneCtrlOpts),
			controllercmd.PrefixOption("infrastructure-", infraCtrlOpts),
			controllercmd.PrefixOption("worker-", workerCtrlOpts),
			controllercmd.PrefixOption("healthcheck-", healthCareCtrlOpts),
			controllercmd.PrefixOption("heartbeat-", heartbeatCtrlOpts),
			controllerSwitches,
			configFileOpts,
			reconcileOpts,
			webhookOptions,
		)
	)

	cmd := &cobra.Command{
		Use: fmt.Sprintf("validator-%s", providervsphere.Type),

		RunE: func(cmd *cobra.Command, args []string) error {
			verflag.PrintAndExitIfRequested()

			if err := aggOption.Complete(); err != nil {
				return fmt.Errorf("error completing options: %w", err)
			}

			if err := heartbeatCtrlOpts.Validate(); err != nil {
				return err
			}

			util.ApplyClientConnectionConfigurationToRESTConfig(configFileOpts.Completed().Config.ClientConnection, restOpts.Completed().Config)

			mgr, err := manager.New(restOpts.Completed().Config, mgrOpts.Completed().Options())
			if err != nil {
				return fmt.Errorf("could not instantiate manager: %w", err)
			}

			scheme := mgr.GetScheme()
			if err := controller.AddToScheme(scheme); err != nil {
				return fmt.Errorf("could not update manager scheme: %w", err)
			}
			if err := vsphereinstall.AddToScheme(scheme); err != nil {
				return fmt.Errorf("could not update manager scheme: %w", err)
			}
			if err := druidv1alpha1.AddToScheme(scheme); err != nil {
				return fmt.Errorf("could not update manager scheme: %w", err)
			}
			if err := machinev1alpha1.AddToScheme(scheme); err != nil {
				return fmt.Errorf("could not update manager scheme: %w", err)
			}
			if err := autoscalingv1.AddToScheme(scheme); err != nil {
				return fmt.Errorf("could not update manager scheme: %w", err)
			}

			// add common meta types to schema for controller-runtime to use v1.ListOptions
			metav1.AddToGroupVersion(scheme, machinev1alpha1.SchemeGroupVersion)

			configFileOpts.Completed().ApplyETCDStorage(&vspherecontrolplaneexposure.DefaultAddOptions.ETCDStorage)
			configFileOpts.Completed().ApplyGardenId(&vspherecontrolplane.DefaultAddOptions.GardenId)
			configFileOpts.Completed().ApplyGardenId(&vsphereinfrastructure.DefaultAddOptions.GardenId)
			configFileOpts.Completed().ApplyHealthCheckConfig(&healthcheck.DefaultAddOptions.HealthCheckConfig)
			healthCareCtrlOpts.Completed().Apply(&healthcheck.DefaultAddOptions.Controller)
			heartbeatCtrlOpts.Completed().Apply(&heartbeat.DefaultAddOptions)
			controlPlaneCtrlOpts.Completed().Apply(&vspherecontrolplane.DefaultAddOptions.Controller)
			infraCtrlOpts.Completed().Apply(&vsphereinfrastructure.DefaultAddOptions.Controller)
			reconcileOpts.Completed().Apply(&vsphereinfrastructure.DefaultAddOptions.IgnoreOperationAnnotation)
			reconcileOpts.Completed().Apply(&vspherecontrolplane.DefaultAddOptions.IgnoreOperationAnnotation)
			reconcileOpts.Completed().Apply(&vsphereworker.DefaultAddOptions.IgnoreOperationAnnotation)
			workerCtrlOpts.Completed().Apply(&vsphereworker.DefaultAddOptions.Controller)

			atomicShootWebhookConfig, err := webhookOptions.Completed().AddToManager(ctx, mgr, nil)
			if err != nil {
				return fmt.Errorf("could not add webhooks to manager: %w", err)
			}
			vspherecontrolplane.DefaultAddOptions.ShootWebhookConfig = atomicShootWebhookConfig
			vspherecontrolplane.DefaultAddOptions.WebhookServerNamespace = webhookOptions.Server.Namespace

			if err := controllerSwitches.Completed().AddToManager(ctx, mgr); err != nil {
				return fmt.Errorf("could not add controllers to manager: %w", err)
			}

			if err := mgr.AddReadyzCheck("informer-sync", gardenerhealthz.NewCacheSyncHealthz(mgr.GetCache())); err != nil {
				return fmt.Errorf("could not add readycheck for informers: %w", err)
			}

			if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
				return fmt.Errorf("could not add health check to manager: %w", err)
			}

			if err := mgr.AddReadyzCheck("webhook-server", mgr.GetWebhookServer().StartedChecker()); err != nil {
				return fmt.Errorf("could not add ready check for webhook server to manager: %w", err)
			}

			if err := mgr.Start(ctx); err != nil {
				return fmt.Errorf("error running manager: %w", err)
			}

			return nil
		},
	}

	flags := cmd.Flags()
	aggOption.AddFlags(flags)
	verflag.AddFlags(flags)

	return cmd
}
