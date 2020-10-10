package integration

import (
	"context"
	"database/sql"
	"github.com/The-Data-Appeal-Company/presto-loadbalancer/pkg/discovery"
	"github.com/The-Data-Appeal-Company/presto-loadbalancer/pkg/healthcheck"
	"github.com/The-Data-Appeal-Company/presto-loadbalancer/pkg/lb"
	"github.com/The-Data-Appeal-Company/presto-loadbalancer/pkg/logging"
	"github.com/The-Data-Appeal-Company/presto-loadbalancer/pkg/models"
	"github.com/The-Data-Appeal-Company/presto-loadbalancer/pkg/routing"
	"github.com/The-Data-Appeal-Company/presto-loadbalancer/pkg/session"
	"github.com/The-Data-Appeal-Company/presto-loadbalancer/pkg/statistics"
	"github.com/The-Data-Appeal-Company/presto-loadbalancer/pkg/tests"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)
import _ "github.com/prestodb/presto-go-client/presto"

var proxyConfig = lb.ProxyConf{SyncDelay: 1 * time.Hour}

func TestIntegration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cluster0, c0, err := tests.CreatePrestoDatabase(ctx)
	require.NoError(t, err)
	defer cluster0.Terminate(ctx)

	cluster1, c1, err := tests.CreatePrestoDatabase(ctx)
	require.NoError(t, err)
	defer cluster1.Terminate(ctx)

	cluster2, c2, err := tests.CreatePrestoDatabase(ctx)
	require.NoError(t, err)
	defer cluster2.Terminate(ctx)

	stateStore := discovery.NewMemoryStorage()
	sessStore := session.NewMemoryStorage()
	hc := healthcheck.NewPrestoHealth()
	stats := statistics.NewPrestoClusterApi()

	router := routing.New(routing.RoundRobin())

	logger := logging.Noop()

	poolCfg := lb.PoolConfig{
		HealthCheckDelay: 5 * time.Second,
		StatisticsDelay:  5 * time.Second,
	}

	pool := lb.NewPool(poolCfg, sessStore, hc, stats, logger)
	poolSync := lb.NewPoolStateSync(stateStore, logging.Noop())

	err = stateStore.Add(ctx, models.Coordinator{
		Name:    "c0",
		URL:     c0,
		Tags:    nil,
		Enabled: true,
	})
	require.NoError(t, err)

	err = stateStore.Add(ctx, models.Coordinator{
		Name:    "c1",
		URL:     c1,
		Tags:    nil,
		Enabled: true,
	})
	require.NoError(t, err)

	err = stateStore.Add(ctx, models.Coordinator{
		Name:    "c2",
		URL:     c2,
		Tags:    nil,
		Enabled: true,
	})
	require.NoError(t, err)

	err = poolSync.Sync(pool)
	require.NoError(t, err)

	proxy := lb.NewPrestoProxy(proxyConfig, pool, poolSync, sessStore, router, logger)

	go func() {
		require.NoError(t, proxy.Init())
		require.NoError(t, proxy.Serve("0.0.0.0:4322"))
	}()

	time.Sleep(300 * time.Millisecond)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			err = testQuery("http://test@localhost:4322?catalog=memory")
			require.NoError(t, err)
		}(&wg)
	}
	wg.Wait()

	err = cluster0.Terminate(ctx)
	require.NoError(t, err)

	err = pool.UpdateStatus()
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		err = testQuery("http://test@localhost:4322?catalog=memory")
		require.NoError(t, err)
	}

}

func testQuery(address string) error {
	dsn := address
	db, err := sql.Open("presto", dsn)
	if err != nil {
		return err
	}

	row, err := db.Query("select 1")
	if err != nil {
		return err
	}

	row.Next()

	var r int
	if err := row.Scan(&r); err != nil {
		return err
	}

	return nil
}