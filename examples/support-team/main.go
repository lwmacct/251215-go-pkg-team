// Package main 演示客服支持团队的多 Agent 协作
//
// 核心特性：
//   - Leader 驱动：所有任务通过 Leader 协调
//   - 持续事件循环：支持多轮对话
//   - 上下文保持：Agent 自动维护对话历史
//   - Delegate 机制：真正的 Actor 消息委托
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	baseagent "github.com/lwmacct/251215-go-pkg-agent/pkg/agent"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/provider/localmock"
	"github.com/lwmacct/251215-go-pkg-team/pkg/actor"
	"github.com/lwmacct/251215-go-pkg-team/pkg/agent"
	"github.com/lwmacct/251215-go-pkg-team/pkg/team"
)

const defaultTimeout = 5 * time.Second

// Session 会话状态
type Session struct {
	taskID        string
	requestCount  int
	leaderPID     *actor.PID
	leaderID      string
	teamPID       *actor.PID
	members       map[string]*AgentMember
	eventCh       chan *baseagent.AgentEvent
	resultCh      chan string
	mu            sync.Mutex
	lastExpertID  string
}

func main() {
	fmt.Println("🎫 客服支持团队 Demo（持续对话版）")
	fmt.Println(strings.Repeat("━", 50))
	fmt.Println()

	// 1. 初始化
	sys, teamPID, members := setup()
	defer sys.Shutdown()

	// 2. 获取 Leader
	leader, err := team.DoGetLeader(teamPID, defaultTimeout)
	if err != nil {
		fmt.Println("❌ 获取 Leader 失败:", err)
		return
	}
	leaderPID := members[leader.ID()].PID()
	fmt.Printf("👑 团队 Leader: %s\n", leader.Name())

	// 3. 订阅所有 Agent 事件
	eventCh := subscribeAllAgents(members)
	fmt.Printf("🔔 已订阅 %d 个 Agent 的事件流\n", len(members))
	fmt.Println("💬 输入问题开始对话，输入 'exit' 退出")
	fmt.Println()

	// 4. 创建会话
	session := &Session{
		leaderPID: leaderPID,
		leaderID:  leader.ID(),
		teamPID:   teamPID,
		members:   members,
		eventCh:   eventCh,
		resultCh:  make(chan string, 1),
	}

	// 5. 启动事件处理 goroutine
	go handleEvents(session)

	// 6. 持续对话循环
	runDialogLoop(session)

	// 7. 输出会话汇总
	printSessionSummary(session)
}

// setup 初始化 Actor 系统、团队和 Agents
func setup() (*actor.System, *actor.PID, map[string]*AgentMember) {
	sys := actor.NewSystem("support-system")

	teamActor := team.NewTeamActor("support-team",
		team.WithDescription("客服支持团队"),
	)
	teamPID := sys.Spawn(teamActor, "team")
	time.Sleep(50 * time.Millisecond)

	members := createAgents(sys, teamPID)
	return sys, teamPID, members
}

// createAgents 创建所有 Agent 并添加到团队
func createAgents(sys *actor.System, teamPID *actor.PID) map[string]*AgentMember {
	members := make(map[string]*AgentMember)
	roles := []string{RoleTriage, RoleTech, RoleBilling, RoleEscalate}

	for i, role := range roles {
		mockProvider := localmock.New(localmock.WithResponse(MockResponses[role]))

		ag, err := baseagent.New().
			Provider(mockProvider).
			Name(AgentNames[role]).
			System(AgentPrompts[role]).
			Build()
		if err != nil {
			continue
		}

		agentActor := agent.New(ag)
		pid := sys.Spawn(agentActor, role)

		parentID := ""
		if i > 0 {
			parentID = RoleTriage
		}
		member := NewAgentMember(role, pid, parentID)
		members[role] = member

		_ = team.DoAddMember(teamPID, member, defaultTimeout)
	}

	time.Sleep(50 * time.Millisecond)
	return members
}

// subscribeAllAgents 订阅所有 Agent 的事件
func subscribeAllAgents(members map[string]*AgentMember) chan *baseagent.AgentEvent {
	eventCh := make(chan *baseagent.AgentEvent, 100)
	subscriber := &agent.Subscriber{
		ID:        "main-subscriber",
		EventChan: eventCh,
	}

	for _, member := range members {
		member.PID().Tell(&agent.Subscribe{Subscriber: subscriber})
	}

	return eventCh
}

// runDialogLoop 持续对话循环
func runDialogLoop(session *Session) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Println(strings.Repeat("━", 50))
		fmt.Print("📥 请输入问题: ")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("\n👋 感谢使用，再见！")
			break
		}

		// 处理用户请求
		processUserRequest(session, input)

		// 等待结果
		select {
		case result := <-session.resultCh:
			fmt.Println()
			fmt.Println("💬 解决方案:")
			fmt.Printf("   %s\n", result)
		case <-time.After(10 * time.Second):
			fmt.Println("   ⏱️ 处理超时")
		}
	}
}

