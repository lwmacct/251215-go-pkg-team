// Package actor 提供轻量级 Actor 模型实现
//
// Actor 模式是一种并发计算模型，每个 Actor 是独立的计算单元：
//   - 拥有私有状态（无需锁保护）
//   - 通过消息邮箱（mailbox）接收消息
//   - 消息处理串行化（一次处理一条）
//   - 可以创建子 Actor、发送消息、修改自身状态
//
// 设计原则:
//   - 最小化依赖，仅使用标准库
//   - 支持本地 Actor，预留分布式扩展接口
//   - 提供完整的监督策略支持
package actor

import (
	"context"
	"fmt"
	"time"
)

// Message Actor 消息接口
// 所有 Actor 间传递的消息都必须实现此接口
type Message interface {
	// Kind 返回消息类型标识，用于路由和监控
	Kind() string
}

// PID (Process ID) Actor 进程标识符
// 类似 Erlang 的 PID，是 Actor 的唯一寻址方式
type PID struct {
	// ID Actor 唯一标识（本地）
	ID string
	// Address 网络地址，本地 Actor 为空
	// 格式: "host:port" 用于未来分布式扩展
	Address string
	// system 所属的 Actor 系统（内部使用）
	system *System
}

// String 返回 PID 的字符串表示
func (p *PID) String() string {
	if p.Address != "" {
		return fmt.Sprintf("%s@%s", p.ID, p.Address)
	}
	return p.ID
}

// Tell 发送消息（fire-and-forget）
func (p *PID) Tell(msg Message) {
	if p.system != nil {
		p.system.Send(p, msg)
	}
}

// TrySend 尝试发送消息（非阻塞）
// 如果邮箱已满，返回 false
func (p *PID) TrySend(msg Message) bool {
	if p.system == nil {
		return false
	}
	return p.system.TrySend(p, msg)
}

// Request 发送请求并等待响应（同步调用）
func (p *PID) Request(msg Message, timeout time.Duration) (Message, error) {
	if p.system == nil {
		return nil, fmt.Errorf("actor system not available")
	}
	return p.system.Request(p, msg, timeout)
}

// Actor Actor 接口
// 实现此接口即可成为 Actor
type Actor interface {
	// Receive 处理接收到的消息
	// ctx 提供 Actor 上下文，msg 为接收到的消息
	Receive(ctx *Context, msg Message)
}

// ActorFunc 函数式 Actor，便于快速创建简单 Actor
type ActorFunc func(ctx *Context, msg Message)

// Receive 实现 Actor 接口
func (f ActorFunc) Receive(ctx *Context, msg Message) {
	f(ctx, msg)
}

// BaseActor 基础 Actor 实现
// 提供默认的空实现，方便嵌入
type BaseActor struct{}

// Receive 默认实现，不处理任何消息
func (b *BaseActor) Receive(_ *Context, _ Message) {}

// Context Actor 执行上下文
// 提供 Actor 执行时所需的环境信息和操作方法
type Context struct {
	// Self 当前 Actor 的 PID
	Self *PID
	// Sender 消息发送者的 PID（如果有）
	Sender *PID
	// Parent 父 Actor 的 PID（如果有）
	Parent *PID
	// Children 子 Actor 列表
	Children []*PID

	// 内部引用
	system       *System
	ctx          context.Context
	message      Message
	responseChan chan Message    // 用于 Request/Response 模式
	requestCtx   context.Context // 请求的 context，用于检查是否已取消
}

// Reply 回复消息给发送者
// 如果是 Request/Response 模式，通过 channel 返回响应
// 如果有 Sender，通过消息发送响应
func (c *Context) Reply(msg Message) {
	// 优先使用 Request/Response 模式
	if c.responseChan != nil {
		// 检查请求是否已取消
		if c.requestCtx != nil {
			select {
			case <-c.requestCtx.Done():
				return
			default:
			}
		}

		select {
		case c.responseChan <- msg:
		case <-c.requestCtx.Done():
		default:
		}
		return
	}
	// 否则通过消息发送
	if c.Sender != nil {
		c.system.SendWithSender(c.Sender, msg, c.Self)
	}
}

// Forward 转发当前消息到另一个 Actor
func (c *Context) Forward(target *PID) {
	if c.message != nil {
		c.system.SendWithSender(target, c.message, c.Sender)
	}
}

// Spawn 创建子 Actor
func (c *Context) Spawn(actor Actor, name string) *PID {
	pid := c.system.spawn(actor, name, c.Self)
	c.Children = append(c.Children, pid)
	return pid
}

