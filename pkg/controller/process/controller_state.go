package process

import (
	"context"
	"fmt"
	"github.com/The-Data-Appeal-Company/trino-loadbalancer/pkg/common/models"
	"github.com/go-redis/redis/v8"
	"sync"
	"time"
)

const RedisStateKey = "controller-last-execution-date"

type State interface {
	Set(context.Context, models.Coordinator, time.Time) error
	Get(context.Context, models.Coordinator) (time.Time, error)
}

type InMemoryState struct {
	state map[string]time.Time
	l     *sync.Mutex
}

func NewInMemoryState() *InMemoryState {
	return &InMemoryState{
		state: make(map[string]time.Time),
		l:     &sync.Mutex{},
	}
}

func (i *InMemoryState) Set(ctx context.Context, coordinator models.Coordinator, t time.Time) error {
	i.l.Lock()
	defer i.l.Unlock()
	i.state[coordinator.Name] = t
	return nil
}

func (i *InMemoryState) Get(ctx context.Context, coordinator models.Coordinator) (time.Time, error) {
	i.l.Lock()
	defer i.l.Unlock()
	return i.state[coordinator.Name], nil
}

type RedisControllerState struct {
	client redis.UniversalClient
}

func NewRedisControllerState(client redis.UniversalClient) RedisControllerState {
	return RedisControllerState{
		client: client,
	}
}

func (r RedisControllerState) Set(ctx context.Context, coord models.Coordinator, duration time.Time) error {
	return r.client.Set(ctx, fmt.Sprintf("%s-%s", RedisStateKey, coord.Name), duration, -1).Err()
}

func (r RedisControllerState) Get(ctx context.Context, coord models.Coordinator) (time.Time, error) {
	state, err := r.client.Get(ctx, fmt.Sprintf("%s-%s", RedisStateKey, coord.Name)).Time()
	if err != nil {
		if err == redis.Nil {
			return time.Unix(0, 0), nil
		} else {
			return time.Time{}, err
		}
	}

	return state, err
}