// processUserRequest 处理用户请求
func processUserRequest(session *Session, input string) {
	session.mu.Lock()
	defer session.mu.Unlock()

	session.requestCount++

	// 创建或复用任务
	if session.taskID == "" {
		taskID, _ := team.DoAddTask(session.teamPID, team.Task{
			Title:       "客户问题",
			Description: input,
			Priority:    2,
		}, defaultTimeout)
		session.taskID = taskID
		_ = team.DoAssignTask(session.teamPID, taskID, session.leaderID, defaultTimeout)
	}

	// 报告进度
	team.DoReportProgress(session.teamPID, session.leaderID, session.taskID,
		fmt.Sprintf("用户请求 #%d: %s", session.requestCount, truncate(input, 30)))

	fmt.Println()
	fmt.Printf("🔍 [Leader: %s] 分析中...\n", AgentNames[session.leaderID])

	// 发送 Run 消息给 Leader
	triageEventCh := make(chan *baseagent.AgentEvent, 16)
	session.leaderPID.Tell(&agent.Run{
		Text:      input,
		Context:   context.Background(),
		EventChan: triageEventCh,
	})

	// 处理 Leader 的直接响应
	go func() {
		for event := range triageEventCh {
			if event.Type == llm.EventTypeDone {
				// 解析分类结果
				result := parseTriageResult(event.Result.Text)
				targetRole := routeToExpert(result.Type)

				session.mu.Lock()
				session.lastExpertID = targetRole
				session.mu.Unlock()

				fmt.Printf("   → 类型: %s, 优先级: %s\n", result.Type, result.Priority)
				fmt.Printf("   → 委托给: %s (Delegate 消息)\n", AgentNames[targetRole])
				fmt.Printf("\n🔧 [%s] 处理中...\n", AgentNames[targetRole])

				// 报告进度
				team.DoReportProgress(session.teamPID, session.leaderID, session.taskID,
					fmt.Sprintf("委托给 %s", AgentNames[targetRole]))

				// Delegate 给专家
				delegateToExpert(session, targetRole, input)
			}
		}
	}()
}

// delegateToExpert 通过 Delegate 消息委托给专家
func delegateToExpert(session *Session, targetRole, userIssue string) {
	_ = team.DoAssignTask(session.teamPID, session.taskID, targetRole, defaultTimeout)

	resultCh := make(chan *agent.DelegateResult, 1)
	session.leaderPID.Tell(&agent.Delegate{
		TargetAgentID: targetRole,
		Task:          userIssue,
		Context:       context.Background(),
		ResultChan:    resultCh,
	})

	fmt.Printf("   ⚡ 发送 Delegate 消息\n")

	go func() {
		result := <-resultCh
		if result.Error != nil {
			session.resultCh <- fmt.Sprintf("处理失败: %v", result.Error)
			return
		}

		fmt.Printf("   ⚡ 收到 Delegate 结果\n")

		// 报告完成
		team.DoReportProgress(session.teamPID, targetRole, session.taskID, "问题已处理")

		// 发送结果
		session.resultCh <- result.Result.Text
	}()
}

// handleEvents 处理全局事件（可扩展用于日志、监控等）
func handleEvents(session *Session) {
	for event := range session.eventCh {
		switch event.Type {
		case llm.EventTypeError:
			fmt.Printf("   ❌ 错误: %v\n", event.Error)
		}
	}
}

// TriageResult 分诊结果
type TriageResult struct {
	Type     string `json:"type"`
	Priority string `json:"priority"`
	Summary  string `json:"summary"`
}

// parseTriageResult 解析分诊结果
func parseTriageResult(text string) *TriageResult {
	var result TriageResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return &TriageResult{Type: "technical", Priority: "medium", Summary: "账户问题"}
	}
	return &result
}

// routeToExpert 根据问题类型路由到专家
func routeToExpert(issueType string) string {
	switch issueType {
	case "billing":
		return RoleBilling
	case "escalate":
		return RoleEscalate
	default:
		return RoleTech
	}
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// printSessionSummary 输出会话汇总
func printSessionSummary(session *Session) {
	fmt.Println()
	fmt.Println("📊 会话汇总")
	fmt.Println(strings.Repeat("━", 50))

	summary, err := team.DoGetSummary(session.teamPID, defaultTimeout)
	if err == nil {
		fmt.Printf("   团队: %s\n", summary.Name)
		fmt.Printf("   成员数: %d\n", summary.MemberCount)
		fmt.Printf("   Leader: %s\n", summary.LeaderName)
		fmt.Println()
	}

	fmt.Printf("   处理请求数: %d\n", session.requestCount)

	logs, err := team.DoGetProgressLog(session.teamPID, 20, defaultTimeout)
	if err == nil {
		fmt.Printf("   进度记录: %d 条\n", len(logs))
	}
}
