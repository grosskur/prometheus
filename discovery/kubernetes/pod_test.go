// Copyright 2016 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kubernetes

import (
	"testing"
	"time"

	"github.com/prometheus/common/log"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/tools/cache"
)

func podStoreKeyFunc(obj interface{}) (string, error) {
	return obj.(*v1.Pod).ObjectMeta.Name, nil
}

func newFakePodInformer() *fakeInformer {
	return newFakeInformer(podStoreKeyFunc)
}

func namespaceStoreKeyFunc(obj interface{}) (string, error) {
	return obj.(*v1.Namespace).ObjectMeta.Name, nil
}

func newFakeNamespaceInformer() *fakeInformer {
	return newFakeInformer(namespaceStoreKeyFunc)
}

func makeTestPodDiscovery() (*Pod, *fakeInformer, *fakeInformer, *fakeInformer) {
	i := newFakePodInformer()
	j := newFakeNodeInformer()
	k := newFakeNamespaceInformer()
	return NewPod(log.Base(), i, j, k), i, j, k
}

func makeMultiPortPod() *v1.Pod {
	return &v1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:        "testpod",
			UID:         "e4c35dec-836e-4931-83c6-13b9dbf39b18",
			Namespace:   "default",
			Labels:      map[string]string{"testlabel": "testvalue"},
			Annotations: map[string]string{"testannotation": "testannotationvalue"},
		},
		Spec: v1.PodSpec{
			NodeName: "testnode",
			Containers: []v1.Container{
				{
					Name: "testcontainer0",
					Ports: []v1.ContainerPort{
						{
							Name:          "testport0",
							Protocol:      v1.ProtocolTCP,
							ContainerPort: int32(9000),
						},
						{
							Name:          "testport1",
							Protocol:      v1.ProtocolUDP,
							ContainerPort: int32(9001),
						},
					},
				},
				{
					Name: "testcontainer1",
				},
			},
		},
		Status: v1.PodStatus{
			PodIP:     "1.2.3.4",
			HostIP:    "2.3.4.5",
			StartTime: &unversioned.Time{Time: time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)},
			Conditions: []v1.PodCondition{
				{
					Type:   v1.PodReady,
					Status: v1.ConditionTrue,
				},
			},
		},
	}
}

func makePod() *v1.Pod {
	return &v1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      "testpod",
			UID:       "e4c35dec-836e-4931-83c6-13b9dbf39b18",
			Namespace: "default",
		},
		Spec: v1.PodSpec{
			NodeName: "testnode",
			Containers: []v1.Container{
				{
					Name: "testcontainer",
					Ports: []v1.ContainerPort{
						{
							Name:          "testport",
							Protocol:      v1.ProtocolTCP,
							ContainerPort: int32(9000),
						},
					},
				},
			},
		},
		Status: v1.PodStatus{
			PodIP:     "1.2.3.4",
			HostIP:    "2.3.4.5",
			StartTime: &unversioned.Time{Time: time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)},
			Conditions: []v1.PodCondition{
				{
					Type:   v1.PodReady,
					Status: v1.ConditionTrue,
				},
			},
		},
	}
}

func makeSimpleNode() *v1.Node {
	return &v1.Node{
		ObjectMeta: v1.ObjectMeta{
			Name:      "testnode",
			Namespace: "default",
		},
		Spec: v1.NodeSpec{
			ExternalID: "123456",
			ProviderID: "test://testproject/testzone/testnode",
		},
	}
}

func makeSimpleNamespace() *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name: "default",
			UID:  "12345678-1234-1234-1234-123456789012",
		},
	}
}

