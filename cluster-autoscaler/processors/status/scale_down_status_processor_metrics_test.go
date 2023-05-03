/*
Copyright 2018 The Kubernetes Authors.

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

package status

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	schedulerframework "k8s.io/kubernetes/pkg/scheduler/framework"

	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	"k8s.io/autoscaler/cluster-autoscaler/config"
	"k8s.io/autoscaler/cluster-autoscaler/context"
	"k8s.io/autoscaler/cluster-autoscaler/simulator"
	"k8s.io/autoscaler/cluster-autoscaler/utils/drain"
)

func TestScaleDownStatusProcessorMetrics(t *testing.T) {
	type test struct {
		name                   string
		statuses               []*ScaleDownStatus
		nodesCount             []int
		nodesCountLabels       [][]string
		nodesCountByNode       []int
		nodesCountByNodeLabels [][]string
	}

	tests := []test{
		{
			name: "no unremovable nodes",
			statuses: []*ScaleDownStatus{
				{},
			},
			nodesCount:             []int{0},
			nodesCountLabels:       [][]string{{"", "", "", "", ""}},
			nodesCountByNode:       []int{0},
			nodesCountByNodeLabels: [][]string{{"", "", "", "", "", ""}},
		},
		{
			name: "simple unremovable node",
			statuses: []*ScaleDownStatus{
				{
					UnremovableNodes: []*UnremovableNode{
						{
							Node: &apiv1.Node{
								ObjectMeta: metav1.ObjectMeta{
									Name: "node1",
								},
							},
							NodeGroup: &fakeNodeGroup{id: "ng1"},
							Reason:    simulator.NotUnderutilized,
						},
					},
				},
			},
			nodesCount:             []int{1},
			nodesCountLabels:       [][]string{{"8", "ng1", "", "", ""}},
			nodesCountByNode:       []int{1},
			nodesCountByNodeLabels: [][]string{{"8", "ng1", "node1", "", "", ""}},
		},
		{
			name: "simple unremovable nodes",
			statuses: []*ScaleDownStatus{
				{
					UnremovableNodes: []*UnremovableNode{
						{
							Node: &apiv1.Node{
								ObjectMeta: metav1.ObjectMeta{
									Name: "node1",
								},
							},
							NodeGroup: &fakeNodeGroup{id: "ng1"},
							Reason:    simulator.NotUnderutilized,
						},
						{
							Node: &apiv1.Node{
								ObjectMeta: metav1.ObjectMeta{
									Name: "node2",
								},
							},
							NodeGroup: &fakeNodeGroup{id: "ng1"},
							Reason:    simulator.NotUnderutilized,
						},
					},
				},
			},
			nodesCount: []int{2, 2},
			nodesCountLabels: [][]string{
				{"8", "ng1", "", "", ""},
				{"8", "ng1", "", "", ""},
			},
			nodesCountByNode: []int{1, 1},
			nodesCountByNodeLabels: [][]string{
				{"8", "ng1", "node1", "", "", ""},
				{"8", "ng1", "node2", "", "", ""},
			},
		},
		{
			name: "simple unremovable node with blocking pod",
			statuses: []*ScaleDownStatus{
				{
					UnremovableNodes: []*UnremovableNode{
						{
							Node: &apiv1.Node{
								ObjectMeta: metav1.ObjectMeta{
									Name: "node1",
								},
							},
							NodeGroup: &fakeNodeGroup{id: "ng1"},
							Reason:    simulator.NotUnderutilized,
							BlockingPod: &drain.BlockingPod{
								Pod: &apiv1.Pod{
									ObjectMeta: metav1.ObjectMeta{
										Name:      "pod1",
										Namespace: "ns1",
									},
								},
								Reason: drain.NotEnoughPdb,
							},
						},
					},
				},
			},
			nodesCount:             []int{1},
			nodesCountLabels:       [][]string{{"8", "ng1", "7", "pod1", "ns1"}},
			nodesCountByNode:       []int{1},
			nodesCountByNodeLabels: [][]string{{"8", "ng1", "node1", "7", "pod1", "ns1"}},
		},
	}

	// Only one because we can't register metrics multiple times
	p := NewMetricsScaleDownStatusProcessor()
	for _, tt := range tests {
		p.reset = time.Now()

		for i, status := range tt.statuses {
			p.Process(&context.AutoscalingContext{}, status)

			assert.Equal(t, tt.nodesCount[i], int(testutil.ToFloat64(unremovableNodesCountByNodeGroup.GaugeVec.WithLabelValues(tt.nodesCountLabels[i]...))))
			assert.Equal(t, tt.nodesCountByNode[i], int(testutil.ToFloat64(unremovableNodesCountByNode.GaugeVec.WithLabelValues(tt.nodesCountByNodeLabels[i]...))))
		}
	}
}

type fakeNodeGroup struct {
	id string
}

func (f *fakeNodeGroup) Id() string {
	return f.id
}

func (f *fakeNodeGroup) MaxSize() int {
	return 0
}

func (f *fakeNodeGroup) MinSize() int {
	return 0
}

func (f *fakeNodeGroup) TargetSize() (int, error) {
	return 0, nil
}

func (f *fakeNodeGroup) IncreaseSize(delta int) error {
	return nil
}

func (f *fakeNodeGroup) DeleteNodes([]*apiv1.Node) error {
	return nil
}

func (f *fakeNodeGroup) DecreaseTargetSize(delta int) error {
	return nil
}

func (f *fakeNodeGroup) Debug() string {
	return ""
}

func (f *fakeNodeGroup) Nodes() ([]cloudprovider.Instance, error) {
	return nil, nil
}

func (f *fakeNodeGroup) TemplateNodeInfo() (*schedulerframework.NodeInfo, error) {
	return nil, nil
}

func (f *fakeNodeGroup) Exist() bool {
	return false
}

func (f *fakeNodeGroup) Create() (cloudprovider.NodeGroup, error) {
	return nil, nil
}

func (f *fakeNodeGroup) Delete() error {
	return nil
}

func (f *fakeNodeGroup) Autoprovisioned() bool {
	return false
}

func (f *fakeNodeGroup) GetOptions(defaults config.NodeGroupAutoscalingOptions) (*config.NodeGroupAutoscalingOptions, error) {
	return nil, nil
}
