# go-pkg-team

基于 Actor 模型的多 AI Agent 团队协作 SDK。

<!--TOC-->

- [架构](#架构) `:12+34`
- [包结构](#包结构) `:46+14`
- [快速开始](#快速开始) `:60+14`
  - [初始化开发环境](#初始化开发环境) `:62+6`
  - [查看所有可用任务](#查看所有可用任务) `:68+6`
- [相关链接](#相关链接) `:74+4`

<!--TOC-->

## 架构

```
┌─────────────────────────────────────────────────────────────┐
│                      Actor System                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐                                           │
│  │  TeamActor   │ ← team.NewTeamActor("my-team")            │
│  └──────┬───────┘                                           │
│         │                                                    │
│    ┌────┴────┬─────────────┐                                │
│    ▼         ▼             ▼                                │
│ ┌──────┐ ┌──────┐     ┌────────┐                           │
│ │Member│ │Member│ ... │TaskMgr │                           │
│ │      │ │      │     │        │                           │
│ └──┬───┘ └──┬───┘     └────────┘                           │
│    │        │                                               │
│    ▼        ▼                                               │
│ ┌──────┐ ┌──────┐                                          │
│ │Agent │ │Agent │  ← agent.New(外部 Agent)                  │
│ │Actor │ │Actor │                                          │
│ └──────┘ └──────┘                                          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

核心设计原则：

- **无锁并发**：所有状态通过 Actor 邮箱管理，无需 RWMutex
- **消息驱动**：通过类型化消息操作（请求-响应 / 即发即忘）
- **容错机制**：监督策略支持崩溃恢复
- **SDK 友好**：扁平包结构，所有类型在根级别

## 包结构

```
pkg/
├── actor/          # Actor 模型基础设施
│   ├── system.go   # Actor 系统
│   ├── types.go    # Message, PID, Context, Props
│   └── supervisor.go
│
├── agent/          # Agent Actor 适配器（包装外部 Agent）
│   ├── agent.go    # AgentActor 实现
│   ├── messages.go # Run, Chat, GetStatus, Subscribe...
│   └── factory.go  # 批量创建工厂
│
└── team/           # 多 Agent 团队协调
    ├── types.go    # Task, Progress, Report, Summary
    ├── messages.go # AddMember, AssignTask, GetSummary...
    ├── actor.go    # TeamActor（基于 Actor 的实现）
    └── team.go     # Team 接口定义
```

## 快速开始

### 初始化开发环境

```shell
pre-commit install
```

### 查看所有可用任务

```shell
task -a
```

## 示例

### 客服支持团队

演示 Actor 消息驱动的多 Agent 协作：

```
examples/support-team/
├── main.go      # 事件驱动架构
└── agents.go    # Agent 角色定义
```

**运行示例**：

```shell
go run ./examples/support-team/
```

**核心特性**：

- **事件订阅** - 通过 `agent.Subscribe` 统一监听所有 Agent 事件
- **Delegate 机制** - 分诊 Agent 通过消息委托任务给专家
- **异步非阻塞** - 事件循环处理，无同步等待

**架构流程**：

```
用户问题 → Triage Agent (分诊)
              ↓ agent.Run
         解析分类结果
              ↓ agent.Delegate
         Expert Agent (专家处理)
              ↓ DelegateResult
         输出解决方案
```

**消息流示例**：

```go
// 1. 订阅所有 Agent 事件
eventCh := make(chan *agentpkg.AgentEvent, 100)
for _, member := range members {
    member.PID().Tell(&agent.Subscribe{
        Subscriber: &agent.Subscriber{ID: "main", EventChan: eventCh},
    })
}

// 2. 发送 Run 消息启动分诊
triagePID.Tell(&agent.Run{
    Text:      userIssue,
    EventChan: triageEventCh,
})

// 3. 分诊完成后通过 Delegate 委托给专家
triagePID.Tell(&agent.Delegate{
    TargetAgentID: "tech-support",  // Actor ID
    Task:          userIssue,
    ResultChan:    resultCh,
})

// 4. 异步接收结果
result := <-resultCh
fmt.Println(result.Result.Text)
```

## 相关链接

- 使用 [Taskfile](https://taskfile.dev) 管理项目 CLI
- 使用 [Pre-commit](https://pre-commit.com/) 管理和维护多语言预提交钩子
