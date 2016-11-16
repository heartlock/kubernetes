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
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/network"
	"k8s.io/kubernetes/pkg/kubelet/network/neutron/types"
	utilsets "k8s.io/kubernetes/pkg/util/sets"
)

const (
	pluginName = "neutron"
)

type NeutronNetworkPlugin struct {
	host          network.Host
	client        clientset.Interface
	podClient     types.PodsClient
	networkClient types.NetworksClient
	subnetClient  types.SubnetsClient
	//	TODO (heartlock)add dbclient for getNetworkOfNamespace
}

func NewNeutronNetworkPlugin(addr string) *NeutronNetworkPlugin {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		glog.Errorf("Connect network provider %s failed: %v", addr, err)
		return nil
	}
	networkClient := types.NewNetworksClient(conn)
	podClient := types.NewPodsClient(conn)
	subnetClient := types.NewSubnetsClient(conn)
	return &NeutronNetworkPlugin{
		podClient:     podClient,
		networkClient: networkClient,
		subnetClient:  subnetClient,
	}
}

// Init initializes the plugin.  This will be called exactly once
// before any other methods are called.
func (plugin *NeutronNetworkPlugin) Init(host network.Host, hairpinMode componentconfig.HairpinMode, nonMasqueradeCIDR string) error {
	// TODO(harryz) hairpinMode & nonMasqueradeCIDR is not supported for now
	plugin.host = host
	plugin.client = host.GetKubeClient()
	return nil
}

func (plugin *NeutronNetworkPlugin) getNetworkOfPod(nsName, podName string) (*types.Network, error) {
	// get pod info
	pod, err := plugin.client.Core().Pods(nsName).Get(podName)
	if err != nil {
		glog.Errorf("Couldn't get info of pod %s: %v", podName, err)
		return nil, err
	}
	subnetID, ok := pod.ObjectMeta.Annotations["nephele/subnetID"]
	if ok == false {
		glog.Errorf("There is no subnet associated with pod %s", podName)
		err := errors.New("There is no subnet associated with this pod")
		return nil, err
	}
	//get subnet info by subnetID
	subnet, err := plugin.subnetClient.GetSubnet(
		context.Background(),
		&types.GetSubnetRequest{
			SubnetID: subnetID,
		},
	)
	if err != nil {
		glog.Errorf("GetSubnet failed: %v", err)
		return nil, err
	}
	//get network info by networkID
	network, err := plugin.networkClient.GetNetwork(
		context.Background(),
		&types.GetNetworkRequest{
			Id: subnet.Subnet.NetworkID,
		},
	)
	if err != nil {
		glog.Errorf("GetNetwork failed: %v", err)
		return nil, err
	}

	return network.Network, nil
}

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
	// get pod info
	pod, err := plugin.client.Core().Pods(namespace).Get(name)
	if err != nil {
		glog.Errorf("Couldn't get info of pod %s: %v", name, err)
		return err
	}
	subnetID, ok := pod.ObjectMeta.Annotations["nephele/subnetID"]
	if ok == false {
		glog.Errorf("There is no subnet associated with pod %s", name)
		err := errors.New("There is no subnet associated with pod")
		return err
	}
	network, err := plugin.getNetworkOfPod(namespace, name)
	if err != nil {
		glog.Errorf("GetNetworkOfPod failed: %v", err)
		return err
	}
	if network == nil {
		glog.V(4).Infof("Network of pod %s is nil, do nothing", name)
		return nil
	}

	resp, err := plugin.podClient.SetupPod(
		context.Background(),
		&types.SetupPodRequest{
			PodName:             name,
			Namespace:           namespace,
			PodInfraContainerID: podInfraContainerID.ID,
			ContainerRuntime:    containerRuntime,
			Network:             network,
			SubnetID:            subnetID,
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
	network, err := plugin.getNetworkOfPod(namespace, name)
	if err != nil {
		glog.Errorf("GetNetworkOfPod failed: %v", err)
		return err
	}
	if network == nil {
		glog.V(4).Infof("Network of pod %s is nil, do nothing", name)
		return nil
	}

	resp, err := plugin.podClient.TeardownPod(
		context.Background(),
		&types.TeardownPodRequest{
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
		glog.Warningf("NetworkProvider TeardownPod %s failed: %v", name, err)
		return err
	}

	return nil
}

func (plugin *NeutronNetworkPlugin) GetPodNetworkStatus(namespace string, name string, podInfraContainerID kubecontainer.ContainerID, containerRuntime string) (*network.PodNetworkStatus, error) {
	podnetwork, err := plugin.getNetworkOfPod(namespace, name)
	if err != nil {
		glog.Errorf("GetNetworkOfPod failed: %v", err)
		return nil, err
	}
	if podnetwork == nil {
		glog.V(4).Infof("Network of pod %s is nil, do nothing", name)
		return nil, nil
	}
	resp, err := plugin.podClient.PodStatus(
		context.Background(),
		&types.PodStatusRequest{
			PodName:             name,
			Namespace:           namespace,
			PodInfraContainerID: podInfraContainerID.ID,
			ContainerRuntime:    containerRuntime,
			Network:             podnetwork,
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
