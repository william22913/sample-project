package health

import "github.com/go-redis/redis/v7"

type redisConn struct {
	redis *redis.Conn
}

func (d *redisConn) Ping() string {
	if err := d.redis.Ping().Err(); err != nil {
		return "DOWN"
	}

	return "UP"
}

func NewRedisConnChecker(
	redis *redis.Conn,
) *redisConn {
	return &redisConn{
		redis: redis,
	}
}
