package session

import (
	"context"
	"encoding/json"
	"time"

	"github.com/LittleCurry/go_first_ai/internal/config"
	"github.com/LittleCurry/go_first_ai/internal/models"
	"github.com/redis/go-redis/v9"
)

// RedisSessionManager Redis会话管理器
type RedisSessionManager struct {
	client  *redis.Client
	maxSize int64         // 每个会话最多保留的消息数
	ttl     time.Duration // 会话过期时间
}

// NewRedisSessionManager 创建Redis会话管理器
func NewRedisSessionManager(cfg *config.Config) *RedisSessionManager {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Host + ":" + cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	return &RedisSessionManager{
		client:  client,
		maxSize: 20,             // 保留最近20条消息
		ttl:     24 * time.Hour, // 会话24小时过期
	}
}

// getSessionKey 获取会话在Redis中的key
func (sm *RedisSessionManager) getSessionKey(sessionID string) string {
	return "session:" + sessionID
}

// getSessionMetaKey 获取会话元数据key
func (sm *RedisSessionManager) getSessionMetaKey(sessionID string) string {
	return "session:meta:" + sessionID
}

// GetOrCreate 获取或创建会话
func (sm *RedisSessionManager) GetOrCreate(ctx context.Context, sessionID, userID string) (*Session, error) {
	metaKey := sm.getSessionMetaKey(sessionID)

	// 检查会话是否存在
	exists, err := sm.client.Exists(ctx, metaKey).Result()
	if err != nil {
		return nil, err
	}

	if exists > 0 {
		// 获取已有会话
		data, err := sm.client.Get(ctx, metaKey).Result()
		if err != nil {
			return nil, err
		}

		var session Session
		if err := json.Unmarshal([]byte(data), &session); err != nil {
			return nil, err
		}

		// 获取消息列表
		messages, err := sm.GetHistory(ctx, sessionID, 0)
		if err != nil {
			return nil, err
		}
		session.Messages = messages

		return &session, nil
	}

	// 创建新会话
	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		Messages:  []models.Message{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 保存元数据
	data, err := json.Marshal(session)
	if err != nil {
		return nil, err
	}

	if err := sm.client.Set(ctx, metaKey, data, sm.ttl).Err(); err != nil {
		return nil, err
	}

	return session, nil
}

// AddMessage 添加消息到会话
func (sm *RedisSessionManager) AddMessage(ctx context.Context, sessionID string, msg models.Message) error {
	sessionKey := sm.getSessionKey(sessionID)
	metaKey := sm.getSessionMetaKey(sessionID)

	// 序列化消息
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// 使用Pipeline批量操作
	pipe := sm.client.Pipeline()

	// 添加到列表右侧
	pipe.RPush(ctx, sessionKey, data)
	// 保持列表长度不超过maxSize
	pipe.LTrim(ctx, sessionKey, -sm.maxSize, -1)
	// 更新会话更新时间
	pipe.HSet(ctx, metaKey, "updated_at", time.Now().Unix())
	// 设置过期时间
	pipe.Expire(ctx, sessionKey, sm.ttl)
	pipe.Expire(ctx, metaKey, sm.ttl)

	_, err = pipe.Exec(ctx)
	return err
}

// GetHistory 获取会话历史
func (sm *RedisSessionManager) GetHistory(ctx context.Context, sessionID string, maxCount int) ([]models.Message, error) {
	sessionKey := sm.getSessionKey(sessionID)

	var start, stop int64
	if maxCount <= 0 {
		// 获取全部
		start = 0
		stop = -1
	} else {
		// 获取最近的maxCount条
		start = -int64(maxCount)
		stop = -1
	}

	// 从Redis获取消息列表
	results, err := sm.client.LRange(ctx, sessionKey, start, stop).Result()
	if err != nil {
		return nil, err
	}

	messages := make([]models.Message, 0, len(results))
	for _, item := range results {
		var msg models.Message
		if err := json.Unmarshal([]byte(item), &msg); err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// Clear 清空会话
func (sm *RedisSessionManager) Clear(ctx context.Context, sessionID string) error {
	sessionKey := sm.getSessionKey(sessionID)
	metaKey := sm.getSessionMetaKey(sessionID)

	pipe := sm.client.Pipeline()
	pipe.Del(ctx, sessionKey)
	pipe.Del(ctx, metaKey)
	_, err := pipe.Exec(ctx)
	return err
}

// GetSessionCount 获取会话中的消息数量
func (sm *RedisSessionManager) GetSessionCount(ctx context.Context, sessionID string) (int64, error) {
	sessionKey := sm.getSessionKey(sessionID)
	return sm.client.LLen(ctx, sessionKey).Result()
}

// Ping 检查Redis连接
func (sm *RedisSessionManager) Ping(ctx context.Context) error {
	return sm.client.Ping(ctx).Err()
}

// Close 关闭Redis连接
func (sm *RedisSessionManager) Close() error {
	return sm.client.Close()
}

// Session 会话结构（用于元数据存储）
type Session struct {
	ID        string           `json:"id"`
	UserID    string           `json:"user_id"`
	Messages  []models.Message `json:"-"` // 不存储到元数据，单独存储
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}
