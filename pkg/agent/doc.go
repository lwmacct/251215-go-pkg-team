// Package agent 提供 AI Agent 的 Actor 适配器
//
// 本包将 [agentpkg.Agent] 包装为 Actor，提供并发安全的消息驱动执行模型。
// Agent 核心功能独立于 Actor 系统，本包是可选的集成层。
//
// # 设计理念
//
// Agent 和 Actor 是两个独立的概念：
//   - 外部 pkg/agent: AI Agent 核心实现，可独立使用
//   - pkg/actor: Actor 并发模型，管理生命周期和消息传递
//   - 本包: 桥接两者，提供并发安全的 Agent 执行
//
// # 核心组件
//
// [AgentActor] 是适配器，将 Agent 包装为 Actor：
//
//	provider, _ := factory.CreateProvider(nil)
//	ag, _ := baseagent.NewAgent(baseagent.WithProvider(provider))
//	agentActor := agent.New(ag)
//	pid := sys.Spawn(agentActor, "assistant")
//
// # 消息类型
//
// 核心执行消息：
//   - [Run]: 执行对话，返回事件流（支持流式/非流式模式）
//   - [Chat]: 同步对话，等待响应（便捷方法，使用非流式模式）
//
// 状态查询消息：
//   - [GetStatus]: 获取 Agent 状态快照
//   - [GetMessages]: 获取消息历史
//
// 生命周期消息：
//   - [Stop]: 优雅停止 Agent
//
// 事件订阅消息（可扩展）：
//   - [Subscribe]: 订阅 Agent 事件
//   - [Unsubscribe]: 取消订阅
//
// 多 Agent 协作消息（可扩展）：
//   - [Delegate]: 委托任务给其他 Agent
//   - [Handoff]: 任务移交
//   - [AgentEvent]: Agent 事件通知
//
// # 使用示例
//
// 非流式对话（默认）：
//
//	eventCh := make(chan *baseagent.AgentEvent, 16)
//	pid.Tell(&agent.Run{Text: "1+1=?", EventChan: eventCh})
//	for event := range eventCh {
//	    if event.Type == llm.EventTypeDone {
//	        fmt.Println(event.Result.Text)
//	    }
//	}
//
// 流式对话：
//
//	eventCh := make(chan *baseagent.AgentEvent, 16)
//	pid.Tell(&agent.Run{
//	    Text:      "写一篇文章",
//	    EventChan: eventCh,
//	    Options:   []baseagent.RunOption{baseagent.WithStreaming(true)},
//	})
//	for event := range eventCh {
//	    switch event.Type {
//	    case llm.EventTypeText:
//	        fmt.Print(event.Text)  // 逐字输出
//	    case llm.EventTypeDone:
//	        fmt.Println("\n完成")
//	    }
//	}
//
// 同步对话：
//
//	replyCh := make(chan *agent.ChatResult, 1)
//	pid.Tell(&agent.Chat{Text: "Hello", ReplyChan: replyCh})
//	result := <-replyCh
//
// 事件订阅：
//
//	// 订阅所有事件
//	monitorCh := make(chan *baseagent.AgentEvent, 32)
//	pid.Tell(&agent.Subscribe{
//	    Subscriber: &agent.Subscriber{
//	        ID:        "monitor",
//	        EventChan: monitorCh,
//	    },
//	})
//
// 任务委托：
//
//	resultCh := make(chan *agent.DelegateResult, 1)
//	pid.Tell(&agent.Delegate{
//	    TargetAgentID: "researcher",
//	    Task:          "Search for X",
//	    ResultChan:    resultCh,
//	})
//
// # 线程安全
//
// AgentActor 通过 Actor 模型保证线程安全：
//   - 消息串行处理，无需加锁
//   - 状态修改在单一 goroutine 中进行
//   - 适合高并发场景
//
// 完整使用示例请参考 example_test.go 或运行 go doc -all。
package agent
