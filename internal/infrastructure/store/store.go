package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/mikiasgoitom/Articulate/internal/domain/contract"
	"github.com/mikiasgoitom/Articulate/internal/domain/entity"
)

type BlogCacheStore struct {
	rdb       *redis.Client
	detailTTL time.Duration
	listTTL   time.Duration
}

func NewBlogCacheStore(rdb *redis.Client) *BlogCacheStore {
	return &BlogCacheStore{
		rdb:       rdb,
		detailTTL: 60 * time.Minute, // 60 minutes
		listTTL:   30 * time.Minute, // 30 minutes
	}
}

func blogDetailKey(slug string) string { return fmt.Sprintf("blog:slug:%s", slug) }

func (c *BlogCacheStore) GetBlogBySlug(ctx context.Context, slug string) (*entity.Blog, bool, error) {
	b, err := c.rdb.Get(ctx, blogDetailKey(slug)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}
	var blog entity.Blog
	if err := json.Unmarshal(b, &blog); err != nil {
		return nil, false, nil
	}
	return &blog, true, nil
}

func (c *BlogCacheStore) SetBlogBySlug(ctx context.Context, slug string, blog *entity.Blog) error {
	data, err := json.Marshal(blog)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, blogDetailKey(slug), data, c.detailTTL).Err()
}

func (c *BlogCacheStore) InvalidateBlogBySlug(ctx context.Context, slug string) error {
	return c.rdb.Del(ctx, blogDetailKey(slug)).Err()
}

func (c *BlogCacheStore) GetBlogsPage(ctx context.Context, key string) (*contract.CachedBlogsPage, bool, error) {
	b, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}
	var page contract.CachedBlogsPage
	if err := json.Unmarshal(b, &page); err != nil {
		return nil, false, nil
	}
	return &page, true, nil
}

func (c *BlogCacheStore) SetBlogsPage(ctx context.Context, key string, page *contract.CachedBlogsPage) error {
	data, err := json.Marshal(page)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, data, c.listTTL).Err()
}

func (c *BlogCacheStore) InvalidateBlogLists(ctx context.Context) error {
	iter := c.rdb.Scan(ctx, 0, "blogs:list:*", 1000).Iterator()
	pipe := c.rdb.Pipeline()
	n := 0
	for iter.Next(ctx) {
		pipe.Del(ctx, iter.Val())
		n++
		if n%200 == 0 {
			if _, err := pipe.Exec(ctx); err != nil {
				return err
			}
		}
	}
	if err := iter.Err(); err != nil {
		return err
	}
	_, _ = pipe.Exec(ctx)
	return nil
}

// --- Fraud Detection Caching ---
// Use Redis sets with TTL for recent views by IP and by User
func recentViewsByIPKey(ip string) string { return fmt.Sprintf("blog:recentviews:ip:%s", ip) }
func recentViewsByUserKey(userID string) string {
	return fmt.Sprintf("blog:recentviews:user:%s", userID)
}

// Add a blogID to the recent views set for an IP, with TTL (ttlSeconds)
func (c *BlogCacheStore) AddRecentViewByIP(ctx context.Context, ip, blogID string, ttlSeconds int64) error {
	key := recentViewsByIPKey(ip)
	if err := c.rdb.SAdd(ctx, key, blogID).Err(); err != nil {
		return err
	}
	return c.rdb.Expire(ctx, key, time.Duration(ttlSeconds)*time.Second).Err()
}

// Get count of unique blogs viewed by this IP in the window
func (c *BlogCacheStore) GetRecentViewCountByIP(ctx context.Context, ip string) (int64, error) {
	return c.rdb.SCard(ctx, recentViewsByIPKey(ip)).Result()
}

// Add an IP to the recent views set for a user, with TTL (ttlSeconds)
func (c *BlogCacheStore) AddRecentViewByUser(ctx context.Context, userID, ip string, ttlSeconds int64) error {
	key := recentViewsByUserKey(userID)
	if err := c.rdb.SAdd(ctx, key, ip).Err(); err != nil {
		return err
	}
	return c.rdb.Expire(ctx, key, time.Duration(ttlSeconds)*time.Second).Err()
}

// Get count of unique IPs used by this user in the window
func (c *BlogCacheStore) GetRecentIPCountByUser(ctx context.Context, userID string) (int64, error) {
	return c.rdb.SCard(ctx, recentViewsByUserKey(userID)).Result()
}
