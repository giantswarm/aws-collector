package collector

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/giantswarm/k8sclient/v4/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	eventCreateFailedReason         = "CFCreateFailed"
	eventDeleteFailedReason         = "CFDeleteFailed"
	eventUpdateRollbackFailedReason = "CFUpdateRollbackFailed"
	eventRollbackFailedReason       = "CFRollbackFailed"
)

var (
	CFFailedCreationDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "cloudformation", "creation_failed"),
		fmt.Sprintf("Amount of Cloudformation stacks which are in state `%s`", cloudformation.StackStatusCreateFailed),
		[]string{
			labelInstallation,
			labelCluster,
		},
		nil,
	)
	CFFailedDeletionDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "cloudformation", "deletion_failed"),
		fmt.Sprintf("Amount of Cloudformation stacks which are in state `%s`", cloudformation.StackStatusDeleteFailed),
		[]string{
			labelInstallation,
			labelCluster,
		},
		nil,
	)
	CFFailedUpdateRollbackDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "cloudformation", "update_rollback_failed"),
		fmt.Sprintf("Amount of Cloudformation stacks which are in state `%s`", cloudformation.StackStatusUpdateRollbackFailed),
		[]string{
			labelInstallation,
			labelCluster,
		},
		nil,
	)
	CFFailedRollbackDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "cloudformation", "rollback_failed"),
		fmt.Sprintf("Amount of Cloudformation stacks which are in state `%s`", cloudformation.StackStatusRollbackFailed),
		[]string{
			labelInstallation,
			labelCluster,
		},
		nil,
	)
)

type EventConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	InstallationName string
}

type Event struct {
	k8sClient k8sclient.Interface
	logger    micrologger.Logger

	installationName string
}

func NewEvent(config EventConfig) (*Event, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.InstallationName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.InstallationName must not be empty", config)
	}

	r := &Event{
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		installationName: config.InstallationName,
	}

	return r, nil
}

func (r *Event) Collect(ch chan<- prometheus.Metric) error {
	ctx := context.Background()

	var list *corev1.EventList
	var err error
	{
		list, err = r.k8sClient.K8sClient().CoreV1().Events("").List(ctx, metav1.ListOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var createFailedCount map[string]int
	var deleteFailedCount map[string]int
	var updateRollbackFailedCount map[string]int
	var rollbackFailedCount map[string]int

	for _, event := range list.Items {
		event := event // dereferencing pointer value into new scope

		switch event.Name {
		case eventCreateFailedReason:
			createFailedCount[event.InvolvedObject.Name]++
		case eventDeleteFailedReason:
			deleteFailedCount[event.InvolvedObject.Name]++
		case eventUpdateRollbackFailedReason:
			updateRollbackFailedCount[event.InvolvedObject.Name]++
		case eventRollbackFailedReason:
			rollbackFailedCount[event.InvolvedObject.Name]++
		}
	}

	for id, count := range createFailedCount {
		ch <- prometheus.MustNewConstMetric(
			CFFailedCreationDesc,
			prometheus.GaugeValue,
			float64(count),
			id,
		)
	}
	for id, count := range deleteFailedCount {
		ch <- prometheus.MustNewConstMetric(
			CFFailedDeletionDesc,
			prometheus.GaugeValue,
			float64(count),
			r.installationName,
			id,
		)
	}
	for id, count := range updateRollbackFailedCount {
		ch <- prometheus.MustNewConstMetric(
			CFFailedUpdateRollbackDesc,
			prometheus.GaugeValue,
			float64(count),
			r.installationName,
			id,
		)
	}
	for id, count := range rollbackFailedCount {
		ch <- prometheus.MustNewConstMetric(
			CFFailedRollbackDesc,
			prometheus.GaugeValue,
			float64(count),
			r.installationName,
			id,
		)
	}
	return nil
}

func (r *Event) Describe(ch chan<- *prometheus.Desc) error {
	ch <- CFFailedCreationDesc
	ch <- CFFailedDeletionDesc
	ch <- CFFailedUpdateRollbackDesc
	ch <- CFFailedRollbackDesc

	return nil
}
