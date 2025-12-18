package team

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lwmacct/251215-go-pkg-agent/pkg/agent"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/provider/localmock"
)

// testMember 测试用的 Mock 成员
type testMember struct {
	id       string
	name     string
	parentID string
	closed   bool
}

func (m *testMember) ID() string       { return m.id }
func (m *testMember) Name() string     { return m.name }
func (m *testMember) ParentID() string { return m.parentID }
func (m *testMember) Close() error {
	m.closed = true
	return nil
}

func newTestMember(id, name string) *testMember {
	return &testMember{id: id, name: name}
}

func TestNew(t *testing.T) {
	team := New()
	assert.NotNil(t, team)
	assert.Equal(t, "default-team", team.Info().Name)
}

func TestNew_WithOptions(t *testing.T) {
	team := New(
		WithName("test-team"),
		WithDescription("Test description"),
	)
	assert.Equal(t, "test-team", team.Info().Name)
	assert.Equal(t, "Test description", team.Info().Description)
}

func TestBaseTeam_AddMember(t *testing.T) {
	team := New()

	m := newTestMember("m1", "Member 1")
	err := team.AddMember(m)
	require.NoError(t, err)

	// First member becomes leader
	assert.Equal(t, m, team.GetLeader())
	assert.True(t, team.IsLeader("m1"))

	// Add second member
	m2 := newTestMember("m2", "Member 2")
	err = team.AddMember(m2)
	require.NoError(t, err)

	// First member still leader
	assert.Equal(t, m, team.GetLeader())
	assert.False(t, team.IsLeader("m2"))
}

func TestBaseTeam_AddMember_Errors(t *testing.T) {
	team := New()

	err := team.AddMember(nil)
	assert.Error(t, err)

	m := &testMember{id: "", name: "No ID"}
	err = team.AddMember(m)
	assert.Error(t, err)
}

func TestBaseTeam_RemoveMember(t *testing.T) {
	team := New()

	m1 := newTestMember("m1", "Member 1")
	m2 := newTestMember("m2", "Member 2")
	_ = team.AddMember(m1)
	_ = team.AddMember(m2)

	team.RemoveMember("m1")

	_, found := team.GetMember("m1")
	assert.False(t, found)

	// m2 should be new leader
	assert.Equal(t, m2, team.GetLeader())
}

func TestBaseTeam_CloseMember(t *testing.T) {
	team := New()

	m := newTestMember("m1", "Member 1")
	_ = team.AddMember(m)

	err := team.CloseMember("m1")
	require.NoError(t, err)

	assert.True(t, m.closed)
	_, found := team.GetMember("m1")
	assert.False(t, found)
}

func TestBaseTeam_CloseMember_NotFound(t *testing.T) {
	team := New()
	err := team.CloseMember("nonexistent")
	assert.Error(t, err)
}

func TestBaseTeam_GetMember(t *testing.T) {
	team := New()

	m := newTestMember("m1", "Member 1")
	_ = team.AddMember(m)

	got, found := team.GetMember("m1")
	assert.True(t, found)
	assert.Equal(t, m, got)

	_, found = team.GetMember("nonexistent")
	assert.False(t, found)
}

func TestBaseTeam_ListMembers(t *testing.T) {
	team := New()

	_ = team.AddMember(newTestMember("m1", "Member 1"))
	_ = team.AddMember(newTestMember("m2", "Member 2"))
	_ = team.AddMember(newTestMember("m3", "Member 3"))

	members := team.ListMembers()
	assert.Len(t, members, 3)
}

func TestBaseTeam_Hierarchy(t *testing.T) {
	team := New()

	root := newTestMember("root", "Root")
	child1 := &testMember{id: "c1", name: "Child 1", parentID: "root"}
	child2 := &testMember{id: "c2", name: "Child 2", parentID: "root"}
	grandchild := &testMember{id: "gc1", name: "Grandchild 1", parentID: "c1"}

	_ = team.AddMember(root)
	_ = team.AddMember(child1)
	_ = team.AddMember(child2)
	_ = team.AddMember(grandchild)

	// Test ListChildMembers
	children := team.ListChildMembers("root")
	assert.Len(t, children, 2)

	// Test ListDescendantMembers
	descendants := team.ListDescendantMembers("root")
	assert.Len(t, descendants, 3)

	// Test GetMemberLineage
	lineage := team.GetMemberLineage("gc1")
	assert.Equal(t, []string{"root", "c1", "gc1"}, lineage)
}

