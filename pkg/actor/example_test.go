package actor_test

import (
	"fmt"
	"time"

	"github.com/lwmacct/251215-go-pkg-team/pkg/actor"
)

// PingMessage 示例消息类型
type PingMessage struct{}

func (m *PingMessage) Kind() string { return "ping" }

// PongMessage 示例响应消息
type PongMessage struct{}

func (m *PongMessage) Kind() string { return "pong" }

// CountMessage 计数器消息
type CountMessage struct {
	Value int
}

func (m *CountMessage) Kind() string { return "count" }

// Example_basic 演示 Actor 系统的基本使用
func Example_basic() {
	// 创建 Actor 系统
	sys := actor.NewSystem("example")
	defer sys.Shutdown()

	// 使用 ActorFunc 快速创建 Actor
	pid := sys.Spawn(actor.ActorFunc(func(ctx *actor.Context, msg actor.Message) {
		switch msg.(type) {
		case *actor.Started:
			fmt.Println("Actor started")
		case *PingMessage:
			fmt.Println("Received Ping")
		}
	}), "greeter")

	// 等待 Actor 启动
	time.Sleep(10 * time.Millisecond)

	// 发送消息
	pid.Tell(&PingMessage{})
	time.Sleep(10 * time.Millisecond)

	// Output:
	// Actor started
	// Received Ping
}

// Example_actorFunc 演示函数式 Actor
func Example_actorFunc() {
	sys := actor.NewSystem("func-example")
	defer sys.Shutdown()

	counter := 0
	pid := sys.Spawn(actor.ActorFunc(func(ctx *actor.Context, msg actor.Message) {
		if m, ok := msg.(*CountMessage); ok {
			counter += m.Value
			fmt.Printf("Counter: %d\n", counter)
		}
	}), "counter")

	time.Sleep(10 * time.Millisecond)

	pid.Tell(&CountMessage{Value: 1})
	pid.Tell(&CountMessage{Value: 2})
	pid.Tell(&CountMessage{Value: 3})
	time.Sleep(50 * time.Millisecond)

	// Output:
	// Counter: 1
	// Counter: 3
	// Counter: 6
}

// Example_pidRequest 演示同步请求响应模式
func Example_pidRequest() {
	sys := actor.NewSystem("request-example")
	defer sys.Shutdown()

	// 创建能够响应的 Actor
	pid := sys.Spawn(actor.ActorFunc(func(ctx *actor.Context, msg actor.Message) {
		if _, ok := msg.(*PingMessage); ok {
			ctx.Reply(&PongMessage{})
		}
	}), "responder")

	time.Sleep(10 * time.Millisecond)

	// 同步请求
	resp, err := pid.Request(&PingMessage{}, time.Second)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", resp.Kind())

	// Output:
	// Response: pong
}

// Example_systemBroadcast 演示广播消息
func Example_systemBroadcast() {
	sys := actor.NewSystem("broadcast-example")
	defer sys.Shutdown()

	// 创建多个 Actor
	for i := 1; i <= 3; i++ {
		name := fmt.Sprintf("worker-%d", i)
		sys.Spawn(actor.ActorFunc(func(ctx *actor.Context, msg actor.Message) {
			if _, ok := msg.(*PingMessage); ok {
				fmt.Printf("%s received ping\n", ctx.Self.ID)
			}
		}), name)
	}

	time.Sleep(10 * time.Millisecond)

	// 广播消息给所有 Actor
	sys.Broadcast(&PingMessage{})
	time.Sleep(50 * time.Millisecond)

	// Unordered output:
	// worker-1 received ping
	// worker-2 received ping
	// worker-3 received ping
}

// Example_newOneForOneStrategy 演示一对一监督策略
func Example_newOneForOneStrategy() {
	// 创建监督策略：在 1 分钟内最多允许 3 次重启
	strategy := actor.NewOneForOneStrategy(
		3,                    // 最大重启次数
		time.Minute,          // 时间窗口
		actor.DefaultDecider, // 使用默认决策器
	)

	// 使用策略创建 Actor
	sys := actor.NewSystem("supervisor-example")
	defer sys.Shutdown()

	props := actor.DefaultProps("worker").WithSupervisor(strategy)

	sys.SpawnWithProps(actor.ActorFunc(func(ctx *actor.Context, msg actor.Message) {
		switch msg.(type) {
		case *actor.Started:
			fmt.Println("Worker started")
		case *actor.Restarting:
			fmt.Println("Worker restarting")
		}
	}), props)

	time.Sleep(10 * time.Millisecond)

	// Output:
	// Worker started
}

// Example_newExponentialBackoffStrategy 演示指数退避策略
func Example_newExponentialBackoffStrategy() {
	strategy := actor.NewExponentialBackoffStrategy(
		100*time.Millisecond, // 初始延迟
		10*time.Second,       // 最大延迟
		5,                    // 最大重启次数
		actor.DefaultDecider,
	)

	fmt.Printf("Strategy type: %T\n", strategy)

	// Output:
	// Strategy type: *actor.ExponentialBackoffStrategy
}

// Example_defaultProps 演示 Props 配置
func Example_defaultProps() {
	props := actor.DefaultProps("my-actor").
		WithMailboxSize(1000).
		WithSupervisor(actor.DefaultSupervisorStrategy())

	fmt.Printf("Name: %s, MailboxSize: %d\n", props.Name, props.MailboxSize)

	// Output:
	// Name: my-actor, MailboxSize: 1000
}

// Example_context 演示 Actor 上下文的使用
func Example_context() {
	sys := actor.NewSystem("context-example")
	defer sys.Shutdown()

	pid := sys.Spawn(actor.ActorFunc(func(ctx *actor.Context, msg actor.Message) {
		switch msg.(type) {
		case *actor.Started:
			fmt.Printf("Self: %s\n", ctx.Self.ID)
			fmt.Printf("System: %s\n", ctx.System().Name())
		}
	}), "demo")

	_ = pid
	time.Sleep(10 * time.Millisecond)

	// Output:
	// Self: demo
	// System: context-example
}

// Example_simpleMessage 演示简单消息的使用
func Example_simpleMessage() {
	msg := actor.NewSimpleMessage("greeting", "Hello, World!")

	fmt.Printf("Kind: %s\n", msg.Kind())
	fmt.Printf("Payload: %v\n", msg.Payload)

	// Output:
	// Kind: greeting
	// Payload: Hello, World!
}
