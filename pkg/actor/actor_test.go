package actor

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============== 测试消息类型 ==============

type PingMessage struct{}

func (p *PingMessage) Kind() string { return "ping" }

type PongMessage struct{}

func (p *PongMessage) Kind() string { return "pong" }

type CountMessage struct {
	Value int
}

func (c *CountMessage) Kind() string { return "count" }

type EchoMessage struct {
	Text string
}

func (e *EchoMessage) Kind() string { return "echo" }

type PanicMessage struct{}

func (p *PanicMessage) Kind() string { return "panic" }

// ============== 测试 Actor ==============

type EchoActor struct {
	BaseActor
	received []Message
	mu       sync.Mutex
}

func (a *EchoActor) Receive(ctx *Context, msg Message) {
	a.mu.Lock()
	a.received = append(a.received, msg)
	a.mu.Unlock()

	// 如果有发送者，回复消息
	if ctx.Sender != nil {
		ctx.Reply(msg)
	}
}

func (a *EchoActor) ReceivedCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.received)
}

type CounterActor struct {
	BaseActor
	count int32
}

func (a *CounterActor) Receive(ctx *Context, msg Message) {
	switch msg.(type) {
	case *CountMessage:
		atomic.AddInt32(&a.count, 1)
	case *Started:
		// 忽略启动消息
	}
}

func (a *CounterActor) Count() int32 {
	return atomic.LoadInt32(&a.count)
}

type RequestResponseActor struct {
	BaseActor
}

func (a *RequestResponseActor) Receive(ctx *Context, msg Message) {
	switch m := msg.(type) {
	case *PingMessage:
		ctx.Reply(&PongMessage{})
	case *EchoMessage:
		ctx.Reply(&EchoMessage{Text: "Echo: " + m.Text})
	}
}

type PanicActor struct {
	BaseActor
	panicCount int32
}

func (a *PanicActor) Receive(_ *Context, msg Message) {
	switch msg.(type) {
	case *PanicMessage:
		atomic.AddInt32(&a.panicCount, 1)
		panic("intentional panic")
	}
}

// ============== 测试用例 ==============

func TestNewSystem(t *testing.T) {
	sys := NewSystem("test")
	require.NotNil(t, sys)
	assert.Equal(t, "test", sys.Name())
	assert.True(t, sys.IsRunning())

	sys.Shutdown()
	assert.False(t, sys.IsRunning())
}

func TestSpawnActor(t *testing.T) {
	sys := NewSystem("test")
	defer sys.Shutdown()

	actor := &EchoActor{}
	pid := sys.Spawn(actor, "echo")

	require.NotNil(t, pid)
	assert.Equal(t, "echo", pid.ID)

	// 等待 Started 消息被处理
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, actor.ReceivedCount()) // Started 消息
}

func TestSendMessage(t *testing.T) {
	sys := NewSystem("test")
	defer sys.Shutdown()

	actor := &CounterActor{}
	pid := sys.Spawn(actor, "counter")

	// 发送消息
	for i := 0; i < 10; i++ {
		pid.Tell(&CountMessage{Value: i})
	}

	// 等待消息被处理
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(10), actor.Count())
}

func TestTrySend(t *testing.T) {
	sys := NewSystem("test")
	defer sys.Shutdown()

	actor := &EchoActor{}
	pid := sys.Spawn(actor, "echo")

	// TrySend 应该成功
	ok := pid.TrySend(&PingMessage{})
	assert.True(t, ok)

	time.Sleep(50 * time.Millisecond)
}

func TestRequestResponse(t *testing.T) {
	sys := NewSystem("test")
	defer sys.Shutdown()

	actor := &RequestResponseActor{}
	pid := sys.Spawn(actor, "responder")

	// 等待 Actor 启动
	time.Sleep(50 * time.Millisecond)

	// 发送请求并等待响应
	resp, err := pid.Request(&PingMessage{}, 1*time.Second)
	require.NoError(t, err)
	require.NotNil(t, resp)

	_, ok := resp.(*PongMessage)
	assert.True(t, ok, "expected PongMessage")
}

func TestRequestResponseWithPayload(t *testing.T) {
	sys := NewSystem("test")
	defer sys.Shutdown()

	actor := &RequestResponseActor{}
	pid := sys.Spawn(actor, "responder")

	time.Sleep(50 * time.Millisecond)

	resp, err := pid.Request(&EchoMessage{Text: "Hello"}, 1*time.Second)
	require.NoError(t, err)
	require.NotNil(t, resp)

	echoResp, ok := resp.(*EchoMessage)
	assert.True(t, ok)
	assert.Equal(t, "Echo: Hello", echoResp.Text)
}

