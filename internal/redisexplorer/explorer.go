package redisexplorer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"wincmp/internal/i18n"
)

// RedisKeyItem 表示一個 Redis Key 的基礎資訊
type RedisKeyItem struct {
	Key  string `json:"key"`
	Type string `json:"type"`
	TTL  int64  `json:"ttl"` // 秒數，-1 表示永久，-2 表示不存在
}

// RedisKeyDetail 表示一個 Redis Key 的詳細資料
type RedisKeyDetail struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	TTL   int64  `json:"ttl"`
	Value string `json:"value"`
	Size  int64  `json:"size"`
}

// RedisDBInfo 表示一個 Redis 資料庫資訊
type RedisDBInfo struct {
	DB   int   `json:"db"`
	Keys int64 `json:"keys"`
}

// getClient 建立短生命週期的 Redis Client
func getClient(addr string, db int) *redis.Client {
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	return redis.NewClient(&redis.Options{
		Addr:        addr,
		DB:          db,
		DialTimeout: 2 * time.Second,
		ReadTimeout: 3 * time.Second,
	})
}

// Ping 測試 Redis 連線是否可用
func Ping(addr string) (bool, error) {
	client := getClient(addr, 0)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		return false, fmt.Errorf("%s: %w", i18n.T("無法連線至 Redis 服務"), err)
	}
	return true, nil
}

// GetDBList 獲取 DB 0 ~ DB 15 的 Key 數量資訊
func GetDBList(addr string) ([]RedisDBInfo, error) {
	var list []RedisDBInfo
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	for db := 0; db < 16; db++ {
		client := getClient(addr, db)
		size, err := client.DBSize(ctx).Result()
		client.Close()
		if err != nil {
			// 若連線中斷則返回錯誤
			return nil, fmt.Errorf("%s DB %d: %w", i18n.T("獲取 Redis DB 資訊失敗"), db, err)
		}
		list = append(list, RedisDBInfo{
			DB:   db,
			Keys: size,
		})
	}
	return list, nil
}

// ScanKeys 分頁安全掃描 Redis Keys (避免阻塞)
func ScanKeys(addr string, db int, cursor uint64, match string, count int64) ([]RedisKeyItem, uint64, error) {
	if count <= 0 {
		count = 50
	}
	if count > 200 {
		count = 200
	}
	if match == "" {
		match = "*"
	}

	client := getClient(addr, db)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	keys, nextCursor, err := client.Scan(ctx, cursor, match, count).Result()
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", i18n.T("掃描 Redis 鍵值失敗"), err)
	}

	var items []RedisKeyItem
	if len(keys) == 0 {
		return items, nextCursor, nil
	}

	// 透過 Pipeline 批次查詢 Type 與 TTL，提高效能
	pipe := client.Pipeline()
	typeCmds := make([]*redis.StatusCmd, len(keys))
	ttlCmds := make([]*redis.DurationCmd, len(keys))

	for i, k := range keys {
		typeCmds[i] = pipe.Type(ctx, k)
		ttlCmds[i] = pipe.TTL(ctx, k)
	}

	_, _ = pipe.Exec(ctx)

	for i, k := range keys {
		keyType := "unknown"
		if typeCmds[i].Err() == nil {
			keyType = typeCmds[i].Val()
		}

		var ttlSec int64 = -1
		if ttlCmds[i].Err() == nil {
			dur := ttlCmds[i].Val()
			if dur == -1*time.Second {
				ttlSec = -1
			} else if dur == -2*time.Second {
				ttlSec = -2
			} else {
				ttlSec = int64(dur.Seconds())
			}
		}

		items = append(items, RedisKeyItem{
			Key:  k,
			Type: keyType,
			TTL:  ttlSec,
		})
	}

	return items, nextCursor, nil
}

// GetKeyDetail 獲取特定 Key 的詳細內容並自動格式化
func GetKeyDetail(addr string, db int, key string) (*RedisKeyDetail, error) {
	client := getClient(addr, db)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	keyType, err := client.Type(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T("查詢鍵值型態失敗"), err)
	}

	dur, _ := client.TTL(ctx, key).Result()
	var ttlSec int64 = -1
	if dur == -1*time.Second {
		ttlSec = -1
	} else if dur == -2*time.Second {
		ttlSec = -2
	} else {
		ttlSec = int64(dur.Seconds())
	}

	detail := &RedisKeyDetail{
		Key:  key,
		Type: keyType,
		TTL:  ttlSec,
	}

	switch keyType {
	case "string":
		val, err := client.Get(ctx, key).Result()
		if err != nil && err != redis.Nil {
			return nil, err
		}
		detail.Value = tryFormatJSON(val)
		detail.Size = int64(len(val))

	case "hash":
		vals, err := client.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		bytes, _ := json.MarshalIndent(vals, "", "  ")
		detail.Value = string(bytes)
		detail.Size = int64(len(vals))

	case "list":
		vals, err := client.LRange(ctx, key, 0, 99).Result()
		if err != nil {
			return nil, err
		}
		bytes, _ := json.MarshalIndent(vals, "", "  ")
		detail.Value = string(bytes)
		detail.Size = int64(len(vals))

	case "set":
		vals, err := client.SMembers(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		bytes, _ := json.MarshalIndent(vals, "", "  ")
		detail.Value = string(bytes)
		detail.Size = int64(len(vals))

	case "zset":
		vals, err := client.ZRangeWithScores(ctx, key, 0, 99).Result()
		if err != nil {
			return nil, err
		}
		type zsetRow struct {
			Member string  `json:"member"`
			Score  float64 `json:"score"`
		}
		rows := make([]zsetRow, 0, len(vals))
		for _, z := range vals {
			member := fmt.Sprintf("%v", z.Member)
			rows = append(rows, zsetRow{Member: member, Score: z.Score})
		}
		bytes, _ := json.Marshal(rows)
		detail.Value = string(bytes)
		detail.Size = int64(len(vals))

	default:
		detail.Value = fmt.Sprintf("(未支援直接預覽的型態: %s)", keyType)
	}

	return detail, nil
}

// tryFormatJSON 嘗試將字串排版為美觀 JSON，若非 JSON 則原樣返回
func tryFormatJSON(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
		var obj interface{}
		if err := json.Unmarshal([]byte(trimmed), &obj); err == nil {
			if pretty, err := json.MarshalIndent(obj, "", "  "); err == nil {
				return string(pretty)
			}
		}
	}
	return raw
}
