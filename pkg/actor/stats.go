package actor

import (
	"sync"
	"sync/atomic"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// Actor 统计信息
// ═══════════════════════════════════════════════════════════════════════════

// ActorStats Actor 运行时统计信息
type ActorStats struct {
	// 消息计数
	MessagesReceived int64 // 接收的消息总数
	MessagesHandled  int64 // 成功处理的消息数
	Errors           int64 // 错误数

	// 延迟统计
	TotalLatency   time.Duration // 总延迟（用于计算平均值）
	AverageLatency time.Duration // 平均延迟
	MaxLatency     time.Duration // 最大延迟
	MinLatency     time.Duration // 最小延迟

	// 时间戳
	StartedAt     time.Time // 启动时间
	LastMessageAt time.Time // 最后消息时间
	LastErrorAt   time.Time // 最后错误时间

	// 错误信息
	LastError error // 最后一个错误
}

// Clone 克隆统计信息（线程安全的快照）
func (s *ActorStats) Clone() *ActorStats {
	return &ActorStats{
		MessagesReceived: s.MessagesReceived,
		MessagesHandled:  s.MessagesHandled,
		Errors:           s.Errors,
		TotalLatency:     s.TotalLatency,
		AverageLatency:   s.AverageLatency,
		MaxLatency:       s.MaxLatency,
		MinLatency:       s.MinLatency,
		StartedAt:        s.StartedAt,
		LastMessageAt:    s.LastMessageAt,
		LastErrorAt:      s.LastErrorAt,
		LastError:        s.LastError,
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// StatsCollector 统计收集器
// ═══════════════════════════════════════════════════════════════════════════

// StatsCollector 线程安全的统计收集器
type StatsCollector struct {
	mu    sync.RWMutex
	stats ActorStats
}

// NewStatsCollector 创建统计收集器
func NewStatsCollector() *StatsCollector {
	return &StatsCollector{
		stats: ActorStats{
			StartedAt:  time.Now(),
			MinLatency: time.Duration(1<<63 - 1), // 最大值，确保第一次会被更新
		},
	}
}

// RecordReceived 记录接收消息
func (c *StatsCollector) RecordReceived() {
	c.mu.Lock()
	c.stats.MessagesReceived++
	c.stats.LastMessageAt = time.Now()
	c.mu.Unlock()
}

// RecordHandled 记录成功处理消息
func (c *StatsCollector) RecordHandled(latency time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stats.MessagesHandled++
	c.stats.TotalLatency += latency

	// 更新平均延迟
	if c.stats.MessagesHandled > 0 {
		c.stats.AverageLatency = c.stats.TotalLatency / time.Duration(c.stats.MessagesHandled)
	}

	// 更新最大/最小延迟
	if latency > c.stats.MaxLatency {
		c.stats.MaxLatency = latency
	}
	if latency < c.stats.MinLatency {
		c.stats.MinLatency = latency
	}
}

// RecordError 记录错误
func (c *StatsCollector) RecordError(err error) {
	c.mu.Lock()
	c.stats.Errors++
	c.stats.LastError = err
	c.stats.LastErrorAt = time.Now()
	c.mu.Unlock()
}

// Stats 获取统计快照
func (c *StatsCollector) Stats() *ActorStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats.Clone()
}

// Reset 重置统计
func (c *StatsCollector) Reset() {
	c.mu.Lock()
	c.stats = ActorStats{
		StartedAt:  time.Now(),
		MinLatency: time.Duration(1<<63 - 1),
	}
	c.mu.Unlock()
}

// ═══════════════════════════════════════════════════════════════════════════
// StatsActor 统计中间件 Actor
// ═══════════════════════════════════════════════════════════════════════════

// StatsActor 包装任意 Actor 添加统计功能
// 使用装饰器模式，不修改原 Actor 代码
type StatsActor struct {
	inner     Actor
	collector *StatsCollector
}

// NewStatsActor 创建带统计的 Actor 包装器
func NewStatsActor(inner Actor) *StatsActor {
	return &StatsActor{
		inner:     inner,
		collector: NewStatsCollector(),
	}
}

// Receive 实现 Actor 接口
func (s *StatsActor) Receive(ctx *Context, msg Message) {
	s.collector.RecordReceived()
	startTime := time.Now()

	// 使用 defer 确保即使 panic 也能记录
	defer func() {
		latency := time.Since(startTime)
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				s.collector.RecordError(err)
			}
			panic(r) // 重新抛出，让监督策略处理
		}
		s.collector.RecordHandled(latency)
	}()

	s.inner.Receive(ctx, msg)
}

// Stats 获取统计信息
func (s *StatsActor) Stats() *ActorStats {
	return s.collector.Stats()
}

// Inner 获取内部 Actor
func (s *StatsActor) Inner() Actor {
	return s.inner
}

// Collector 获取统计收集器（用于外部记录错误等）
func (s *StatsActor) Collector() *StatsCollector {
	return s.collector
}

// ═══════════════════════════════════════════════════════════════════════════
// 原子统计收集器（更高性能版本）
// ═══════════════════════════════════════════════════════════════════════════

// AtomicStatsCollector 使用原子操作的高性能统计收集器
// 适用于高吞吐场景，但功能较 StatsCollector 简化
type AtomicStatsCollector struct {
	messagesReceived atomic.Int64
	messagesHandled  atomic.Int64
	errors           atomic.Int64
	totalLatencyNs   atomic.Int64

	// 非原子字段，需要锁保护
	mu            sync.RWMutex
	startedAt     time.Time
	lastMessageAt time.Time
	lastError     error
}

// NewAtomicStatsCollector 创建原子统计收集器
func NewAtomicStatsCollector() *AtomicStatsCollector {
	return &AtomicStatsCollector{
		startedAt: time.Now(),
	}
}

// RecordReceived 记录接收（原子操作）
func (c *AtomicStatsCollector) RecordReceived() {
	c.messagesReceived.Add(1)
	c.mu.Lock()
	c.lastMessageAt = time.Now()
	c.mu.Unlock()
}

// RecordHandled 记录处理完成（原子操作）
func (c *AtomicStatsCollector) RecordHandled(latency time.Duration) {
	c.messagesHandled.Add(1)
	c.totalLatencyNs.Add(int64(latency))
}

// RecordError 记录错误
func (c *AtomicStatsCollector) RecordError(err error) {
	c.errors.Add(1)
	c.mu.Lock()
	c.lastError = err
	c.mu.Unlock()
}

// Stats 获取统计快照
func (c *AtomicStatsCollector) Stats() *ActorStats {
	received := c.messagesReceived.Load()
	handled := c.messagesHandled.Load()
	errors := c.errors.Load()
	totalLatency := time.Duration(c.totalLatencyNs.Load())

	var avgLatency time.Duration
	if handled > 0 {
		avgLatency = totalLatency / time.Duration(handled)
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return &ActorStats{
		MessagesReceived: received,
		MessagesHandled:  handled,
		Errors:           errors,
		TotalLatency:     totalLatency,
		AverageLatency:   avgLatency,
		StartedAt:        c.startedAt,
		LastMessageAt:    c.lastMessageAt,
		LastError:        c.lastError,
	}
}