func TestRequestTimeout(t *testing.T) {
	sys := NewSystem("test")
	defer sys.Shutdown()

	// Actor 不回复消息
	actor := &EchoActor{}
	pid := sys.Spawn(actor, "silent")

	time.Sleep(50 * time.Millisecond)

	// 应该超时
	_, err := pid.Request(&PingMessage{}, 100*time.Millisecond)
	require.Error(t, err)

	_, ok := err.(*ResponseTimeout)
	assert.True(t, ok, "expected ResponseTimeout error")
}

func TestBroadcast(t *testing.T) {
	sys := NewSystem("test")
	defer sys.Shutdown()

	actors := make([]*CounterActor, 5)
	for i := 0; i < 5; i++ {
		actors[i] = &CounterActor{}
		sys.Spawn(actors[i], "counter-"+string(rune('a'+i)))
	}

	time.Sleep(50 * time.Millisecond)

	// 广播消息
	sys.Broadcast(&CountMessage{Value: 1})

	time.Sleep(100 * time.Millisecond)

	// 所有 Actor 都应该收到消息
	for i, actor := range actors {
		assert.Equal(t, int32(1), actor.Count(), "actor %d should have received message", i)
	}
}

func TestBroadcastWithFilter(t *testing.T) {
	sys := NewSystem("test")
	defer sys.Shutdown()

	actorA := &CounterActor{}
	actorB := &CounterActor{}
	actorC := &CounterActor{}

	sys.Spawn(actorA, "counter-a")
	sys.Spawn(actorB, "counter-b")
	sys.Spawn(actorC, "other-c")

	time.Sleep(50 * time.Millisecond)

	// 只广播给 counter-* Actor
	sys.BroadcastWithFilter(&CountMessage{Value: 1}, func(pid *PID) bool {
		return len(pid.ID) > 7 && pid.ID[:7] == "counter"
	})

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(1), actorA.Count())
	assert.Equal(t, int32(1), actorB.Count())
	assert.Equal(t, int32(0), actorC.Count()) // 被过滤
}

func TestActorFunc(t *testing.T) {
	sys := NewSystem("test")
	defer sys.Shutdown()

	var received atomic.Int32

	// 使用函数式 Actor
	actor := ActorFunc(func(ctx *Context, msg Message) {
		if _, ok := msg.(*PingMessage); ok {
			received.Add(1)
		}
	})

	pid := sys.Spawn(actor, "func-actor")

	pid.Tell(&PingMessage{})
	pid.Tell(&PingMessage{})
	pid.Tell(&PingMessage{})

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(3), received.Load())
}

func TestStopActor(t *testing.T) {
	sys := NewSystem("test")
	defer sys.Shutdown()

	actor := &EchoActor{}
	pid := sys.Spawn(actor, "echo")

	time.Sleep(50 * time.Millisecond)

	// 停止 Actor
	sys.Stop(pid)

	time.Sleep(100 * time.Millisecond)

	// Actor 应该不存在了
	_, ok := sys.GetActor("echo")
	assert.False(t, ok)
}

func TestStopGracefully(t *testing.T) {
	sys := NewSystem("test")
	defer sys.Shutdown()

	actor := &EchoActor{}
	pid := sys.Spawn(actor, "echo")

	time.Sleep(50 * time.Millisecond)

	err := sys.StopGracefully(pid, 1*time.Second)
	assert.NoError(t, err)

	_, ok := sys.GetActor("echo")
	assert.False(t, ok)
}

func TestListActors(t *testing.T) {
	sys := NewSystem("test")
	defer sys.Shutdown()

	sys.Spawn(&EchoActor{}, "actor-1")
	sys.Spawn(&EchoActor{}, "actor-2")
	sys.Spawn(&EchoActor{}, "actor-3")

	time.Sleep(50 * time.Millisecond)

	pids := sys.ListActors()
	assert.Len(t, pids, 3)
	assert.Equal(t, 3, sys.Count())
}

func TestStats(t *testing.T) {
	sys := NewSystem("test")
	defer sys.Shutdown()

	actor := &CounterActor{}
	pid := sys.Spawn(actor, "counter")

	// 发送消息
	for i := 0; i < 100; i++ {
		pid.Tell(&CountMessage{Value: i})
	}

	time.Sleep(200 * time.Millisecond)

	stats := sys.Stats()
	assert.Equal(t, int64(1), stats.TotalActors)
	assert.GreaterOrEqual(t, stats.TotalMessages, int64(100))
}

