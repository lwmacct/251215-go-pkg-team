package agent

import (
	"context"

	agentpkg "github.com/lwmacct/251215-go-pkg-agent/pkg/agent"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
)

// ═══════════════════════════════════════════════════════════════════════════
// 核心执行消息
// ═══════════════════════════════════════════════════════════════════════════

// Run 执行对话请求
//
// 这是 Agent 的核心消息类型，对应 Agent.Run() 方法。
// 事件通过 EventChan 返回，channel 会在完成后关闭。
//
// 支持两种模式：
//   - 非流式（默认）: 一次性返回完整结果，适合简单问答
//   - 流式: 实时返回文本增量，适合长文本生成
//
// 使用示例:
//
//	// 非流式（默认）
//	eventCh := make(chan *agentpkg.AgentEvent, 16)
//	pid.Tell(&agent.Run{Text: "1+1=?", EventChan: eventCh})
//	for event := range eventCh {
//	    if event.Type == llm.EventTypeDone {
//	        fmt.Println(event.Result.Text)
//	    }
//	}
//
//	// 流式
//	eventCh := make(chan *agentpkg.AgentEvent, 16)
//	pid.Tell(&agent.Run{
//	    Text:      "写一篇文章",
//	    EventChan: eventCh,
//	    Options:   []agentpkg.RunOption{agentpkg.WithStreaming(true)},
//	})
//	for event := range eventCh {
//	    if event.Type == llm.EventTypeText {
//	        fmt.Print(event.Text)  // 逐字输出
//	    }
//	}
type Run struct {
	Text      string
	Context   context.Context
	EventChan chan *agentpkg.AgentEvent
	Options   []agentpkg.RunOption // 执行选项（如流式模式）
}

// Kind 实现 actor.Message 接口
func (m *Run) Kind() string { return "agent.run" }

// Chat 同步对话请求
//
// 这是 Agent.Chat() 的 Actor 消息封装。
// 结果通过 ReplyChan 返回。
//
// 使用示例:
//
//	replyCh := make(chan *ChatResult, 1)
//	pid.Tell(&agent.Chat{Text: "Hello", ReplyChan: replyCh})
//	result := <-replyCh
type Chat struct {
	Text      string
	Context   context.Context
	ReplyChan chan *ChatResult
}

// Kind 实现 actor.Message 接口
func (m *Chat) Kind() string { return "agent.chat" }

// ChatResult 对话结果
type ChatResult struct {
	Result *agentpkg.Result
	Error  error
}

// ═══════════════════════════════════════════════════════════════════════════
// 状态查询消息
// ═══════════════════════════════════════════════════════════════════════════

// GetStatus 获取状态请求
type GetStatus struct {
	ReplyChan chan *agentpkg.Status
}

// Kind 实现 actor.Message 接口
func (m *GetStatus) Kind() string { return "agent.get_status" }

// GetMessages 获取消息历史请求
type GetMessages struct {
	ReplyChan chan []llm.Message
}

// Kind 实现 actor.Message 接口
func (m *GetMessages) Kind() string { return "agent.get_messages" }

// ═══════════════════════════════════════════════════════════════════════════
// 生命周期消息
// ═══════════════════════════════════════════════════════════════════════════

// Stop 停止请求
type Stop struct {
	Reason string
}

// Kind 实现 actor.Message 接口
func (m *Stop) Kind() string { return "agent.stop" }

// ═══════════════════════════════════════════════════════════════════════════
// 事件订阅消息（可扩展）
// ═══════════════════════════════════════════════════════════════════════════

// Subscribe 订阅 Agent 事件
//
// 订阅后，Agent 的所有事件会转发到指定 Actor。
// 适用于监控、日志、UI 更新等场景。
//
// 使用示例:
//
//	// 订阅所有事件
//	agentPID.Tell(&agent.Subscribe{
//	    Subscriber: monitorPID,
//	    EventTypes: nil,  // nil 表示订阅所有类型
//	})
//
//	// 只订阅错误事件
//	agentPID.Tell(&agent.Subscribe{
//	    Subscriber: alertPID,
//	    EventTypes: []llm.EventType{llm.EventTypeError},
//	})
type Subscribe struct {
	Subscriber *Subscriber
}

// Kind 实现 actor.Message 接口
func (m *Subscribe) Kind() string { return "agent.subscribe" }

// Subscriber 订阅者信息
type Subscriber struct {
	// ID 订阅者标识（用于取消订阅）
	ID string
	// EventChan 事件通道
	EventChan chan *agentpkg.AgentEvent
	// EventTypes 感兴趣的事件类型（nil 表示所有）
	EventTypes []llm.EventType
}

// Unsubscribe 取消订阅
type Unsubscribe struct {
	SubscriberID string
}

// Kind 实现 actor.Message 接口
func (m *Unsubscribe) Kind() string { return "agent.unsubscribe" }

// ═══════════════════════════════════════════════════════════════════════════
// 多 Agent 协作消息（可扩展）
// ═══════════════════════════════════════════════════════════════════════════

// Delegate 委托任务给其他 Agent
//
// 用于多 Agent 协作场景，父 Agent 可以将任务委托给子 Agent。
//
// 使用示例:
//
//	parentPID.Tell(&agent.Delegate{
//	    TargetAgentID: "researcher",
//	    Task:          "Search for information about X",
//	    ResultChan:    resultCh,
//	})
type Delegate struct {
	// TargetAgentID 目标 Agent ID
	TargetAgentID string
	// Task 任务描述
	Task string
	// Context 执行上下文
	Context context.Context
	// ResultChan 结果通道
	ResultChan chan *DelegateResult
}

// Kind 实现 actor.Message 接口
func (m *Delegate) Kind() string { return "agent.delegate" }

// DelegateResult 委托任务结果
type DelegateResult struct {
	// AgentID 执行任务的 Agent ID
	AgentID string
	// Result 执行结果
	Result *agentpkg.Result
	// Error 错误信息
	Error error
}

// Handoff 任务移交
//
// 当前 Agent 完全移交控制权给另一个 Agent。
// 与 Delegate 不同，Handoff 后当前 Agent 不再参与该任务。
type Handoff struct {
	// TargetAgentID 目标 Agent ID
	TargetAgentID string
	// Context 任务上下文（包含历史消息）
	Context context.Context
	// Messages 要传递的消息历史
	Messages []llm.Message
}

// Kind 实现 actor.Message 接口
func (m *Handoff) Kind() string { return "agent.handoff" }

// AgentEvent Agent 事件通知（Actor 间传递）
//
// 当 Agent 产生事件时，通过此消息通知订阅者。
type AgentEvent struct {
	// AgentID 产生事件的 Agent ID
	AgentID string
	// Event 事件内容
	Event *agentpkg.AgentEvent
}

// Kind 实现 actor.Message 接口
func (m *AgentEvent) Kind() string { return "agent.event" }
