/*
Copyright 2021 The Mizar Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package app

import (
	"fmt"
	"strings"

	"centaurusinfra.io/mizar/pkg/object"
	"centaurusinfra.io/mizar/pkg/util/grpcclientutil"
	"centaurusinfra.io/mizar/pkg/util/netutil"
	"centaurusinfra.io/mizar/pkg/util/netvariablesutil"

	cniTypesVer "github.com/containernetworking/cni/pkg/types/current"
)

func DoInit(netVariables *object.NetVariables) (string, error) {
	netvariablesutil.LoadEnvVariables(netVariables)
	return netvariablesutil.MountNetNSIfNeeded(netVariables)
}

func DoCmdAdd(netVariables *object.NetVariables, stdinData []byte) (cniTypesVer.Result, string, error) {
	tracelog := strings.Builder{}
	result := cniTypesVer.Result{
		CNIVersion: netVariables.CniVersion,
	}

	if err := netvariablesutil.LoadCniConfig(netVariables, stdinData); err != nil {
		return result, tracelog.String(), err
	}
	tracelog.WriteString(fmt.Sprintf("CNI_ADD: Args: '%s'\n", netVariables))

	interfaces, err := grpcclientutil.ConsumeInterfaces(*netVariables)
	if err != nil {
		return result, tracelog.String(), err
	}
	if len(interfaces) == 0 {
		return result, tracelog.String(), fmt.Errorf("No interfaces found for Pod '%s/%s'", netVariables.K8sPodNamespace, netVariables.K8sPodName)
	}

	for index, intf := range interfaces {
		tracelog.WriteString(fmt.Sprintf("CNI_ADD: Activating interface: '%s'\n", intf))
		activateIfLog, err := netutil.ActivateInterface(
			netVariables.IfName,
			netVariables.NetNS,
			intf.Veth.Name,
			intf.Address.IpPrefix,
			intf.Address.IpAddress,
			intf.Address.GatewayIp)
		if activateIfLog != "" {
			tracelog.WriteString(fmt.Sprintf("CNI_ADD: Activate interface result: '%s'\n", activateIfLog))
		}
		if err != nil {
			return result, tracelog.String(), err
		}

		result.Interfaces = append(result.Interfaces, &cniTypesVer.Interface{
			Name:    intf.InterfaceId.Interface,
			Mac:     intf.Address.Mac,
			Sandbox: netVariables.NetNS,
		})

		_, ipnet, err := netutil.ParseCIDR(intf.Address.IpAddress)
		if err != nil {
			return result, tracelog.String(), err
		}
		result.IPs = append(result.IPs, &cniTypesVer.IPConfig{
			Version:   intf.Address.Version,
			Address:   *ipnet,
			Gateway:   netutil.ParseIP(intf.Address.GatewayIp),
			Interface: cniTypesVer.Int(index),
		})
	}

	return result, tracelog.String(), nil
}

func DoCmdDel(netVariables *object.NetVariables, stdinData []byte) (cniTypesVer.Result, string, error) {
	tracelog := strings.Builder{}
	result := cniTypesVer.Result{
		CNIVersion: netVariables.CniVersion,
	}

	if err := netvariablesutil.LoadCniConfig(netVariables, stdinData); err != nil {
		return result, tracelog.String(), err
	}
	tracelog.WriteString(fmt.Sprintf("CNI_DEL: Deleting NetNS: '%s'\n", netVariables.NetNS))
	netutil.DeleteNetNS(netVariables.NetNS)

	return result, tracelog.String(), nil
}