func TestBaseTeam_TaskManagement(t *testing.T) {
	team := New()

	m := newTestMember("m1", "Worker")
	_ = team.AddMember(m)

	// Add task
	taskID := team.AddTask(Task{
		Title:       "Test Task",
		Description: "Test description",
	})
	assert.NotEmpty(t, taskID)

	// Get task
	task, found := team.GetTask(taskID)
	require.True(t, found)
	assert.Equal(t, "Test Task", task.Title)
	assert.Equal(t, TaskStatusPending, task.Status)

	// Assign task
	err := team.AssignTask(taskID, "m1")
	require.NoError(t, err)

	task, _ = team.GetTask(taskID)
	assert.Equal(t, "m1", task.AssignedTo)

	// Update status
	err = team.UpdateTaskStatus(taskID, TaskStatusInProgress, "m1")
	require.NoError(t, err)

	task, _ = team.GetTask(taskID)
	assert.Equal(t, TaskStatusInProgress, task.Status)

	// Complete task
	err = team.UpdateTaskStatus(taskID, TaskStatusCompleted, "m1")
	require.NoError(t, err)

	task, _ = team.GetTask(taskID)
	assert.Equal(t, TaskStatusCompleted, task.Status)
	assert.NotNil(t, task.CompletedAt)
}

func TestBaseTeam_TaskErrors(t *testing.T) {
	team := New()

	// Assign to nonexistent member
	taskID := team.AddTask(Task{Title: "Task"})
	err := team.AssignTask(taskID, "nonexistent")
	assert.Error(t, err)

	// Update nonexistent task
	err = team.UpdateTaskStatus("nonexistent", TaskStatusCompleted, "")
	assert.Error(t, err)

	// Assign nonexistent task
	m := newTestMember("m1", "Worker")
	_ = team.AddMember(m)
	err = team.AssignTask("nonexistent", "m1")
	assert.Error(t, err)
}

func TestBaseTeam_GetTaskBoard(t *testing.T) {
	team := New()

	team.AddTask(Task{Title: "Task 1"})
	team.AddTask(Task{Title: "Task 2"})

	board := team.GetTaskBoard()
	assert.Len(t, board.Tasks, 2)
}

func TestBaseTeam_GetMemberTasks(t *testing.T) {
	team := New()

	m := newTestMember("m1", "Worker")
	_ = team.AddMember(m)

	t1 := team.AddTask(Task{Title: "Task 1"})
	team.AddTask(Task{Title: "Task 2"})
	t3 := team.AddTask(Task{Title: "Task 3"})

	_ = team.AssignTask(t1, "m1")
	_ = team.AssignTask(t3, "m1")

	tasks := team.GetMemberTasks("m1")
	assert.Len(t, tasks, 2)
}

func TestBaseTeam_Progress(t *testing.T) {
	team := New()

	m := newTestMember("m1", "Worker")
	_ = team.AddMember(m)

	team.ReportProgress("m1", "", "Started work")
	team.ReportProgress("m1", "task-1", "Making progress")

	logs := team.GetProgressLog(10)
	assert.Len(t, logs, 3) // including join message
}

func TestBaseTeam_GetProgressLog_Limit(t *testing.T) {
	team := New()

	for range 10 {
		team.ReportProgress("", "", "Progress")
	}

	logs := team.GetProgressLog(3)
	assert.Len(t, logs, 3)

	logs = team.GetProgressLog(0)
	assert.Len(t, logs, 10)

	logs = team.GetProgressLog(100)
	assert.Len(t, logs, 10)
}

func TestBaseTeam_Reports(t *testing.T) {
	team := New()

	m := newTestMember("m1", "Reporter")
	_ = team.AddMember(m)

	reportID := team.AddReport("m1", "Test Report", "Report content")
	assert.NotEmpty(t, reportID)

	report, found := team.GetReport(reportID)
	require.True(t, found)
	assert.Equal(t, "Test Report", report.Title)
	assert.Equal(t, "Report content", report.Content)
	assert.Equal(t, "m1", report.CreatedBy)
	assert.Equal(t, "Reporter", report.AgentName)

	reports := team.GetReports()
	assert.Len(t, reports, 1)
}

func TestBaseTeam_GetReport_NotFound(t *testing.T) {
	team := New()
	_, found := team.GetReport("nonexistent")
	assert.False(t, found)
}

func TestBaseTeam_GetSummary(t *testing.T) {
	team := New(
		WithName("test-team"),
		WithDescription("Test description"),
	)

	m := newTestMember("m1", "Worker")
	_ = team.AddMember(m)

	team.AddTask(Task{Title: "Task 1", Status: TaskStatusPending})
	team.AddTask(Task{Title: "Task 2", Status: TaskStatusInProgress})
	team.AddTask(Task{Title: "Task 3", Status: TaskStatusCompleted})
	team.AddTask(Task{Title: "Task 4", Status: TaskStatusBlocked})

	summary := team.GetSummary()

	assert.Equal(t, "test-team", summary.Name)
	assert.Equal(t, "Test description", summary.Description)
	assert.Equal(t, 1, summary.MemberCount)
	assert.Equal(t, "m1", summary.LeaderID)
	assert.Equal(t, "Worker", summary.LeaderName)
	assert.Equal(t, 4, summary.TotalTasks)
	assert.Equal(t, 1, summary.PendingTasks)
	assert.Equal(t, 1, summary.InProgressTasks)
	assert.Equal(t, 1, summary.CompletedTasks)
	assert.Equal(t, 1, summary.BlockedTasks)
}