func TestPodDiscoveryInitial(t *testing.T) {
	n, i, j, k := makeTestPodDiscovery()
	i.GetStore().Add(makeMultiPortPod())
	j.GetStore().Add(makeSimpleNode())
	k.GetStore().Add(makeSimpleNamespace())

	k8sDiscoveryTest{
		discovery: n,
		expectedInitial: []*config.TargetGroup{
			{
				Targets: []model.LabelSet{
					{
						"__address__":                                   "1.2.3.4:9000",
						"__meta_kubernetes_pod_container_name":          "testcontainer0",
						"__meta_kubernetes_pod_container_port_name":     "testport0",
						"__meta_kubernetes_pod_container_port_number":   "9000",
						"__meta_kubernetes_pod_container_port_protocol": "TCP",
					},
					{
						"__address__":                                   "1.2.3.4:9001",
						"__meta_kubernetes_pod_container_name":          "testcontainer0",
						"__meta_kubernetes_pod_container_port_name":     "testport1",
						"__meta_kubernetes_pod_container_port_number":   "9001",
						"__meta_kubernetes_pod_container_port_protocol": "UDP",
					},
					{
						"__address__":                          "1.2.3.4",
						"__meta_kubernetes_pod_container_name": "testcontainer1",
					},
				},
				Labels: model.LabelSet{
					"__meta_kubernetes_pod_name":                      "testpod",
					"__meta_kubernetes_pod_uid":                       "e4c35dec-836e-4931-83c6-13b9dbf39b18",
					"__meta_kubernetes_pod_start_time":                "1970-01-01 00:00:00 +0000 UTC",
					"__meta_kubernetes_namespace":                     "default",
					"__meta_kubernetes_namespace_uid":                 "12345678-1234-1234-1234-123456789012",
					"__meta_kubernetes_pod_label_testlabel":           "testvalue",
					"__meta_kubernetes_pod_annotation_testannotation": "testannotationvalue",
					"__meta_kubernetes_pod_node_name":                 "testnode",
					"__meta_kubernetes_pod_node_external_id":          "123456",
					"__meta_kubernetes_pod_node_provider_id":          "test://testproject/testzone/testnode",
					"__meta_kubernetes_pod_ip":                        "1.2.3.4",
					"__meta_kubernetes_pod_host_ip":                   "2.3.4.5",
					"__meta_kubernetes_pod_ready":                     "true",
				},
				Source: "pod/default/testpod",
			},
		},
	}.Run(t)
}

func TestPodDiscoveryAdd(t *testing.T) {
	n, i, j, k := makeTestPodDiscovery()
	j.GetStore().Add(makeSimpleNode())
	k.GetStore().Add(makeSimpleNamespace())

	k8sDiscoveryTest{
		discovery:  n,
		afterStart: func() { go func() { i.Add(makePod()) }() },
		expectedRes: []*config.TargetGroup{
			{
				Targets: []model.LabelSet{
					{
						"__address__":                                   "1.2.3.4:9000",
						"__meta_kubernetes_pod_container_name":          "testcontainer",
						"__meta_kubernetes_pod_container_port_name":     "testport",
						"__meta_kubernetes_pod_container_port_number":   "9000",
						"__meta_kubernetes_pod_container_port_protocol": "TCP",
					},
				},
				Labels: model.LabelSet{
					"__meta_kubernetes_pod_name":             "testpod",
					"__meta_kubernetes_pod_uid":              "e4c35dec-836e-4931-83c6-13b9dbf39b18",
					"__meta_kubernetes_pod_start_time":       "1970-01-01 00:00:00 +0000 UTC",
					"__meta_kubernetes_namespace":            "default",
					"__meta_kubernetes_namespace_uid":        "12345678-1234-1234-1234-123456789012",
					"__meta_kubernetes_pod_node_name":        "testnode",
					"__meta_kubernetes_pod_node_external_id": "123456",
					"__meta_kubernetes_pod_node_provider_id": "test://testproject/testzone/testnode",
					"__meta_kubernetes_pod_ip":               "1.2.3.4",
					"__meta_kubernetes_pod_host_ip":          "2.3.4.5",
					"__meta_kubernetes_pod_ready":            "true",
				},
				Source: "pod/default/testpod",
			},
		},
	}.Run(t)
}

