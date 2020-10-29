package discovery

import (
	"context"
	"fmt"
	"github.com/The-Data-Appeal-Company/presto-loadbalancer/pkg/models"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"net/url"
)

type K8sClusterProvider struct {
	k8sClient     k8s.Interface
	SelectorTags  map[string]string
	clusterDomain string
}

func NewK8sClusterProvider(k8sClient k8s.Interface, selectorTags map[string]string, clusterDomain string) *K8sClusterProvider {
	return &K8sClusterProvider{k8sClient: k8sClient, SelectorTags: selectorTags, clusterDomain: clusterDomain}
}

func (k *K8sClusterProvider) Discover() ([]models.Coordinator, error) {

	ctx := context.TODO()
	coordinators := make([]models.Coordinator, 0)
	namespaces, err := k.k8sClient.CoreV1().Namespaces().List(ctx , v1.ListOptions{})

	if err != nil {
		return nil, err
	}

	for _, ns := range namespaces.Items {

		labelSelector := v1.LabelSelector{MatchLabels: k.SelectorTags}

		services, err := k.k8sClient.CoreV1().Services(ns.Namespace).List(ctx, v1.ListOptions{
			LabelSelector: labelSelector.String(),
		})

		if err != nil {
			return nil, err
		}

		for _, svc := range services.Items {
			svcUrl, err := url.Parse(fmt.Sprintf("%s.%s.%s", svc.Name, svc.Namespace, k.clusterDomain))
			if err != nil {
				return nil, err
			}

			dist, err := models.ParsePrestoDist(k.SelectorTags["presto.distribution"])

			if err != nil {
				return nil, err
			}

			coordinators = append(coordinators, models.Coordinator{
				Name:         svc.Name,
				URL:          svcUrl,
				Tags:         k.SelectorTags,
				Enabled:      true,
				Distribution: dist,
			})
		}
	}

	return coordinators, nil

}
