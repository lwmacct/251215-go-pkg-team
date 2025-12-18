package team_test

import (
	"fmt"
	"time"

	"github.com/lwmacct/251215-go-pkg-team/pkg/actor"
	"github.com/lwmacct/251215-go-pkg-team/pkg/team"
)

// mockMember 测试用的 Mock 成员
type mockMember struct {
	id       string
	name     string
	parentID string
}

func (m *mockMember) ID() string       { return m.id }
func (m *mockMember) Name() string     { return m.name }
func (m *mockMember) ParentID() string { return m.parentID }

// 确保 mockMember 实现了 Member 接口
var _ team.Member = (*mockMember)(nil)

// Example_basic 演示 Team 的基本用法
func Example_basic() {
	// 创建团队
	myTeam := team.New(
		team.WithName("dev-team"),
		team.WithDescription("Development team for project X"),
	)

	// 创建成员
	leader := &mockMember{id: "agent-1", name: "Leader Agent"}
	worker := &mockMember{id: "agent-2", name: "Worker Agent", parentID: "agent-1"}

	// 添加成员（第一个成员自动成为 Leader）
	_ = myTeam.AddMember(leader)
	_ = myTeam.AddMember(worker)

	// 检查 Leader
	fmt.Printf("Leader: %s\n", myTeam.GetLeader().Name())
	fmt.Printf("Is agent-1 leader: %v\n", myTeam.IsLeader("agent-1"))

	// Output:
	// Leader: Leader Agent
	// Is agent-1 leader: true
}

// Example_task 演示任务管理
func Example_task() {
	myTeam := team.New(team.WithName("task-demo"))

	// 添加成员
	worker := &mockMember{id: "worker-1", name: "Task Worker"}
	_ = myTeam.AddMember(worker)

	// 添加任务
	taskID := myTeam.AddTask(team.Task{
		Title:       "Implement feature",
		Description: "Implement the new feature X",
		Priority:    3,
	})

	// 分配任务
	_ = myTeam.AssignTask(taskID, "worker-1")

	// 更新任务状态
	_ = myTeam.UpdateTaskStatus(taskID, team.TaskStatusInProgress, "worker-1")

	// 获取任务
	task, _ := myTeam.GetTask(taskID)
	fmt.Printf("Task: %s, Status: %s, Assigned to: %s\n",
		task.Title, task.Status, task.AssignedTo)

	// Output:
	// Task: Implement feature, Status: in_progress, Assigned to: worker-1
}

// Example_progress 演示进度追踪
func Example_progress() {
	myTeam := team.New(team.WithName("progress-demo"))

	// 添加成员
	worker := &mockMember{id: "worker-1", name: "Progress Worker"}
	_ = myTeam.AddMember(worker)

	// 报告进度
	myTeam.ReportProgress("worker-1", "", "Started analyzing requirements")
	myTeam.ReportProgress("worker-1", "", "Completed initial design")

	// 获取进度日志
	logs := myTeam.GetProgressLog(10)
	for _, log := range logs {
		if log.TaskID == "" {
			fmt.Printf("[%s] %s\n", log.AgentName, log.Message)
		}
	}

	// Output:
	// [Progress Worker] 加入团队，当前成员数: 1
	// [Progress Worker] Started analyzing requirements
	// [Progress Worker] Completed initial design
}

// Example_report 演示报告管理
func Example_report() {
	myTeam := team.New(team.WithName("report-demo"))

	// 添加成员
	analyst := &mockMember{id: "analyst-1", name: "Data Analyst"}
	_ = myTeam.AddMember(analyst)

	// 添加报告
	reportID := myTeam.AddReport("analyst-1", "Weekly Summary", "This week we completed...")

	// 获取报告
	report, _ := myTeam.GetReport(reportID)
	fmt.Printf("Report: %s by %s\n", report.Title, report.AgentName)

	// 获取所有报告
	reports := myTeam.GetReports()
	fmt.Printf("Total reports: %d\n", len(reports))

	// Output:
	// Report: Weekly Summary by Data Analyst
	// Total reports: 1
}

