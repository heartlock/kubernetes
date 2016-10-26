/*
Copyright 2014 The Kubernetes Authors All rights reserved.

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

package neutron

import (
	"errors"
	"net"

	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"k8s.io/kubernetes/pkg/apis/componentconfig"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/network"
	"k8s.io/kubernetes/pkg/kubelet/network/neutron/types"
	utilsets "k8s.io/kubernetes/pkg/util/sets"
)

const (
	pluginName = "neutron"
)

type NeutronNetworkPlugin struct {
	host      network.Host
	podClient types.PodsClient
	//	TODO (heartlock)add dbclient for getNetworkOfNamespace
}

func NewNeutronNetworkPlugin(addr string) *NeutronNetworkPlugin {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		glog.Errorf("Connect network provider %s failed: %v", addr, err)
		return nil
	}

	podClient := types.NewPodsClient(conn)
	return &NeutronNetworkPlugin{
		podClient: podClient,
	}
}

// Init initializes the plugin.  This will be called exactly once
// before any other methods are called.
func (plugin *NeutronNetworkPlugin) Init(host network.Host, hairpinMode componentconfig.HairpinMode, nonMasqueradeCIDR string) error {
	// TODO(harryz) hairpinMode & nonMasqueradeCIDR is not supported for now
	plugin.host = host
	return nil
}

/*func (plugin *NeutronNetworkPlugin) getNetworkOfNamespace(nsName string) (*types.Network, error) {
	// get namespace info
	namespace, err := plugin.client.Core().Namespaces().Get(nsName)
	if err != nil {
		glog.Errorf("Couldn't get info of namespace %s: %v", nsName, err)
		return nil, err
	}
	if namespace.Spec.Network == "" {
		glog.Warningf("There is no network associated with namespace %s", nsName)
		return nil, nil
	}

	// get network of namespace
	network, err := plugin.client.Core().Networks().Get(namespace.Spec.Network)
	if err != nil {
		glog.Errorf("Couldn't get network for namespace %s: %v", namespace.Name, err)
		return nil, err
	}

	var networkInfo *types.Network
	if network.Spec.ProviderNetworkID != "" {
		networkInfo, err = plugin.provider.Networks().GetNetworkByID(network.Spec.ProviderNetworkID)
	} else {
		networkName := networkprovider.BuildNetworkName(network.Name, network.Spec.TenantID)
		networkInfo, err = plugin.provider.Networks().GetNetwork(networkName)
	}
	if err != nil {
		glog.Errorf("Couldn't get network info from networkprovider: %v", err)
		return nil, err
	}

	return networkInfo, nil
}*/

// Name returns the plugin's name. This will be used when searching
// for a plugin by name, e.g.
func (plugin *NeutronNetworkPlugin) Name() string {
	return pluginName
}

func (plugin *NeutronNetworkPlugin) Capabilities() utilsets.Int {

	return nil
}

// SetUpPod is the method called after the infra container of
// the pod has been created but before the other containers of the
// pod are launched.
func (plugin *NeutronNetworkPlugin) SetUpPod(namespace string, name string, podInfraContainerID kubecontainer.ContainerID, containerRuntime string) error {
	/*network, err := plugin.getNetworkOfNamespace(namespace)
	if err != nil {
		glog.Errorf("GetNetworkOfNamespace failed: %v", err)
		return err
	}

	if network == nil {
		glog.V(4).Infof("Network of namespace %s is nil, do nothing", namespace)
		return nil
	}*/

	network := types.Network{
		Uid:      "d9aa68d1-5e77-4e47-b4b6-0b56fe85c2af",
		TenantID: "a2581b5b574e4115afb6db4d7d704fb7",
	}

	resp, err := plugin.podClient.SetupPod(
		context.Background(),
		&types.SetupPodRequest{
			PodName:             name,
			Namespace:           namespace,
			PodInfraContainerID: podInfraContainerID.ID,
			ContainerRuntime:    containerRuntime,
			Network:             network,
		},
	)
	if err != nil || resp.Error != "" {
		if err == nil {
			err = errors.New(resp.Error)
		}
		glog.Warningf("neutron SetupPod %s failed: %v", name, err)
		return err
	}

	return nil
}

// TearDownPod is the method called before a pod's infra container will be deleted
func (plugin *NeutronNetworkPlugin) TearDownPod(namespace string, name string, podInfraContainerID kubecontainer.ContainerID, containerRuntime string) error {
	/*network, err := plugin.getNetworkOfNamespace(namespace)
	if err != nil {
		glog.Errorf("GetNetworkOfNamespace failed: %v", err)
		return err
	}

	if network == nil {
		glog.V(4).Infof("Network of namespace %s is nil, do nothing", namespace)
		return nil
	}*/

	resp, err := plugin.podClient.TeardownPod(
		context.Background(),
		&types.TeardownPodRequest{
			PodName:             name,
			Namespace:           namespace,
			PodInfraContainerID: podInfraContainerID.ID,
			ContainerRuntime:    containerRuntime,
			//Network:             network,
		},
	)
	if err != nil || resp.Error != "" {
		if err == nil {
			err = errors.New(resp.Error)
		}
		glog.Warningf("NetworkProvider TeardownPod %s failed: %v", name, err)
		return err
	}

	return nil
}

func (plugin *NeutronNetworkPlugin) GetPodNetworkStatus(namespace string, name string, podInfraContainerID kubecontainer.ContainerID, containerRuntime string) (*network.PodNetworkStatus, error) {
	/*networkInfo, err := plugin.getNetworkOfNamespace(namespace)
	if err != nil {
		glog.Errorf("GetNetworkOfNamespace failed: %v", err)
		return nil, err
	}

	if networkInfo == nil {
		glog.V(4).Infof("Network of namespace %s is nil, do nothing", namespace)
		return nil, nil
	}*/

	resp, err := plugin.podClient.PodStatus(
		context.Background(),
		&types.PodStatusRequest{
			PodName:             name,
			Namespace:           namespace,
			PodInfraContainerID: podInfraContainerID.ID,
			ContainerRuntime:    containerRuntime,
			//Network:             network,
		},
	)
	if err != nil || resp.Error != "" {
		if err == nil {
			err = errors.New(resp.Error)
		}
		glog.Warningf("NetworkProvider TeardownPod %s failed: %v", name, err)
		return nil, err
	}
	status := network.PodNetworkStatus{
		IP: net.ParseIP(resp.Ip),
	}

	return &status, nil
}

func (plugin *NeutronNetworkPlugin) Status() error {
	// TODO is there any way to detect plugin is ready?
	return nil
}

func (plugin *NeutronNetworkPlugin) Event(name string, details map[string]interface{}) {
}
