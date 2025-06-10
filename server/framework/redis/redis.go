// redis.go
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"server/framework/config"
	"server/framework/logger"

	"github.com/redis/go-redis/v9"
)

var (
	redisClient *redis.Client
)

// InitRedis 初始化Redis连接
func InitRedis() error {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.GlobalConfig.Redis.Host, config.GlobalConfig.Redis.Port),
		Password: config.GlobalConfig.Redis.Password,
		DB:       config.GlobalConfig.Redis.DB,
		PoolSize: 100,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("连接Redis失败: %v", err)
	}

	logger.Infof("Redis连接成功: %s:%d", config.GlobalConfig.Redis.Host, config.GlobalConfig.Redis.Port)
	return nil
}

// GetClient 获取Redis客户端
func GetClient() *redis.Client {
	return redisClient
}

// Set 设置缓存
func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	var data []byte
	var err error

	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		data, err = json.Marshal(value)
		if err != nil {
			return fmt.Errorf("序列化值失败: %v", err)
		}
	}

	return redisClient.Set(ctx, key, data, expiration).Err()
}

// Get 获取缓存
func Get(ctx context.Context, key string) (string, error) {
	return redisClient.Get(ctx, key).Result()
}

// Del 删除缓存
func Del(ctx context.Context, keys ...string) error {
	return redisClient.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func Exists(ctx context.Context, keys ...string) (int64, error) {
	return redisClient.Exists(ctx, keys...).Result()
}

// Expire 设置过期时间
func Expire(ctx context.Context, key string, expiration time.Duration) error {
	return redisClient.Expire(ctx, key, expiration).Err()
}

// TTL 获取剩余过期时间
func TTL(ctx context.Context, key string) (time.Duration, error) {
	return redisClient.TTL(ctx, key).Result()
}

// HSet 设置哈希表字段
func HSet(ctx context.Context, key string, values ...interface{}) error {
	return redisClient.HSet(ctx, key, values...).Err()
}

// HGet 获取哈希表字段
func HGet(ctx context.Context, key, field string) (string, error) {
	return redisClient.HGet(ctx, key, field).Result()
}

// HGetAll 获取哈希表所有字段
func HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return redisClient.HGetAll(ctx, key).Result()
}

// HDel 删除哈希表字段
func HDel(ctx context.Context, key string, fields ...string) error {
	return redisClient.HDel(ctx, key, fields...).Err()
}

// HExists 检查哈希表字段是否存在
func HExists(ctx context.Context, key, field string) (bool, error) {
	return redisClient.HExists(ctx, key, field).Result()
}

// LPush 从左侧推入列表
func LPush(ctx context.Context, key string, values ...interface{}) error {
	return redisClient.LPush(ctx, key, values...).Err()
}

// RPush 从右侧推入列表
func RPush(ctx context.Context, key string, values ...interface{}) error {
	return redisClient.RPush(ctx, key, values...).Err()
}

// LRange 获取列表范围
func LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return redisClient.LRange(ctx, key, start, stop).Result()
}

// LTrim 修剪列表
func LTrim(ctx context.Context, key string, start, stop int64) error {
	return redisClient.LTrim(ctx, key, start, stop).Err()
}

// Keys 获取匹配的键
func Keys(ctx context.Context, pattern string) ([]string, error) {
	return redisClient.Keys(ctx, pattern).Result()
}

// Close 关闭 Redis 客户端
func Close() error {
	if redisClient != nil {
		return redisClient.Close()
	}
	return nil
}