func TestBaseTeam_Shutdown(t *testing.T) {
	team := New()

	m1 := newTestMember("m1", "Member 1")
	m2 := newTestMember("m2", "Member 2")
	_ = team.AddMember(m1)
	_ = team.AddMember(m2)

	team.Shutdown()

	assert.True(t, m1.closed)
	assert.True(t, m2.closed)
	assert.Nil(t, team.GetLeader())
	assert.Empty(t, team.ListMembers())
}

func TestBaseTeam_Concurrency(t *testing.T) {
	team := New()

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent adds
	wg.Add(numGoroutines)
	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			m := newTestMember(
				string(rune('a'+id%26))+string(rune('0'+id)),
				"Member",
			)
			_ = team.AddMember(m)
		}(i)
	}
	wg.Wait()

	// Concurrent reads
	wg.Add(numGoroutines)
	for range numGoroutines {
		go func() {
			defer wg.Done()
			_ = team.ListMembers()
			_ = team.GetSummary()
			_ = team.GetProgressLog(10)
		}()
	}
	wg.Wait()
}

// ═══════════════════════════════════════════════════════════════════════════
// ForkMember 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestBaseTeam_ForkMember(t *testing.T) {
	t.Run("should_fork_agent_successfully", func(t *testing.T) {
		team := New(WithName("test-team"))
		mockProvider := localmock.New(localmock.WithResponse("OK"))

		// 创建源 Agent
		srcAgent, err := agent.New().
			ID("base-agent").
			Name("Base Agent").
			Model("gpt-4").
			System("Original prompt").
			Provider(mockProvider).
			Build()
		require.NoError(t, err)
		defer func() { _ = srcAgent.Close() }()

		err = team.AddMember(srcAgent)
		require.NoError(t, err)

		// 克隆 Agent
		err = team.ForkMember("base-agent", "worker-1",
			agent.WithName("Worker 1"),
			agent.WithPrompt("Specialized task"),
			agent.WithProvider(mockProvider),
		)
		require.NoError(t, err)

		// 验证新成员存在
		worker, found := team.GetMember("worker-1")
		assert.True(t, found)
		assert.Equal(t, "worker-1", worker.ID())
		assert.Equal(t, "Worker 1", worker.Name())

		// 验证配置被正确克隆和覆盖
		workerAgent := worker.(*agent.Agent)
		workerCfg := workerAgent.Config()
		assert.Equal(t, "worker-1", workerCfg.ID)
		assert.Equal(t, "Worker 1", workerCfg.Name)
		assert.Equal(t, "Specialized task", workerCfg.SystemPrompt)
		assert.Equal(t, "gpt-4", workerCfg.LLM.Model) // 继承源 Agent 的 Model

		// 验证是不同的实例
		assert.NotEqual(t, srcAgent, workerAgent)

		_ = workerAgent.Close()
	})

	t.Run("should_fail_if_source_not_found", func(t *testing.T) {
		team := New()

		err := team.ForkMember("nonexistent", "new-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "source member not found")
	})

	t.Run("should_fail_if_new_id_exists", func(t *testing.T) {
		team := New()
		mockProvider := localmock.New(localmock.WithResponse("OK"))

		// 添加源 Agent
		srcAgent, err := agent.New().
			ID("base-agent").
			Model("gpt-4").
			Provider(mockProvider).
			Build()
		require.NoError(t, err)
		defer func() { _ = srcAgent.Close() }()

		err = team.AddMember(srcAgent)
		require.NoError(t, err)

		// 添加另一个成员占用目标 ID
		existingAgent, err := agent.New().
			ID("worker-1").
			Model("gpt-4").
			Provider(mockProvider).
			Build()
		require.NoError(t, err)
		defer func() { _ = existingAgent.Close() }()

		err = team.AddMember(existingAgent)
		require.NoError(t, err)

		// 尝试用已存在的 ID 克隆
		err = team.ForkMember("base-agent", "worker-1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("should_fail_if_source_is_not_agent", func(t *testing.T) {
		team := New()

		// 添加非 Agent 成员
		testMem := newTestMember("test-member", "Test")
		err := team.AddMember(testMem)
		require.NoError(t, err)

		// 尝试克隆非 Agent 成员
		err = team.ForkMember("test-member", "new-member")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "is not an *agent.Agent")
	})

	t.Run("should_fail_with_invalid_option_type", func(t *testing.T) {
		team := New()
		mockProvider := localmock.New(localmock.WithResponse("OK"))

		// 添加源 Agent
		srcAgent, err := agent.New().
			ID("base-agent").
			Model("gpt-4").
			Provider(mockProvider).
			Build()
		require.NoError(t, err)
		defer func() { _ = srcAgent.Close() }()

		err = team.AddMember(srcAgent)
		require.NoError(t, err)

		// 传入无效的选项类型
		err = team.ForkMember("base-agent", "worker-1", "invalid-option")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid option type")
	})

	t.Run("should_support_multiple_forks", func(t *testing.T) {
		team := New()
		mockProvider := localmock.New(localmock.WithResponse("OK"))

		// 创建源 Agent
		baseAgent, err := agent.New().
			ID("base-agent").
			Name("Base Agent").
			Model("gpt-4").
			System("Base prompt").
			Provider(mockProvider).
			Build()
		require.NoError(t, err)
		defer func() { _ = baseAgent.Close() }()

		err = team.AddMember(baseAgent)
		require.NoError(t, err)

		// 克隆多个 Worker
		for i := 1; i <= 3; i++ {
			workerID := "worker-" + string(rune('0'+i))
			workerName := "Worker " + string(rune('0'+i))

			err := team.ForkMember("base-agent", workerID,
				agent.WithName(workerName),
				agent.WithPrompt("Task "+string(rune('0'+i))),
				agent.WithProvider(mockProvider),
			)
			require.NoError(t, err, "Fork %d failed", i)
		}

		// 验证所有成员都存在
		members := team.ListMembers()
		assert.Len(t, members, 4) // base + 3 workers

		// 清理
		for _, m := range members {
			if ag, ok := m.(*agent.Agent); ok && ag != baseAgent {
				_ = ag.Close()
			}
		}
	})
}