func TestSimpleMessage(t *testing.T) {
	msg := NewSimpleMessage("test.event", map[string]string{"key": "value"})
	assert.Equal(t, "test.event", msg.Kind())
	assert.Equal(t, map[string]string{"key": "value"}, msg.Payload)
}

func TestPIDString(t *testing.T) {
	pid := &PID{ID: "test-actor"}
	assert.Equal(t, "test-actor", pid.String())

	pid2 := &PID{ID: "remote-actor", Address: "localhost:8080"}
	assert.Equal(t, "remote-actor@localhost:8080", pid2.String())
}

func TestSpawnDuplicate(t *testing.T) {
	sys := NewSystem("test")
	defer sys.Shutdown()

	actor1 := &EchoActor{}
	actor2 := &EchoActor{}

	pid1 := sys.Spawn(actor1, "echo")
	pid2 := sys.Spawn(actor2, "echo") // 重复名称

	// 应该返回相同的 PID
	assert.Equal(t, pid1.ID, pid2.ID)
	assert.Equal(t, 1, sys.Count())
}

// ============== 监督策略测试 ==============

func TestOneForOneStrategy(t *testing.T) {
	strategy := NewOneForOneStrategy(3, time.Minute, DefaultDecider)

	// 前3次应该返回 Restart
	for i := 0; i < 3; i++ {
		result := strategy.HandleFailure(nil, nil, nil, "error")
		assert.Equal(t, DirectiveRestart, result)
	}

	// 第4次应该返回 Stop
	result := strategy.HandleFailure(nil, nil, nil, "error")
	assert.Equal(t, DirectiveStop, result)
}

func TestExponentialBackoffStrategy(t *testing.T) {
	strategy := NewExponentialBackoffStrategy(
		100*time.Millisecond,
		1*time.Second,
		3,
		DefaultDecider,
	)

	// 第一次应该返回 100ms 延迟
	result := strategy.HandleFailure(nil, nil, nil, "error")
	dwd, ok := result.(DirectiveWithDelay)
	require.True(t, ok)
	assert.Equal(t, DirectiveRestart, dwd.Directive)
	assert.Equal(t, 100*time.Millisecond, dwd.Delay)

	// 第二次应该返回 200ms 延迟
	result = strategy.HandleFailure(nil, nil, nil, "error")
	dwd, ok = result.(DirectiveWithDelay)
	require.True(t, ok)
	assert.Equal(t, 200*time.Millisecond, dwd.Delay)

	// 第三次应该返回 400ms 延迟
	result = strategy.HandleFailure(nil, nil, nil, "error")
	dwd, ok = result.(DirectiveWithDelay)
	require.True(t, ok)
	assert.Equal(t, 400*time.Millisecond, dwd.Delay)

	// 第四次应该返回 Stop
	result = strategy.HandleFailure(nil, nil, nil, "error")
	assert.Equal(t, DirectiveStop, result)
}

func TestDefaultDeciders(t *testing.T) {
	assert.Equal(t, DirectiveRestart, DefaultDecider("error"))
	assert.Equal(t, DirectiveStop, StoppingDecider("error"))
	assert.Equal(t, DirectiveEscalate, EscalatingDecider("error"))
	assert.Equal(t, DirectiveResume, ResumingDecider("error"))
}

func TestDirectiveString(t *testing.T) {
	assert.Equal(t, "Resume", DirectiveResume.String())
	assert.Equal(t, "Restart", DirectiveRestart.String())
	assert.Equal(t, "Stop", DirectiveStop.String())
	assert.Equal(t, "Escalate", DirectiveEscalate.String())
}

// ============== 并发测试 ==============

func TestConcurrentSend(t *testing.T) {
	sys := NewSystem("test")
	defer sys.Shutdown()

	actor := &CounterActor{}
	pid := sys.Spawn(actor, "counter")

	time.Sleep(50 * time.Millisecond)

	// 并发发送消息
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pid.Tell(&CountMessage{Value: 1})
		}()
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, int32(100), actor.Count())
}

func TestConcurrentSpawn(t *testing.T) {
	sys := NewSystem("test")
	defer sys.Shutdown()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			actor := &EchoActor{}
			sys.Spawn(actor, "actor-"+string(rune('0'+idx%10))+string(rune('0'+idx/10)))
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	// 应该创建了所有不重复的 Actor
	assert.LessOrEqual(t, sys.Count(), 50)
}
