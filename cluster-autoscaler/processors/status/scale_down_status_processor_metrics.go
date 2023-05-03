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
	"time"

	"k8s.io/autoscaler/cluster-autoscaler/context"
	"k8s.io/autoscaler/cluster-autoscaler/simulator"
	"k8s.io/autoscaler/cluster-autoscaler/utils/drain"
	k8smetrics "k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
	klog "k8s.io/klog/v2"
)

const (
	caNamespace = "cluster_autoscaler"
	nodeMetrics = true
	resetTime   = 5 * time.Minute
)

var (
	unremovableNodesCountByNodeGroup = k8smetrics.NewGaugeVec(
		&k8smetrics.GaugeOpts{
			Namespace: caNamespace,
			Name:      "detailed_unremovable_nodes_count_by_node_group",
			Help:      "Number of nodes currently considered unremovable by CA. Dimensioned by reason and node group.",
		},
		[]string{"reason", "node_group", "blocking_pod_reason", "blocking_pod_name", "blocking_pod_namespace"},
	)

	unremovableNodesCountByNode = k8smetrics.NewGaugeVec(
		&k8smetrics.GaugeOpts{
			Namespace: caNamespace,
			Name:      "detailed_unremovable_nodes_count_by_node",
			Help:      "Number of nodes currently considered unremovable by CA. Dimensioned by reason and node.",
		},
		[]string{"reason", "node_group", "node", "blocking_pod_reason", "blocking_pod_name", "blocking_pod_namespace"},
	)

	unremovableNodesCountResetTime = k8smetrics.NewGauge(
		&k8smetrics.GaugeOpts{
			Namespace: caNamespace,
			Name:      "detailed_unremovable_nodes_count_reset_time",
			Help:      "Time when the detailed_unremovable_nodes_count_by_node_group metric will reset next.",
		},
	)
)

// NewMetricsScaleDownStatusProcessor creates a default instance of ScaleUpStatusProcessor.
func NewMetricsScaleDownStatusProcessor() *MetricsScaleDownStatusProcessor {
	legacyregistry.MustRegister(unremovableNodesCountByNodeGroup)
	legacyregistry.MustRegister(unremovableNodesCountByNode)
	return &MetricsScaleDownStatusProcessor{
		reset: time.Now().Add(2 * time.Minute),
	}
}

// MetricsScaleDownStatusProcessor is a ScaleDownStatusProcessor implementations useful for testing.
type MetricsScaleDownStatusProcessor struct {
	reset time.Time
}

// Process processes the status of the cluster after a scale-down.
func (p *MetricsScaleDownStatusProcessor) Process(context *context.AutoscalingContext, status *ScaleDownStatus) {
	// Reset metrics every 2 minutes to avoid stale metrics. This is because of the way the statuses come in, we aren't guaranteed to get a report on every unremovable node each time
	// Because of this if we reset every time, we would get metrics with holes in them. Instead, reset every 2 minutes, and alert when something is over 5 minutes in a state
	// Technically we still have holes, because after a reset it can take a cycle or 2 to fill then again, but this is ok for our purposes
	if time.Now().After(p.reset) {
		klog.V(1).Infof("Resetting unremovable node metrics")
		p.reset = time.Now().Add(2 * time.Minute)
		unremovableNodesCountByNodeGroup.Reset()
		unremovableNodesCountByNode.Reset()
	} else {
		klog.V(1).Infof("Not resetting unremovable node metrics until %v", p.reset.Format(time.RFC3339))
	}
	unremovableNodesCountResetTime.Set(float64(p.reset.Unix()))

	type nodeGroupReason struct {
		nodeGroup string
		reason    simulator.UnremovableReason
		podReason *drain.BlockingPod
	}
	unremovableNodes := make(map[nodeGroupReason]int)
	for _, node := range status.UnremovableNodes {
		unremovableNodes[nodeGroupReason{node.NodeGroup.Id(), node.Reason, node.BlockingPod}]++
	}
	for nodeGroupReason, count := range unremovableNodes {
		if nodeGroupReason.podReason != nil {
			unremovableNodesCountByNodeGroup.WithLabelValues(fmt.Sprintf("%v", nodeGroupReason.reason), nodeGroupReason.nodeGroup, fmt.Sprintf("%v", nodeGroupReason.podReason.Reason), nodeGroupReason.podReason.Pod.Name, nodeGroupReason.podReason.Pod.Namespace).Set(float64(count))
		} else {
			unremovableNodesCountByNodeGroup.WithLabelValues(fmt.Sprintf("%v", nodeGroupReason.reason), nodeGroupReason.nodeGroup, "", "", "").Set(float64(count))
		}
	}

	// This would not scale well with node count. But we aren't there yet.
	// Could filter by reason if needed
	if nodeMetrics {
		for _, node := range status.UnremovableNodes {
			if node.BlockingPod != nil {
				unremovableNodesCountByNode.WithLabelValues(fmt.Sprintf("%v", node.Reason), node.NodeGroup.Id(), node.Node.Name, fmt.Sprintf("%v", node.BlockingPod.Reason), node.BlockingPod.Pod.Name, node.BlockingPod.Pod.Namespace).Set(1)
			} else {
				unremovableNodesCountByNode.WithLabelValues(fmt.Sprintf("%v", node.Reason), node.NodeGroup.Id(), node.Node.Name, "", "", "").Set(1)
			}
		}
	}

}

// CleanUp cleans up the processor's internal structures.
func (p *MetricsScaleDownStatusProcessor) CleanUp() {
}
