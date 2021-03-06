package discovery

import (
	"context"
	"fmt"
	"github.com/The-Data-Appeal-Company/trino-loadbalancer/pkg/common/models"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/emr"
	"github.com/stretchr/testify/require"
	"net/url"
	"testing"
)

type EMRMock struct{}

func (E *EMRMock) ListClustersPagesWithContext(ctx aws.Context, input *emr.ListClustersInput, fn func(*emr.ListClustersOutput, bool) bool, opts ...request.Option) error {
	output := &emr.ListClustersOutput{
		Clusters: []*emr.ClusterSummary{
			{
				Id: aws.String("teminatedCluster"),
				Status: &emr.ClusterStatus{
					State: aws.String(emr.ClusterStateTerminating),
				},
			},
			{
				Id: aws.String("waiting_no_trino"),
				Status: &emr.ClusterStatus{
					State: aws.String(emr.ClusterStateWaiting),
				},
			},
			{
				Id: aws.String("running_trino"),
				Status: &emr.ClusterStatus{
					State: aws.String(emr.ClusterStateRunning),
				},
			},
			{
				Id: aws.String("running_trinodb"),
				Status: &emr.ClusterStatus{
					State: aws.String(emr.ClusterStateRunning),
				},
			},
			{
				Id: aws.String("running_trinodb_no_tags"),
				Status: &emr.ClusterStatus{
					State: aws.String(emr.ClusterStateRunning),
				},
			},
		},
	}
	fn(output, true)
	return nil
}

func (E *EMRMock) DescribeCluster(input *emr.DescribeClusterInput) (*emr.DescribeClusterOutput, error) {
	if *input.ClusterId == "waiting_no_trino" {
		return &emr.DescribeClusterOutput{
			Cluster: &emr.Cluster{
				Id:                     input.ClusterId,
				InstanceCollectionType: aws.String(emr.InstanceCollectionTypeInstanceGroup),
				Applications: []*emr.Application{
					{
						Name: aws.String("no_trino"),
					},
				},
			},
		}, nil
	} else if *input.ClusterId == "running_trino" {
		return &emr.DescribeClusterOutput{
			Cluster: &emr.Cluster{
				Id: input.ClusterId,
				Applications: []*emr.Application{
					{
						Name: aws.String("trino"),
					},
				},
				InstanceCollectionType: aws.String(emr.InstanceCollectionTypeInstanceGroup),
				Tags: []*emr.Tag{
					{
						Key:   aws.String("component"),
						Value: aws.String("coordinator"),
					},
				},
			},
		}, nil
	} else if *input.ClusterId == "running_trinodb" {
		return &emr.DescribeClusterOutput{
			Cluster: &emr.Cluster{
				Id: input.ClusterId,
				Applications: []*emr.Application{
					{
						Name: aws.String("trino"),
					},
				},
				InstanceCollectionType: aws.String(emr.InstanceCollectionTypeInstanceGroup),
				Tags: []*emr.Tag{
					{
						Key:   aws.String("component"),
						Value: aws.String("coordinator"),
					},
				},
			},
		}, nil
	} else if *input.ClusterId == "running_trinodb_no_tags" {
		return &emr.DescribeClusterOutput{
			Cluster: &emr.Cluster{
				Id: input.ClusterId,
				Applications: []*emr.Application{
					{
						Name: aws.String("trino"),
					},
				},
				InstanceCollectionType: aws.String(emr.InstanceCollectionTypeInstanceGroup),
			},
		}, nil
	}
	return nil, fmt.Errorf("invalid cluster id")
}

func (E *EMRMock) ListInstances(input *emr.ListInstancesInput) (*emr.ListInstancesOutput, error) {
	if *input.ClusterId == "running_trino" {
		return &emr.ListInstancesOutput{
			Instances: []*emr.Instance{
				{
					Id:               aws.String("id_1"),
					PrivateIpAddress: aws.String("10.11.12.13"),
				},
			},
		}, nil
	} else if *input.ClusterId == "running_trinodb" {
		return &emr.ListInstancesOutput{
			Instances: []*emr.Instance{
				{
					Id:               aws.String("id_2"),
					PrivateIpAddress: aws.String("10.12.13.14"),
				},
			},
		}, nil
	} else if *input.ClusterId == "running_trinodb_no_tags" {
		return &emr.ListInstancesOutput{
			Instances: []*emr.Instance{
				{
					Id:               aws.String("id_3"),
					PrivateIpAddress: aws.String("10.13.14.15"),
				},
			},
		}, nil
	} else if *input.ClusterId == "waiting_no_trino" {
		return &emr.ListInstancesOutput{
			Instances: []*emr.Instance{
				{
					Id:               aws.String("id_4"),
					PrivateIpAddress: aws.String("10.13.14.15"),
				},
			},
		}, nil
	}
	return nil, fmt.Errorf("invalid instance %s", *input.ClusterId)
}

func TestClusterProvider_Discover(t *testing.T) {
	type fields struct {
		emrClient  ElasticMapReduce
		SelectTags map[string]string
	}
	tests := []struct {
		name    string
		fields  fields
		want    []models.Coordinator
		wantErr bool
	}{
		{
			name: "should discover trino from both group and fleet and filter cluster by tags",
			fields: fields{
				emrClient: &EMRMock{},
				SelectTags: map[string]string{
					"component": "coordinator",
				},
			},
			want: []models.Coordinator{
				{
					Name: "running_trino",
					URL:  getUrlOrPanic("http://10.11.12.13:8889"),
					Tags: map[string]string{
						"component": "coordinator",
					},
					Enabled: true,
				},
				{
					Name: "running_trinodb",
					URL:  getUrlOrPanic("http://10.12.13.14:8889"),
					Tags: map[string]string{
						"component": "coordinator",
					},
					Enabled: true,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterProvider{
				emrClient:  tt.fields.emrClient,
				SelectTags: tt.fields.SelectTags,
			}
			got, err := c.Discover(context.TODO())
			if (err != nil) != tt.wantErr {
				t.Errorf("Discover() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func getUrlOrPanic(s string) *url.URL {
	parsed, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return parsed
}
