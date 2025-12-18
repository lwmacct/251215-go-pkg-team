package agent

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	agentpkg "github.com/lwmacct/251215-go-pkg-agent/pkg/agent"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-team/pkg/actor"
)

// AgentActor 将 Agent 包装为 Actor
//
// 这是适配器模式，不修改原有 Agent 代码。
// Agent 可以独立使用，Actor 是可选的并发安全包装。
//
// 新设计说明:
//   - 核心方法: handleRun（对应 Agent.Run）
//   - 便捷方法: handleChat（对应 Agent.Chat）
//   - 支持事件订阅（Subscribe/Unsubscribe）
//   - 支持多 Agent 协作（Delegate/Handoff）
type AgentActor struct {
	agent agentpkg.AgentInterface

	// 状态
	mu        sync.RWMutex
	isRunning bool
	lastError error

	// 事件订阅者
	subscribers map[string]*Subscriber

	// 统计（使用通用组件）
	stats *actor.StatsCollector

	// 生命周期
	ctx    context.Context
	cancel context.CancelFunc

	// 日志
	logger *slog.Logger
}

// Stats 是 AgentActor 统计信息的类型别名
type Stats = actor.ActorStats

// New 创建 Agent Actor 适配器
func New(ag agentpkg.AgentInterface) *AgentActor {
	ctx, cancel := context.WithCancel(context.Background())
	return &AgentActor{
		agent:       ag,
		isRunning:   true,
		subscribers: make(map[string]*Subscriber),
		stats:       actor.NewStatsCollector(),
		ctx:         ctx,
		cancel:      cancel,
		logger:      slog.Default(),
	}
}

// NewWithLogger 创建带日志的 Agent Actor
func NewWithLogger(ag agentpkg.AgentInterface, logger *slog.Logger) *AgentActor {
	aa := New(ag)
	aa.logger = logger
	return aa
}