// Example_hierarchy 演示层级管理
func Example_hierarchy() {
	myTeam := team.New(team.WithName("hierarchy-demo"))

	// 创建层级结构
	root := &mockMember{id: "root", name: "Root"}
	child1 := &mockMember{id: "child-1", name: "Child 1", parentID: "root"}
	child2 := &mockMember{id: "child-2", name: "Child 2", parentID: "root"}
	grandchild := &mockMember{id: "grandchild-1", name: "Grandchild 1", parentID: "child-1"}

	_ = myTeam.AddMember(root)
	_ = myTeam.AddMember(child1)
	_ = myTeam.AddMember(child2)
	_ = myTeam.AddMember(grandchild)

	// 获取直接子成员
	children := myTeam.ListChildMembers("root")
	fmt.Printf("Root has %d direct children\n", len(children))

	// 获取所有后代
	descendants := myTeam.ListDescendantMembers("root")
	fmt.Printf("Root has %d total descendants\n", len(descendants))

	// 获取血统
	lineage := myTeam.GetMemberLineage("grandchild-1")
	fmt.Printf("Grandchild lineage: %v\n", lineage)

	// Output:
	// Root has 2 direct children
	// Root has 3 total descendants
	// Grandchild lineage: [root child-1 grandchild-1]
}

// Example_summary 演示状态摘要
func Example_summary() {
	myTeam := team.New(
		team.WithName("summary-demo"),
		team.WithDescription("Demo team"),
	)

	// 添加成员和任务
	worker := &mockMember{id: "worker", name: "Worker"}
	_ = myTeam.AddMember(worker)

	myTeam.AddTask(team.Task{Title: "Task 1", Status: team.TaskStatusCompleted})
	myTeam.AddTask(team.Task{Title: "Task 2", Status: team.TaskStatusInProgress})
	myTeam.AddTask(team.Task{Title: "Task 3", Status: team.TaskStatusPending})

	// 获取摘要
	summary := myTeam.GetSummary()
	fmt.Printf("Team: %s\n", summary.Name)
	fmt.Printf("Members: %d\n", summary.MemberCount)
	fmt.Printf("Tasks: %d total, %d completed, %d in progress, %d pending\n",
		summary.TotalTasks, summary.CompletedTasks, summary.InProgressTasks, summary.PendingTasks)

	// Output:
	// Team: summary-demo
	// Members: 1
	// Tasks: 3 total, 1 completed, 1 in progress, 1 pending
}

// Example_actorBasic 演示 Actor-based TeamActor 的基本用法
//
// TeamActor 通过 Actor 模型实现 lock-free 并发。
func Example_actorBasic() {
	// 创建 Actor 系统
	sys := actor.NewSystem("team-example")
	defer sys.Shutdown()

	// 创建 TeamActor
	teamActor := team.NewTeamActor("actor-team",
		team.WithDescription("Actor-based team demo"),
	)
	teamPID := sys.Spawn(teamActor, "team")

	// 等待启动
	time.Sleep(10 * time.Millisecond)

	// 添加成员（通过消息）
	worker := &mockMember{id: "actor-worker", name: "Actor Worker"}
	err := team.DoAddMember(teamPID, worker, time.Second)
	if err != nil {
		fmt.Println("AddMember error:", err)
		return
	}

	// 获取 Leader
	leader, err := team.DoGetLeader(teamPID, time.Second)
	if err != nil {
		fmt.Println("GetLeader error:", err)
		return
	}
	fmt.Printf("Leader: %s\n", leader.Name())

	// 添加任务
	taskID, err := team.DoAddTask(teamPID, team.Task{
		Title:       "Actor Task",
		Description: "Task managed via Actor messages",
	}, time.Second)
	if err != nil {
		fmt.Println("AddTask error:", err)
		return
	}
	fmt.Printf("Task ID: %s\n", taskID)

	// 分配任务
	err = team.DoAssignTask(teamPID, taskID, "actor-worker", time.Second)
	if err != nil {
		fmt.Println("AssignTask error:", err)
		return
	}

	// 报告进度（fire-and-forget）
	team.DoReportProgress(teamPID, "actor-worker", taskID, "Working on it")

	// 等待进度被记录
	time.Sleep(10 * time.Millisecond)

	// 获取摘要
	summary, err := team.DoGetSummary(teamPID, time.Second)
	if err != nil {
		fmt.Println("GetSummary error:", err)
		return
	}
	fmt.Printf("Members: %d, Tasks: %d\n", summary.MemberCount, summary.TotalTasks)

	// Output:
	// Leader: Actor Worker
	// Task ID: task-1
	// Members: 1, Tasks: 1
}