func TestPodDiscoveryDelete(t *testing.T) {
	n, i, j, k := makeTestPodDiscovery()
	i.GetStore().Add(makePod())
	j.GetStore().Add(makeSimpleNode())
	k.GetStore().Add(makeSimpleNamespace())

	k8sDiscoveryTest{
		discovery:  n,
		afterStart: func() { go func() { i.Delete(makePod()) }() },
		expectedInitial: []*config.TargetGroup{
			{
				Targets: []model.LabelSet{
					{
						"__address__":                                   "1.2.3.4:9000",
						"__meta_kubernetes_pod_container_name":          "testcontainer",
						"__meta_kubernetes_pod_container_port_name":     "testport",
						"__meta_kubernetes_pod_container_port_number":   "9000",
						"__meta_kubernetes_pod_container_port_protocol": "TCP",
					},
				},
				Labels: model.LabelSet{
					"__meta_kubernetes_pod_name":             "testpod",
					"__meta_kubernetes_pod_uid":              "e4c35dec-836e-4931-83c6-13b9dbf39b18",
					"__meta_kubernetes_pod_start_time":       "1970-01-01 00:00:00 +0000 UTC",
					"__meta_kubernetes_namespace":            "default",
					"__meta_kubernetes_namespace_uid":        "12345678-1234-1234-1234-123456789012",
					"__meta_kubernetes_pod_node_name":        "testnode",
					"__meta_kubernetes_pod_node_external_id": "123456",
					"__meta_kubernetes_pod_node_provider_id": "test://testproject/testzone/testnode",
					"__meta_kubernetes_pod_ip":               "1.2.3.4",
					"__meta_kubernetes_pod_host_ip":          "2.3.4.5",
					"__meta_kubernetes_pod_ready":            "true",
				},
				Source: "pod/default/testpod",
			},
		},
		expectedRes: []*config.TargetGroup{
			{
				Source: "pod/default/testpod",
			},
		},
	}.Run(t)
}

func TestPodDiscoveryDeleteUnknownCacheState(t *testing.T) {
	n, i, j, k := makeTestPodDiscovery()
	i.GetStore().Add(makePod())
	j.GetStore().Add(makeSimpleNode())
	k.GetStore().Add(makeSimpleNamespace())

	k8sDiscoveryTest{
		discovery:  n,
		afterStart: func() { go func() { i.Delete(cache.DeletedFinalStateUnknown{Obj: makePod()}) }() },
		expectedInitial: []*config.TargetGroup{
			{
				Targets: []model.LabelSet{
					{
						"__address__":                                   "1.2.3.4:9000",
						"__meta_kubernetes_pod_container_name":          "testcontainer",
						"__meta_kubernetes_pod_container_port_name":     "testport",
						"__meta_kubernetes_pod_container_port_number":   "9000",
						"__meta_kubernetes_pod_container_port_protocol": "TCP",
					},
				},
				Labels: model.LabelSet{
					"__meta_kubernetes_pod_name":             "testpod",
					"__meta_kubernetes_pod_uid":              "e4c35dec-836e-4931-83c6-13b9dbf39b18",
					"__meta_kubernetes_pod_start_time":       "1970-01-01 00:00:00 +0000 UTC",
					"__meta_kubernetes_namespace":            "default",
					"__meta_kubernetes_namespace_uid":        "12345678-1234-1234-1234-123456789012",
					"__meta_kubernetes_pod_node_name":        "testnode",
					"__meta_kubernetes_pod_node_external_id": "123456",
					"__meta_kubernetes_pod_node_provider_id": "test://testproject/testzone/testnode",
					"__meta_kubernetes_pod_ip":               "1.2.3.4",
					"__meta_kubernetes_pod_host_ip":          "2.3.4.5",
					"__meta_kubernetes_pod_ready":            "true",
				},
				Source: "pod/default/testpod",
			},
		},
		expectedRes: []*config.TargetGroup{
			{
				Source: "pod/default/testpod",
			},
		},
	}.Run(t)
}