// Receive 处理接收到的消息（Actor 核心方法）
func (a *AgentActor) Receive(ctx *actor.Context, msg actor.Message) {
	a.stats.RecordReceived()
	startTime := time.Now()

	defer func() {
		a.stats.RecordHandled(time.Since(startTime))
	}()

	// 处理系统消息
	switch m := msg.(type) {
	case *actor.Started:
		a.logger.Debug("AgentActor started", "agent_id", a.agent.ID())
		return

	case *actor.Stopping:
		a.logger.Debug("AgentActor stopping", "agent_id", a.agent.ID())
		a.handleStop()
		return

	case *actor.Stopped:
		a.logger.Debug("AgentActor stopped", "agent_id", a.agent.ID())
		return

	case *actor.Restarting:
		a.logger.Debug("AgentActor restarting", "agent_id", a.agent.ID())
		return

	// ─────────────────────────────────────────────────────────────────────
	// 核心执行消息
	// ─────────────────────────────────────────────────────────────────────

	case *Run:
		a.handleRun(ctx, m)

	case *Chat:
		a.handleChat(ctx, m)

	// ─────────────────────────────────────────────────────────────────────
	// 状态查询消息
	// ─────────────────────────────────────────────────────────────────────

	case *GetStatus:
		a.handleGetStatus(m)

	case *GetMessages:
		a.handleGetMessages(m)

	// ─────────────────────────────────────────────────────────────────────
	// 生命周期消息
	// ─────────────────────────────────────────────────────────────────────

	case *Stop:
		a.handleStop()
		ctx.StopSelf()

	// ─────────────────────────────────────────────────────────────────────
	// 事件订阅消息
	// ─────────────────────────────────────────────────────────────────────

	case *Subscribe:
		a.handleSubscribe(m)

	case *Unsubscribe:
		a.handleUnsubscribe(m)

	// ─────────────────────────────────────────────────────────────────────
	// 多 Agent 协作消息
	// ─────────────────────────────────────────────────────────────────────

	case *Delegate:
		a.handleDelegate(ctx, m)

	case *Handoff:
		a.handleHandoff(ctx, m)

	default:
		a.logger.Warn("AgentActor received unknown message", "type", msg.Kind())
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 核心执行处理
// ═══════════════════════════════════════════════════════════════════════════

// handleRun 处理 Run 消息
func (a *AgentActor) handleRun(_ *actor.Context, msg *Run) {
	execCtx := actor.MergeContexts(a.ctx, msg.Context)

	go func() {
		defer func() {
			if msg.EventChan != nil {
				close(msg.EventChan)
			}
		}()

		// 调用 Agent.Run()，传递执行选项
		eventCh := a.agent.Run(execCtx, msg.Text, msg.Options...)

		for event := range eventCh {
			// 发送到请求者
			if msg.EventChan != nil {
				select {
				case msg.EventChan <- event:
				case <-execCtx.Done():
					return
				}
			}

			// 广播给订阅者
			a.broadcastEvent(event)

			// 记录错误
			if event.Type == llm.EventTypeError && event.Error != nil {
				a.recordError(event.Error)
			}
		}
	}()
}

// handleChat 处理 Chat 消息（同步对话）
func (a *AgentActor) handleChat(_ *actor.Context, msg *Chat) {
	execCtx := actor.MergeContexts(a.ctx, msg.Context)

	go func() {
		result, err := a.agent.Chat(execCtx, msg.Text)

		response := &ChatResult{
			Result: result,
			Error:  err,
		}

		if err := actor.IgnoreContextError(err); err != nil {
			a.recordError(err)
		}

		if !actor.TrySend(msg.ReplyChan, response) {
			a.logger.Warn("chat reply channel full or closed")
		}
	}()
}

// ═══════════════════════════════════════════════════════════════════════════
// 状态查询处理
// ═══════════════════════════════════════════════════════════════════════════

// handleGetStatus 处理获取状态
func (a *AgentActor) handleGetStatus(msg *GetStatus) {
	status := a.agent.Status()
	actor.TrySend(msg.ReplyChan, status)
}

// handleGetMessages 处理获取消息历史
func (a *AgentActor) handleGetMessages(msg *GetMessages) {
	messages := a.agent.Messages()
	actor.TrySend(msg.ReplyChan, messages)
}

// ═══════════════════════════════════════════════════════════════════════════
// 生命周期处理
// ═══════════════════════════════════════════════════════════════════════════

// handleStop 处理停止
func (a *AgentActor) handleStop() {
	a.mu.Lock()
	a.isRunning = false
	a.mu.Unlock()

	a.cancel()

	// 关闭所有订阅者通道
	for _, sub := range a.subscribers {
		if sub.EventChan != nil {
			close(sub.EventChan)
		}
	}
	a.subscribers = make(map[string]*Subscriber)

	if err := a.agent.Close(); err != nil {
		a.logger.Warn("agent close error", "error", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 事件订阅处理
// ═══════════════════════════════════════════════════════════════════════════

// handleSubscribe 处理订阅
func (a *AgentActor) handleSubscribe(msg *Subscribe) {
	if msg.Subscriber == nil || msg.Subscriber.ID == "" {
		a.logger.Warn("invalid subscriber")
		return
	}

	a.mu.Lock()
	a.subscribers[msg.Subscriber.ID] = msg.Subscriber
	a.mu.Unlock()

	a.logger.Debug("subscriber added",
		"subscriber_id", msg.Subscriber.ID,
		"event_types", msg.Subscriber.EventTypes,
	)
}

// handleUnsubscribe 处理取消订阅
func (a *AgentActor) handleUnsubscribe(msg *Unsubscribe) {
	a.mu.Lock()
	delete(a.subscribers, msg.SubscriberID)
	a.mu.Unlock()

	a.logger.Debug("subscriber removed", "subscriber_id", msg.SubscriberID)
}

// broadcastEvent 广播事件给所有订阅者
func (a *AgentActor) broadcastEvent(event *agentpkg.AgentEvent) {
	a.mu.RLock()
	subscribers := make([]*Subscriber, 0, len(a.subscribers))
	for _, sub := range a.subscribers {
		subscribers = append(subscribers, sub)
	}
	a.mu.RUnlock()

	for _, sub := range subscribers {
		// 检查事件类型过滤
		if len(sub.EventTypes) > 0 && !containsEventType(sub.EventTypes, event.Type) {
			continue
		}

		// 非阻塞发送
		select {
		case sub.EventChan <- event:
		default:
			a.logger.Warn("subscriber event channel full",
				"subscriber_id", sub.ID,
				"event_type", event.Type,
			)
		}
	}
}

// containsEventType 检查事件类型是否在列表中
func containsEventType(types []llm.EventType, t llm.EventType) bool {
	for _, et := range types {
		if et == t {
			return true
		}
	}
	return false
}

// ═══════════════════════════════════════════════════════════════════════════
// 多 Agent 协作处理
// ═══════════════════════════════════════════════════════════════════════════

// handleDelegate 处理任务委托
func (a *AgentActor) handleDelegate(ctx *actor.Context, msg *Delegate) {
	// 查找目标 Agent Actor
	targetPID, found := ctx.System().GetActor(msg.TargetAgentID)
	if !found {
		if msg.ResultChan != nil {
			actor.TrySend(msg.ResultChan, &DelegateResult{
				AgentID: msg.TargetAgentID,
				Error:   fmt.Errorf("target agent not found: %s", msg.TargetAgentID),
			})
		}
		return
	}

	// 创建事件通道
	eventCh := make(chan *agentpkg.AgentEvent, 16)

	// 发送 Run 消息给目标 Agent
	targetPID.Tell(&Run{
		Text:      msg.Task,
		Context:   msg.Context,
		EventChan: eventCh,
	})

	// 异步等待结果
	go func() {
		var result *agentpkg.Result
		var lastError error

		for event := range eventCh {
			switch event.Type {
			case llm.EventTypeDone:
				result = event.Result
			case llm.EventTypeError:
				lastError = event.Error
			}
		}

		if msg.ResultChan != nil {
			actor.TrySend(msg.ResultChan, &DelegateResult{
				AgentID: msg.TargetAgentID,
				Result:  result,
				Error:   lastError,
			})
		}
	}()
}

// handleHandoff 处理任务移交
func (a *AgentActor) handleHandoff(ctx *actor.Context, msg *Handoff) {
	// 查找目标 Agent Actor
	targetPID, found := ctx.System().GetActor(msg.TargetAgentID)
	if !found {
		a.logger.Error("handoff target not found", "target", msg.TargetAgentID)
		return
	}

	// TODO: 实现消息历史传递
	// 这需要目标 Agent 支持接收历史消息的接口
	_ = targetPID
	_ = msg.Messages

	a.logger.Info("handoff initiated",
		"from", a.agent.ID(),
		"to", msg.TargetAgentID,
		"messages", len(msg.Messages),
	)
}

// ═══════════════════════════════════════════════════════════════════════════
// 辅助方法
// ═══════════════════════════════════════════════════════════════════════════

// recordError 记录错误
func (a *AgentActor) recordError(err error) {
	a.stats.RecordError(err)
	a.mu.Lock()
	a.lastError = err
	a.mu.Unlock()
}

// Agent 获取底层 Agent
func (a *AgentActor) Agent() agentpkg.AgentInterface {
	return a.agent
}

// Stats 获取统计信息
func (a *AgentActor) Stats() *Stats {
	return a.stats.Stats()
}

// SubscriberCount 获取订阅者数量
func (a *AgentActor) SubscriberCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.subscribers)
}
