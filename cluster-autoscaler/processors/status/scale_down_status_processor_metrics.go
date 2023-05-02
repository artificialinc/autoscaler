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
	"fmt"

	"k8s.io/autoscaler/cluster-autoscaler/context"
	"k8s.io/autoscaler/cluster-autoscaler/simulator"
	k8smetrics "k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
)

const (
	caNamespace = "cluster_autoscaler"
	nodeMetrics = true
)

var (
	unremovableNodesCount = k8smetrics.NewGaugeVec(
		&k8smetrics.GaugeOpts{
			Namespace: caNamespace,
			Name:      "unremovable_nodes_count_by_node_group",
			Help:      "Number of nodes currently considered unremovable by CA. Dimensioned by reason and node group.",
		},
		[]string{"reason", "node_group"},
	)

	unremovableNodesCountByNode = k8smetrics.NewGaugeVec(
		&k8smetrics.GaugeOpts{
			Namespace: caNamespace,
			Name:      "unremovable_nodes_count_by_node",
			Help:      "Number of nodes currently considered unremovable by CA. Dimensioned by reason and node.",
		},
		[]string{"reason", "node_group", "node"},
	)
)

// NewMetricsScaleDownStatusProcessor creates a default instance of ScaleUpStatusProcessor.
func NewMetricsScaleDownStatusProcessor() ScaleDownStatusProcessor {
	legacyregistry.MustRegister(unremovableNodesCount)
	legacyregistry.MustRegister(unremovableNodesCountByNode)
	return &MetricsScaleDownStatusProcessor{}
}

// MetricsScaleDownStatusProcessor is a ScaleDownStatusProcessor implementations useful for testing.
type MetricsScaleDownStatusProcessor struct{}

// Process processes the status of the cluster after a scale-down.
func (p *MetricsScaleDownStatusProcessor) Process(context *context.AutoscalingContext, status *ScaleDownStatus) {
	type nodeGroupReason struct {
		nodeGroup string
		reason    simulator.UnremovableReason
	}
	unremovableNodes := make(map[nodeGroupReason]int)
	for _, node := range status.UnremovableNodes {
		unremovableNodes[nodeGroupReason{node.NodeGroup.Id(), node.Reason}]++
	}
	for nodeGroupReason, count := range unremovableNodes {
		unremovableNodesCount.WithLabelValues(fmt.Sprintf("%v", nodeGroupReason.reason), nodeGroupReason.nodeGroup).Set(float64(count))
	}

	// This would not scale well with node count. But we aren't there yet.
	// Could filter by reason if needed
	if nodeMetrics {
		for _, node := range status.UnremovableNodes {
			unremovableNodesCountByNode.WithLabelValues(fmt.Sprintf("%v", node.Reason), node.NodeGroup.Id(), node.Node.Name).Set(1)
		}
	}
}

// CleanUp cleans up the processor's internal structures.
func (p *MetricsScaleDownStatusProcessor) CleanUp() {
}