func TestPodDiscoveryUpdate(t *testing.T) {
	n, i, j, k := makeTestPodDiscovery()
	i.GetStore().Add(&v1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      "testpod",
			UID:       "e4c35dec-836e-4931-83c6-13b9dbf39b18",
			Namespace: "default",
		},
		Spec: v1.PodSpec{
			NodeName: "testnode",
			Containers: []v1.Container{
				{
					Name: "testcontainer",
					Ports: []v1.ContainerPort{
						{
							Name:          "testport",
							Protocol:      v1.ProtocolTCP,
							ContainerPort: int32(9000),
						},
					},
				},
			},
		},
		Status: v1.PodStatus{
			PodIP:     "1.2.3.4",
			HostIP:    "2.3.4.5",
			StartTime: &unversioned.Time{Time: time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)},
		},
	})
	j.GetStore().Add(makeSimpleNode())
	k.GetStore().Add(makeSimpleNamespace())

	k8sDiscoveryTest{
		discovery:  n,
		afterStart: func() { go func() { i.Update(makePod()) }() },
		expectedInitial: []*config.TargetGroup{
			{
				Targets: []model.LabelSet{
					{
						"__address__":                                   "1.2.3.4:9000",
						"__meta_kubernetes_pod_container_name":          "testcontainer",
						"__meta_kubernetes_pod_container_port_name":     "testport",
						"__meta_kubernetes_pod_container_port_number":   "9000",
						"__meta_kubernetes_pod_container_port_protocol": "TCP",
					},
				},
				Labels: model.LabelSet{
					"__meta_kubernetes_pod_name":             "testpod",
					"__meta_kubernetes_pod_uid":              "e4c35dec-836e-4931-83c6-13b9dbf39b18",
					"__meta_kubernetes_pod_start_time":       "1970-01-01 00:00:00 +0000 UTC",
					"__meta_kubernetes_namespace":            "default",
					"__meta_kubernetes_namespace_uid":        "12345678-1234-1234-1234-123456789012",
					"__meta_kubernetes_pod_node_name":        "testnode",
					"__meta_kubernetes_pod_node_external_id": "123456",
					"__meta_kubernetes_pod_node_provider_id": "test://testproject/testzone/testnode",
					"__meta_kubernetes_pod_ip":               "1.2.3.4",
					"__meta_kubernetes_pod_host_ip":          "2.3.4.5",
					"__meta_kubernetes_pod_ready":            "unknown",
				},
				Source: "pod/default/testpod",
			},
		},
		expectedRes: []*config.TargetGroup{
			{
				Targets: []model.LabelSet{
					{
						"__address__":                                   "1.2.3.4:9000",
						"__meta_kubernetes_pod_container_name":          "testcontainer",
						"__meta_kubernetes_pod_container_port_name":     "testport",
						"__meta_kubernetes_pod_container_port_number":   "9000",
						"__meta_kubernetes_pod_container_port_protocol": "TCP",
					},
				},
				Labels: model.LabelSet{
					"__meta_kubernetes_pod_name":             "testpod",
					"__meta_kubernetes_pod_uid":              "e4c35dec-836e-4931-83c6-13b9dbf39b18",
					"__meta_kubernetes_pod_start_time":       "1970-01-01 00:00:00 +0000 UTC",
					"__meta_kubernetes_namespace":            "default",
					"__meta_kubernetes_namespace_uid":        "12345678-1234-1234-1234-123456789012",
					"__meta_kubernetes_pod_node_name":        "testnode",
					"__meta_kubernetes_pod_node_external_id": "123456",
					"__meta_kubernetes_pod_node_provider_id": "test://testproject/testzone/testnode",
					"__meta_kubernetes_pod_ip":               "1.2.3.4",
					"__meta_kubernetes_pod_host_ip":          "2.3.4.5",
					"__meta_kubernetes_pod_ready":            "true",
				},
				Source: "pod/default/testpod",
			},
		},
	}.Run(t)
}
