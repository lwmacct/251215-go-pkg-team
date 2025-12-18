package actor

import (
	"context"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// 通用请求-回复辅助函数
// ═══════════════════════════════════════════════════════════════════════════

// RequestMessage 带回复通道的消息包装器
// 用于实现类型安全的请求-回复模式
type RequestMessage[T any] struct {
	inner     Message
	ReplyChan chan T
}

// Kind 实现 Message 接口，代理到内部消息
func (r *RequestMessage[T]) Kind() string {
	return r.inner.Kind()
}

// Inner 获取内部消息
func (r *RequestMessage[T]) Inner() Message {
	return r.inner
}

// Reply 发送回复（非阻塞，如果通道满则丢弃）
func (r *RequestMessage[T]) Reply(value T) bool {
	if r.ReplyChan == nil {
		return false
	}
	select {
	case r.ReplyChan <- value:
		return true
	default:
		return false
	}
}

// NewRequestMessage 创建请求消息
func NewRequestMessage[T any](msg Message) *RequestMessage[T] {
	return &RequestMessage[T]{
		inner:     msg,
		ReplyChan: make(chan T, 1),
	}
}

// Ask 向 Actor 发送消息并等待响应
// 这是一个泛型辅助函数，简化了请求-回复模式的使用
//
// 用法示例:
//
//	type GetStatusMsg struct{}
//	func (m *GetStatusMsg) Kind() string { return "get_status" }
//
//	status, err := actor.Ask[*Status](pid, &GetStatusMsg{}, 5*time.Second)
func Ask[T any](pid *PID, msg Message, timeout time.Duration) (T, error) {
	var zero T

	replyCh := make(chan T, 1)

	// 创建包装消息
	reqMsg := &RequestMessage[T]{
		inner:     msg,
		ReplyChan: replyCh,
	}

	pid.Tell(reqMsg)

	select {
	case result := <-replyCh:
		return result, nil
	case <-time.After(timeout):
		return zero, &ResponseTimeout{Target: pid, Timeout: timeout}
	}
}

// AskWithContext 带 context 的请求-回复
// 支持通过 context 取消请求
func AskWithContext[T any](ctx context.Context, pid *PID, msg Message) (T, error) {
	var zero T

	replyCh := make(chan T, 1)

	reqMsg := &RequestMessage[T]{
		inner:     msg,
		ReplyChan: replyCh,
	}

	pid.Tell(reqMsg)

	select {
	case result := <-replyCh:
		return result, nil
	case <-ctx.Done():
		return zero, ctx.Err()
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Context 工具函数
// ═══════════════════════════════════════════════════════════════════════════

// MergeContexts 合并两个 context，任一取消则返回的 context 也取消
// 这在需要同时监听多个取消信号时很有用
//
// 用法示例:
//
//	// actorCtx: Actor 生命周期 context
//	// requestCtx: 单次请求 context
//	mergedCtx := actor.MergeContexts(actorCtx, requestCtx)
func MergeContexts(parent, child context.Context) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	if child == nil {
		return parent
	}

	ctx, cancel := context.WithCancel(parent)

	go func() {
		select {
		case <-parent.Done():
			cancel()
		case <-child.Done():
			cancel()
		case <-ctx.Done():
			// 已取消，退出 goroutine
		}
	}()

	return ctx
}

// MergeContextsWithCancel 合并 context 并返回取消函数
// 调用者负责在不再需要时调用 cancel 以释放资源
func MergeContextsWithCancel(parent, child context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	if child == nil {
		return context.WithCancel(parent)
	}

	ctx, cancel := context.WithCancel(parent)

	go func() {
		select {
		case <-parent.Done():
			cancel()
		case <-child.Done():
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx, cancel
}

// ═══════════════════════════════════════════════════════════════════════════
// 通道工具函数
// ═══════════════════════════════════════════════════════════════════════════

// TrySend 尝试非阻塞发送到通道
// 如果通道为 nil、已满或已关闭，返回 false
func TrySend[T any](ch chan<- T, value T) bool {
	if ch == nil {
		return false
	}
	select {
	case ch <- value:
		return true
	default:
		return false
	}
}

// TrySendWithContext 带 context 的尝试发送
// 如果 context 取消或通道满，返回 false
func TrySendWithContext[T any](ctx context.Context, ch chan<- T, value T) bool {
	if ch == nil {
		return false
	}
	select {
	case ch <- value:
		return true
	case <-ctx.Done():
		return false
	default:
		return false
	}
}

// SendOrDrop 发送消息，如果通道满则丢弃
// 返回是否成功发送
func SendOrDrop[T any](ch chan<- T, value T) bool {
	return TrySend(ch, value)
}

// ═══════════════════════════════════════════════════════════════════════════
// 错误处理工具
// ═══════════════════════════════════════════════════════════════════════════

// IsContextError 检查错误是否为 context 相关错误
func IsContextError(err error) bool {
	return err == context.Canceled || err == context.DeadlineExceeded
}

// IgnoreContextError 如果是 context 错误则返回 nil
func IgnoreContextError(err error) error {
	if IsContextError(err) {
		return nil
	}
	return err
}
