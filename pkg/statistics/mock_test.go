package statistics

import (
	"errors"
	"github.com/The-Data-Appeal-Company/presto-loadbalancer/pkg/models"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMockRetriever(t *testing.T) {

	mockErr := errors.New("generic err")
	mockStats := models.ClusterStatistics{}
	retriever := Mock(mockStats, mockErr)

	stats, err := retriever.GetStatistics(models.Coordinator{})

	require.Equal(t, mockErr, err)
	require.Equal(t, mockStats, stats)

}