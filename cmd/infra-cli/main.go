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

package main

import (
	"io/ioutil"
	"os"

	"github.com/gardener/gardener/extensions/pkg/log"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	vspherelog "github.com/vmware/vsphere-automation-sdk-go/runtime/log"

	"k8s.io/cloud-provider-vsphere/pkg/cloudprovider/vsphere"

	infra_cli "github.com/gardener/gardener-extension-provider-vsphere/pkg/cmd/infra-cli"
	"github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure"
)

var (
	// Used for flags.
	stateFile        string
	outputStateFile  string
	specFile         string
	cfgFile          string
	clusterName      string
	ipPoolName       string
	owner            string
	kubeconfig       string
	cloudProfileName string
	region           string
	outputConfigFile string

	config *infrastructure.NSXTConfig

	logger logr.Logger

	rootCmd = &cobra.Command{
		Use:   "vsphere-infra-cli",
		Short: "vSphere provider cli tools",
	}
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file with NSX-T configuration")
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "kubeconfig of virtual garden")
	rootCmd.PersistentFlags().StringVar(&cloudProfileName, "cloudprofile", "", "name of the vSphere cloud profile")
	rootCmd.PersistentFlags().StringVar(&region, "region", "", "region of the vSphere cloud profile")

	destroyCmd := &cobra.Command{
		Use:   "destroy",
		Short: "destroys NSX-T infrastructure as given by state",
		Long: `Destroys the NSX-T infrastructure as created with the infrastructure controlller. 
				You need either to provide the state  or the infrastructure spec as used to create it.
				You can retrieve the state from the infrastructure object in the control plane namespace. 
				Run from the shell
				k -n shoot--<foo>--<bar> get infra <bar> -ojson | jq -r '.status.providerStatus.nsxtInfraState'`,
	}
	destroyCmd.Flags().StringVar(&stateFile, "state", "", "file with infrastructure state as json")
	destroyCmd.Flags().StringVar(&specFile, "spec", "", "file with infrastructure spec as json")
	destroyCmd.Run = destroyInfra
	rootCmd.AddCommand(destroyCmd)

	destroyLBCmd := &cobra.Command{
		Use:   "destroy-loadbalancers",
		Short: "destroys NSX-T load balancers",
		Long: `Destroys the NSX-T load balancers as created with the vSphere cloud-controlller-manager. 
               It uses the cleanup functionality and needs the cluster name, IP pool name and owner tag`,
	}
	destroyLBCmd.Flags().StringVar(&clusterName, "clusterName", "", "cluster name as tagged in NSX-T load balancer objects")
	destroyLBCmd.Flags().StringVar(&ipPoolName, "ipPoolName", "", "IP pool name")
	destroyLBCmd.Flags().StringVar(&owner, "owner", "", "owner tag")
	destroyLBCmd.Run = destroyLoadBalancers
	rootCmd.AddCommand(destroyLBCmd)

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "create NSX-T infrastructure as given by spec",
		Long: `Creates the NSX-T infrastructure the same way as it would with the infrastructure controlller. 
				You need to provide the infrastructure spec.`,
	}
	createCmd.Flags().StringVar(&specFile, "spec", "", "file with infrastructure spec as json")
	createCmd.Flags().StringVar(&outputStateFile, "outputState", "", "filename to store the state")
	createCmd.Run = createInfra
	rootCmd.AddCommand(createCmd)

	createConfigFileCmd := &cobra.Command{
		Use:   "createConfigFile",
		Short: "create config file with NSX-T configuration by reading from cloud profile",
		Long:  `Creates the NSX-T config file by reading the cloud profile and secret on the virtual garden.`,
	}
	createConfigFileCmd.Flags().StringVar(&outputConfigFile, "outputConfigFile", "", "filename to store the config file")
	createConfigFileCmd.Run = createConfigFile
	rootCmd.AddCommand(createConfigFileCmd)
}

