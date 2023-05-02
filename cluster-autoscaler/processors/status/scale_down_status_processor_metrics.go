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
)

var unremovableNodesCount = k8smetrics.NewGaugeVec(
	&k8smetrics.GaugeOpts{
		Namespace: caNamespace,
		Name:      "unremovable_nodes_count_by_node_group",
		Help:      "Number of nodes currently considered unremovable by CA. Dimensioned by reason and node group.",
	},
	[]string{"reason", "node_group"},
)

// NewMetricsScaleDownStatusProcessor creates a default instance of ScaleUpStatusProcessor.
func NewMetricsScaleDownStatusProcessor() ScaleDownStatusProcessor {
	legacyregistry.MustRegister(unremovableNodesCount)
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
}

// CleanUp cleans up the processor's internal structures.
func (p *MetricsScaleDownStatusProcessor) CleanUp() {
}
