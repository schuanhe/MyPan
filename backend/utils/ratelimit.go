package utils

import (
	"sync"
	"time"
)

// ipRecord 记录某个 IP 在窗口内的尝试次数
type ipRecord struct {
	count     int
	windowEnd time.Time
}

// RateLimiter 基于内存的 IP 滑动窗口频率限制器（线程安全）
type RateLimiter struct {
	mu       sync.Mutex
	records  map[string]*ipRecord
	limit    int           // 窗口内允许的最大次数
	window   time.Duration // 时间窗口长度
}

// NewRateLimiter 创建限流器，limit 为窗口内最大请求数，window 为窗口时长
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		records: make(map[string]*ipRecord),
		limit:   limit,
		window:  window,
	}
	// 后台定期清理过期记录，避免内存泄漏
	go rl.cleanupLoop()
	return rl
}

// Allow 返回 true 表示允许本次请求，false 表示超限
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rec, ok := rl.records[ip]
	if !ok || now.After(rec.windowEnd) {
		// 新窗口
		rl.records[ip] = &ipRecord{count: 1, windowEnd: now.Add(rl.window)}
		return true
	}
	rec.count++
	return rec.count <= rl.limit
}

// cleanupLoop 每分钟清理已过期的 IP 记录
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, rec := range rl.records {
			if now.After(rec.windowEnd) {
				delete(rl.records, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// 预定义限流器实例
var (
	// LoginLimiter 登录接口：60秒内同 IP 最多 10 次
	LoginLimiter = NewRateLimiter(10, 60*time.Second)

	// SharePasswordLimiter 密码分享提取码：60秒内同 IP 最多 5 次
	SharePasswordLimiter = NewRateLimiter(5, 60*time.Second)
)