func initConfig() {
	if cfgFile == "" {
		if kubeconfig == "" || cloudProfileName == "" {
			panic("missing config file (or alternatively provide kubeconfig and cloudprofile name)")
		}
		initConfigFromVirtualGarden()
		return
	}

	cfgContent, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		panic(err)
	}

	config = &infrastructure.NSXTConfig{}
	err = yaml.Unmarshal([]byte(cfgContent), config)
	if err != nil {
		panic(errors.Wrapf(err, "unmarshalling config file failed"))
	}
}

func initConfigFromVirtualGarden() {
	if kubeconfig == "" {
		panic("missing kubeconfig")
	}
	if cloudProfileName == "" {
		panic("missing cloudprofile name")
	}
	cfg, err := infra_cli.BuildConfigFile(kubeconfig, cloudProfileName, region)
	if err != nil {
		panic(errors.Wrapf(err, "BuildConfigFile failed"))
	}
	config = cfg
}

func destroyInfra(cmd *cobra.Command, args []string) {
	var stateString *string
	var specString *string

	if stateFile == "" && specFile == "" {
		panic("either stateFile or specFile is needed for destroy")
	}
	if stateFile != "" {
		state, err := ioutil.ReadFile(stateFile)
		if err != nil {
			panic(errors.Wrapf(err, "cannot read state file: %s", stateFile))
		}
		s := string(state)
		stateString = &s
	}
	if specFile != "" {
		spec, err := ioutil.ReadFile(specFile)
		if err != nil {
			panic(errors.Wrapf(err, "cannot read spec file: %s", specFile))
		}
		s := string(spec)
		specString = &s
	}
	resultingState, err := infra_cli.DestroyInfrastructure(logger, config, stateString, specString)
	if resultingState != nil {
		err2 := saveFile(stateFile, *resultingState)
		if err2 != nil {
			logger.Error(err2, "saving state failed, please save it yourself from the console output!")
		}
	}
	if err != nil {
		panic(errors.Wrapf(err, "DestroyInfrastructure failed"))
	}
}

func createInfra(cmd *cobra.Command, args []string) {
	if specFile == "" {
		panic("missing infrastructure spec needed for create")
	}
	spec, err := ioutil.ReadFile(specFile)
	if err != nil {
		panic(errors.Wrapf(err, "cannot read spec file: %s", specFile))
	}
	if outputStateFile == "" {
		panic("missing outputState filename needed to store the state")
	}
	resultingState, err := infra_cli.CreateInfrastructure(logger, config, string(spec))
	if resultingState != nil {
		err2 := saveFile(outputStateFile, *resultingState)
		if err2 != nil {
			logger.Error(err2, "saving state failed, please save it yourself from the console output!")
		}
	}
	if err != nil {
		panic(errors.Wrapf(err, "CreateInfrastructure failed"))
	}
}

func createConfigFile(cmd *cobra.Command, args []string) {
	if outputConfigFile == "" {
		panic("missing output config file name")
	}
	bytes, err := yaml.Marshal(&config)
	if err != nil {
		panic(err)
	}
	err = saveFile(outputConfigFile, string(bytes))
	if err != nil {
		panic(err)
	}
	logger.Info("written config file to " + outputConfigFile)
}

func saveFile(filename, contents string) error {
	if fileExists(filename) {
		err := os.Rename(filename, filename+".bak")
		if err != nil {
			return err
		}
	}
	return ioutil.WriteFile(filename, []byte(contents), 0644)
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func destroyLoadBalancers(cmd *cobra.Command, args []string) {
	if clusterName == "" {
		panic("missing clusterName for destroying load balancers")
	}
	if ipPoolName == "" {
		panic("missing ipPoolName for destroying load balancers")
	}
	if owner == "" {
		panic("missing owner for destroying load balancers")
	}
	err := infra_cli.DestroyLoadBalancers(config, clusterName, ipPoolName, owner)
	if err != nil {
		panic(errors.Wrapf(err, "DestroyInfrastructure failed"))
	}
}

func main() {
	logger = log.ZapLogger(false)
	vspherelog.SetLogger(vsphere.NewKlogBridge())

	err := rootCmd.Execute()
	if err != nil {
		panic(err)
	}
}
