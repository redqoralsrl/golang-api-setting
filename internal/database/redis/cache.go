package redis

import (
	"go-template/config"

	"github.com/redis/go-redis/v9"
)

func ConnectCache(config *config.Config) (cache *redis.Client) {
	client := redis.NewClient(&redis.Options{
		Addr: config.RedisHost + ":" + config.RedisPort,
	})
	cache = client
	return cache
}