// SpawnWithProps 使用属性创建子 Actor
func (c *Context) SpawnWithProps(actor Actor, props *Props) *PID {
	pid := c.system.spawnWithProps(actor, props, c.Self)
	c.Children = append(c.Children, pid)
	return pid
}

// Stop 停止指定 Actor
func (c *Context) Stop(pid *PID) {
	c.system.Stop(pid)
}

// StopSelf 停止当前 Actor
func (c *Context) StopSelf() {
	c.system.Stop(c.Self)
}

// Context 获取 Go context
func (c *Context) Context() context.Context {
	return c.ctx
}

// Message 获取当前正在处理的消息
func (c *Context) Message() Message {
	return c.message
}

// System 获取 Actor 系统引用
func (c *Context) System() *System {
	return c.system
}

// Watch 监控另一个 Actor
// 当被监控的 Actor 终止时，会收到 Terminated 消息
func (c *Context) Watch(pid *PID) {
	c.system.Send(pid, &Watch{Watcher: c.Self})
}

// Unwatch 取消监控
func (c *Context) Unwatch(pid *PID) {
	c.system.Send(pid, &Unwatch{Watcher: c.Self})
}

// Props Actor 属性配置
type Props struct {
	// Name Actor 名称
	Name string
	// MailboxSize 邮箱大小
	MailboxSize int
	// Dispatcher 调度器类型
	Dispatcher DispatcherType
	// SupervisorStrategy 监督策略
	SupervisorStrategy SupervisorStrategy
}

// DefaultProps 默认属性
func DefaultProps(name string) *Props {
	return &Props{
		Name:               name,
		MailboxSize:        100,
		Dispatcher:         DispatcherDefault,
		SupervisorStrategy: nil,
	}
}

// WithMailboxSize 设置邮箱大小
func (p *Props) WithMailboxSize(size int) *Props {
	p.MailboxSize = size
	return p
}

// WithSupervisor 设置监督策略
func (p *Props) WithSupervisor(strategy SupervisorStrategy) *Props {
	p.SupervisorStrategy = strategy
	return p
}

// DispatcherType 调度器类型
type DispatcherType int

const (
	// DispatcherDefault 默认调度器（每个 Actor 一个 goroutine）
	DispatcherDefault DispatcherType = iota
	// DispatcherShared 共享调度器（多个 Actor 共享 goroutine 池）
	DispatcherShared
)

// ============== 系统消息 ==============

// Started Actor 启动完成消息
type Started struct{}

// Kind 实现 Message 接口
func (s *Started) Kind() string { return "system.started" }

// Stopping Actor 正在停止消息
type Stopping struct{}

// Kind 实现 Message 接口
func (s *Stopping) Kind() string { return "system.stopping" }

// Stopped Actor 已停止消息
type Stopped struct{}

// Kind 实现 Message 接口
func (s *Stopped) Kind() string { return "system.stopped" }

// Restarting Actor 正在重启消息
type Restarting struct{}

// Kind 实现 Message 接口
func (r *Restarting) Kind() string { return "system.restarting" }

// PoisonPill 毒丸消息，优雅停止 Actor
type PoisonPill struct{}

// Kind 实现 Message 接口
func (p *PoisonPill) Kind() string { return "system.poison_pill" }

// Watch 监控请求
type Watch struct {
	Watcher *PID
}

// Kind 实现 Message 接口
func (w *Watch) Kind() string { return "system.watch" }

// Unwatch 取消监控
type Unwatch struct {
	Watcher *PID
}

// Kind 实现 Message 接口
func (u *Unwatch) Kind() string { return "system.unwatch" }

// Terminated Actor 终止通知
type Terminated struct {
	Who *PID
}

// Kind 实现 Message 接口
func (t *Terminated) Kind() string { return "system.terminated" }

// ============== 请求/响应支持 ==============

// ResponseTimeout 响应超时错误
type ResponseTimeout struct {
	Target  *PID
	Timeout time.Duration
}

// Kind 实现 Message 接口
func (r *ResponseTimeout) Kind() string { return "system.response_timeout" }

// Error 实现 error 接口
func (r *ResponseTimeout) Error() string {
	return fmt.Sprintf("request to %s timed out after %v", r.Target, r.Timeout)
}

// ============== 通用消息类型 ==============

// SimpleMessage 简单消息，用于快速创建消息
type SimpleMessage struct {
	kind    string
	Payload any
}

// NewSimpleMessage 创建简单消息
func NewSimpleMessage(kind string, payload any) *SimpleMessage {
	return &SimpleMessage{kind: kind, Payload: payload}
}

// Kind 实现 Message 接口
func (m *SimpleMessage) Kind() string { return m.kind }
