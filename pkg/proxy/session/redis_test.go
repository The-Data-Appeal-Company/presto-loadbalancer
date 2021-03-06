package session

import (
	"context"
	"errors"
	"github.com/The-Data-Appeal-Company/trino-loadbalancer/pkg/api/trino"
	"github.com/The-Data-Appeal-Company/trino-loadbalancer/pkg/common/tests"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestRedisLinkerLinkCluster(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	container, client, err := tests.CreateRedisServer(ctx)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, container.Terminate(ctx))
	}()

	storage := RedisLinkerStorage{
		redis:  client,
		prefix: "test",
		ttl:    10 * time.Minute,
	}

	queryInfo := trino.QueryInfo{
		User:          "user",
		QueryID:       "query",
		TransactionID: "tx",
	}

	const coordinator = "benny-lo-spenny"
	err = storage.Link(ctx, queryInfo, coordinator)
	require.NoError(t, err)

	linkedCoordinator, err := storage.Get(ctx, queryInfo)

	require.NoError(t, err)
	require.Equal(t, coordinator, linkedCoordinator)

	err = storage.Unlink(ctx, queryInfo)
	require.NoError(t, err)

	_, err = storage.Get(ctx, queryInfo)

	require.Error(t, err)
	require.True(t, errors.Is(err, ErrLinkNotFound))

}

func TestRedisLinkerLinkNotFoundErr(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	container, client, err := tests.CreateRedisServer(ctx)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, container.Terminate(ctx))
	}()

	storage := RedisLinkerStorage{
		redis:  client,
		prefix: "test",
		ttl:    10 * time.Minute,
	}

	_, err = storage.Get(ctx, trino.QueryInfo{
		User:          "user",
		QueryID:       "query",
		TransactionID: "tx",
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, ErrLinkNotFound))
}
