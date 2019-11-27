/*
 * Copyright 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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
 *
 */

package main

import (
	goflag "flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	"k8s.io/kubernetes/cmd/cloud-controller-manager/app"
	_ "k8s.io/kubernetes/pkg/util/prometheusclientgo" // for client metric registration
	_ "k8s.io/kubernetes/pkg/version/prometheus"      // for version metric registration

	lbprovider "github.com/gardener/nsxt-lb-provider/pkg/nsxt-lb-provider"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/klog"
)

const (
	AppName = "nsxt-lb-provider"
)

var version = "version-unknown"

func main() {
	lbprovider.Version = version
	lbprovider.AppName = AppName

	rand.Seed(time.Now().UTC().UnixNano())

	command := app.NewCloudControllerManagerCommand()

	// TODO: once we switch everything over to Cobra commands, we can go back to calling
	// utilflag.InitFlags() (by removing its pflag.Parse() call). For now, we have to set the
	// normalize func and add the go flag set by hand.
	pflag.CommandLine.SetNormalizeFunc(flag.WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	// utilflag.InitFlags()
	logs.InitLogs()
	defer logs.FlushLogs()

	klog.V(1).Infof("%s version: %s", AppName, version)

	// Set cloud-provider flag to vsphere
	var versionFlag *pflag.Value
	var clusterNameFlag *pflag.Value
	command.Flags().VisitAll(func(flag *pflag.Flag) {
		switch flag.Name {
		case "cloud-provider":
			_ = flag.Value.Set(lbprovider.ProviderName)
			flag.DefValue = lbprovider.ProviderName
		case "cluster-name":
			clusterNameFlag = &flag.Value
		case "version":
			versionFlag = &flag.Value
		}
	})

	command.Use = AppName
	innerRun := command.Run
	command.Run = func(cmd *cobra.Command, args []string) {
		if versionFlag != nil && (*versionFlag).String() != "false" {
			fmt.Printf("%s %s\n", AppName, version)
			os.Exit(0)
		}
		if clusterNameFlag != nil {
			lbprovider.ClusterName = (*clusterNameFlag).String()
		} else {
			panic("cluster-name flag not found")
		}
		innerRun(cmd, args)
	}

	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
