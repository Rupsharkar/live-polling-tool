package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisService struct {
	Client *redis.Client
}

type Update struct {
	Option int     `json:"option"`
	Counts []int64 `json:"counts"`
}

func NewRedisService(c *redis.Client) *RedisService {
	return &RedisService{Client: c}
}

func (r *RedisService) countKey(pollID string) string {
	return "poll:" + pollID + ":counts"
}

func (r *RedisService) channel(pollID string) string {
	return "poll:" + pollID + ":updates"
}

func (r *RedisService) SetCounts(ctx context.Context, pollID string, counts []int64) error {
	key := r.countKey(pollID)

	values := make(map[string]interface{})

	for i, n := range counts {
		values[strconv.Itoa(i)] = n
	}

	if err := r.Client.Del(ctx, key).Err(); err != nil {
		return err
	}

	if len(values) == 0 {
		return nil
	}

	return r.Client.HSet(ctx, key, values).Err()
}

func (r *RedisService) Increment(ctx context.Context, pollID string, option int) ([]int64, error) {
	key := r.countKey(pollID)

	if err := r.Client.HIncrBy(ctx, key, strconv.Itoa(option), 1).Err(); err != nil {
		return nil, err
	}

	values, err := r.Client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	counts := make([]int64, 0)

	for i := 0; ; i++ {
		value, ok := values[strconv.Itoa(i)]
		if !ok {
			break
		}

		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, err
		}

		counts = append(counts, n)
	}

	return counts, nil
}

func (r *RedisService) Publish(ctx context.Context, pollID string, update Update) error {
	data, err := json.Marshal(update)
	if err != nil {
		return err
	}

	return r.Client.Publish(ctx, r.channel(pollID), data).Err()
}

func (r *RedisService) Subscribe(pollID string) *redis.PubSub {
	return r.Client.Subscribe(context.Background(), r.channel(pollID))
}

func (r *RedisService) Expire(ctx context.Context, pollID string) {
	_ = r.Client.Expire(ctx, r.countKey(pollID), 24*time.Hour).Err()
}

func (r *RedisService) String() string {
	return fmt.Sprintf("RedisService")
}
