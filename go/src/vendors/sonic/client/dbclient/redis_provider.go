// This file is reponsible for maintaining connections for redis DBs.
package dbclient

import "github.com/go-redis/redis"

const (
	redisServerAddr     = "localhost:6379"
	redisServerPassword = ""
	COUNTER_DB_ID       = 2
)

var dbToRedisClientMapping map[int]*redis.Client

func init() {
	dbToRedisClientMapping = make(map[int]*redis.Client)
}

/* Gets redis connection for a specific redis database. The connection is re-used.
   If multiple go routines call this at same time, the last client will be stored. */
func GetRedisConnectionForDatabase(databaseIdentifier int) *redis.Client {
	redisClient, ok := dbToRedisClientMapping[databaseIdentifier]

	if ok {
		return redisClient
	}

	var client = redis.NewClient(&redis.Options{
		Addr:     redisServerAddr,
		Password: redisServerPassword,
		DB:       databaseIdentifier,
	})

	dbToRedisClientMapping[databaseIdentifier] = client
	return client
}

type RedisProviderInterface interface {
	HmGet(database int, key string, fields []string) ([]interface{}, error)
	HGetAll(database int, key string) (map[string]string, error)
}

type RedisProvider struct {
}

func (redisProvider *RedisProvider) HmGet(database int, key string, fields []string) ([]interface{}, error) {
	return GetRedisConnectionForDatabase(database).HMGet(key, fields...).Result()
}

func (redisProvider *RedisProvider) HGetAll(database int, key string) (map[string]string, error) {
	return GetRedisConnectionForDatabase(database).HGetAll(key).Result()
}

