package agent

import (
	"context"
	"time"

	agentpkg "github.com/lwmacct/251215-go-pkg-agent/pkg/agent"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-team/pkg/actor"
)

// DoChat 向 Agent Actor 发送对话请求并等待响应
//
// 这是一个便捷函数，封装了创建 channel、发送消息、等待响应的流程。
func DoChat(pid *actor.PID, text string, timeout time.Duration) (*agentpkg.Result, error) {
	replyCh := make(chan *ChatResult, 1)

	pid.Tell(&Chat{
		Text:      text,
		Context:   context.Background(),
		ReplyChan: replyCh,
	})

	select {
	case result := <-replyCh:
		return result.Result, result.Error
	case <-time.After(timeout):
		return nil, context.DeadlineExceeded
	}
}

// DoGetStatus 获取 Agent 状态
func DoGetStatus(pid *actor.PID, timeout time.Duration) (*agentpkg.Status, error) {
	replyCh := make(chan *agentpkg.Status, 1)

	pid.Tell(&GetStatus{
		ReplyChan: replyCh,
	})

	select {
	case status := <-replyCh:
		return status, nil
	case <-time.After(timeout):
		return nil, context.DeadlineExceeded
	}
}

// DoGetMessages 获取 Agent 消息历史
func DoGetMessages(pid *actor.PID, timeout time.Duration) ([]llm.Message, error) {
	replyCh := make(chan []llm.Message, 1)

	pid.Tell(&GetMessages{
		ReplyChan: replyCh,
	})

	select {
	case messages := <-replyCh:
		return messages, nil
	case <-time.After(timeout):
		return nil, context.DeadlineExceeded
	}
}