func TestBaseTeam_ForkMember_Independence(t *testing.T) {
	t.Run("forked_agents_should_be_independent", func(t *testing.T) {
		team := New()
		mockProvider := localmock.New(localmock.WithResponse("OK"))

		// 创建源 Agent
		baseAgent, err := agent.New().
			ID("base-agent").
			Name("Base").
			Model("gpt-4").
			MaxTokens(1000).
			Provider(mockProvider).
			Build()
		require.NoError(t, err)
		defer func() { _ = baseAgent.Close() }()

		err = team.AddMember(baseAgent)
		require.NoError(t, err)

		// 克隆
		err = team.ForkMember("base-agent", "worker-1", agent.WithProvider(mockProvider))
		require.NoError(t, err)

		worker, _ := team.GetMember("worker-1")
		workerAgent := worker.(*agent.Agent)
		defer func() { _ = workerAgent.Close() }()

		// 验证是不同的实例
		assert.NotEqual(t, baseAgent, workerAgent)

		// 获取配置快照
		baseCfg := baseAgent.Config()
		workerCfg := workerAgent.Config()

		// 验证配置独立（值相同但是不同的对象）
		assert.Equal(t, baseCfg.LLM.Model, workerCfg.LLM.Model)
		assert.Equal(t, baseCfg.MaxTokens, workerCfg.MaxTokens)
		assert.NotEqual(t, baseCfg.ID, workerCfg.ID) // ID 不同
	})
}

func TestBaseTeam_ForkMember_Concurrent(t *testing.T) {
	t.Run("concurrent_forks_should_be_safe", func(t *testing.T) {
		team := New()
		mockProvider := localmock.New(localmock.WithResponse("OK"))

		// 创建源 Agent
		baseAgent, err := agent.New().
			ID("base-agent").
			Name("Base Agent").
			Model("gpt-4").
			Provider(mockProvider).
			Build()
		require.NoError(t, err)
		defer func() { _ = baseAgent.Close() }()

		err = team.AddMember(baseAgent)
		require.NoError(t, err)

		// 并发克隆
		const numWorkers = 20
		var wg sync.WaitGroup
		wg.Add(numWorkers)
		errors := make(chan error, numWorkers)

		for i := range numWorkers {
			go func(idx int) {
				defer wg.Done()
				workerID := "worker-" + string(rune('a'+idx%26)) + string(rune('0'+idx/26))
				workerName := "Worker " + string(rune('0'+idx))

				err := team.ForkMember("base-agent", workerID,
					agent.WithName(workerName),
					agent.WithProvider(mockProvider),
				)
				if err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// 验证没有错误
		for err := range errors {
			t.Errorf("Fork failed: %v", err)
		}

		// 验证所有 Worker 都被创建
		members := team.ListMembers()
		assert.GreaterOrEqual(t, len(members), numWorkers)

		// 清理
		for _, m := range members {
			if ag, ok := m.(*agent.Agent); ok && ag != baseAgent {
				_ = ag.Close()
			}
		}
	})
}
