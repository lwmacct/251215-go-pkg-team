package agent_test

import (
	"context"
	"fmt"
	"time"

	baseagent "github.com/lwmacct/251215-go-pkg-agent/pkg/agent"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/provider/localmock"
	"github.com/lwmacct/251215-go-pkg-team/pkg/actor"
	"github.com/lwmacct/251215-go-pkg-team/pkg/agent"
)

// Example_basic 演示 AgentActor 的基本使用
//
// 使用 Mock Provider 创建可测试的 AgentActor。
func Example_basic() {
	// 创建 Actor 系统
	sys := actor.NewSystem("example")
	defer sys.Shutdown()

	// 创建 Mock Provider
	mockProvider := localmock.New(localmock.WithResponse("Hello from Mock!"))
	defer func() { _ = mockProvider.Close() }()

	// 创建 Agent（使用 Builder 模式）
	ag, err := baseagent.New().
		Provider(mockProvider).
		Name("example-agent").
		System("You are a test assistant.").
		Build()
	if err != nil {
		fmt.Println("Failed to create agent:", err)
		return
	}

	// 包装为 AgentActor 并启动
	agentActor := agent.New(ag)
	pid := sys.Spawn(agentActor, "assistant")

	// 等待启动
	time.Sleep(50 * time.Millisecond)

	// 发送对话请求
	result, err := agent.DoChat(pid, "Hello", 5*time.Second)
	if err != nil {
		fmt.Println("Chat failed:", err)
		return
	}

	fmt.Println("Response:", result.Text)

	// 停止 Actor
	pid.Tell(&agent.Stop{Reason: "done"})
	time.Sleep(50 * time.Millisecond)

	fmt.Println("Done")

	// Output:
	// Response: Hello from Mock!
	// Done
}

// Example_factory 演示使用 Factory 批量创建 AgentActor
func Example_factory() {
	// 创建 Actor 系统
	sys := actor.NewSystem("factory-example")
	defer sys.Shutdown()

	// 创建 Mock Provider
	mockProvider := localmock.New(localmock.WithResponse("Factory response"))
	defer func() { _ = mockProvider.Close() }()

	// 创建 Factory（共享 Provider 和默认配置）
	factory := agent.NewFactory(
		mockProvider,
		baseagent.WithPrompt("You are a helpful assistant."),
	)

	// 使用 Factory 创建多个 Agent
	pid1, err := factory.CreateAndSpawn(sys, "agent-1")
	if err != nil {
		fmt.Println("Failed:", err)
		return
	}

	pid2, err := factory.CreateAndSpawn(sys, "agent-2",
		baseagent.WithName("CustomName"), // 可覆盖默认配置
	)
	if err != nil {
		fmt.Println("Failed:", err)
		return
	}

	time.Sleep(50 * time.Millisecond)

	fmt.Printf("Created agents: %s, %s\n", pid1.ID, pid2.ID)
	fmt.Printf("Total actors: %d\n", sys.Count())

	// Output:
	// Created agents: agent-1, agent-2
	// Total actors: 2
}

// Example_doChat 演示同步对话辅助函数
func Example_doChat() {
	sys := actor.NewSystem("chat-example")
	defer sys.Shutdown()

	mockProvider := localmock.New(localmock.WithResponse("Hello!"))
	ag, _ := baseagent.NewAgent(
		baseagent.WithProvider(mockProvider),
		baseagent.WithName("ChatBot"),
	)

	agentActor := agent.New(ag)
	pid := sys.Spawn(agentActor, "chatbot")
	time.Sleep(50 * time.Millisecond)

	// DoChat 封装了 channel 创建和超时处理
	result, err := agent.DoChat(pid, "Hi there!", 5*time.Second)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Bot:", result.Text)

	// Output:
	// Bot: Hello!
}

// Example_chatRawMessage 演示直接发送 Chat 消息（底层方式）
func Example_chatRawMessage() {
	msg := &agent.Chat{
		Text:    "Hello, World!",
		Context: context.Background(),
	}

	fmt.Printf("Message kind: %s\n", msg.Kind())

	// Output:
	// Message kind: agent.chat
}

// Example_getStatus 演示状态查询消息
func Example_getStatus() {
	msg := &agent.GetStatus{}

	fmt.Printf("Message kind: %s\n", msg.Kind())

	// Output:
	// Message kind: agent.get_status
}

// Example_stop 演示停止消息
func Example_stop() {
	msg := &agent.Stop{
		Reason: "User requested shutdown",
	}

	fmt.Printf("Message kind: %s, reason: %s\n", msg.Kind(), msg.Reason)

	// Output:
	// Message kind: agent.stop, reason: User requested shutdown
}
