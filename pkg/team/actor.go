package team

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/lwmacct/251215-go-pkg-agent/pkg/agent"
	"github.com/lwmacct/251215-go-pkg-team/pkg/actor"
)

// TeamActor Actor-based Team 实现
//
// TeamActor 完全消除了 RWMutex，通过消息邮箱实现并发安全。
// 所有操作通过消息传递，Actor 串行处理保证状态一致性。
//
// Thread Safety: TeamActor 天然并发安全，无需额外锁保护。
type TeamActor struct {
	// 团队信息
	name        string
	description string
	createdAt   time.Time

	// 成员管理
	leader  Member
	members map[string]Member

	// 任务管理
	taskBoard *TaskBoard

	// 进度与报告
	progressLog []Progress
	reports     []Report

	// 日志
	logger *slog.Logger

	// 统计
	stats *actor.StatsCollector
}

// NewTeamActor 创建 TeamActor
func NewTeamActor(name string, opts ...Option) *TeamActor {
	ta := &TeamActor{
		name:        name,
		createdAt:   time.Now(),
		members:     make(map[string]Member),
		taskBoard:   NewTaskBoard(),
		progressLog: make([]Progress, 0),
		reports:     make([]Report, 0),
		logger:      slog.Default(),
		stats:       actor.NewStatsCollector(),
	}

	// 应用选项（复用 BaseTeam 的 Option）
	bt := &BaseTeam{name: name}
	for _, opt := range opts {
		opt(bt)
	}
	ta.name = bt.name
	ta.description = bt.description

	return ta
}

// Receive 处理接收到的消息（Actor 核心方法）
func (t *TeamActor) Receive(ctx *actor.Context, msg actor.Message) {
	t.stats.RecordReceived()
	startTime := time.Now()

	defer func() {
		t.stats.RecordHandled(time.Since(startTime))
	}()

	switch m := msg.(type) {
	// ─────────────────────────────────────────────────────────────────────
	// 系统消息
	// ─────────────────────────────────────────────────────────────────────

	case *actor.Started:
		t.logger.Debug("TeamActor started", "team", t.name)

	case *actor.Stopping:
		t.logger.Debug("TeamActor stopping", "team", t.name)
		t.handleShutdown()

	case *actor.Stopped:
		t.logger.Debug("TeamActor stopped", "team", t.name)

	// ─────────────────────────────────────────────────────────────────────
	// 成员管理消息
	// ─────────────────────────────────────────────────────────────────────

	case *AddMemberMsg:
		err := t.handleAddMember(m.Member)
		actor.TrySend(m.ReplyChan, err)

	case *RemoveMemberMsg:
		t.handleRemoveMember(m.MemberID)

	case *CloseMemberMsg:
		err := t.handleCloseMember(m.MemberID)
		actor.TrySend(m.ReplyChan, err)

	case *GetMemberMsg:
		member, found := t.members[m.MemberID]
		actor.TrySend(m.ReplyChan, &GetMemberResult{Member: member, Found: found})

	case *ListMembersMsg:
		members := t.listMembersInternal()
		actor.TrySend(m.ReplyChan, members)

	case *ListChildMembersMsg:
		children := t.listChildMembersInternal(m.ParentID)
		actor.TrySend(m.ReplyChan, children)

	case *ListDescendantMembersMsg:
		descendants := t.listDescendantMembersInternal(m.ParentID)
		actor.TrySend(m.ReplyChan, descendants)

	case *GetMemberLineageMsg:
		lineage := t.getMemberLineageInternal(m.MemberID)
		actor.TrySend(m.ReplyChan, lineage)

	case *ForkMemberMsg:
		err := t.handleForkMember(m.SourceID, m.NewID, m.Options)
		actor.TrySend(m.ReplyChan, err)

	// ─────────────────────────────────────────────────────────────────────
	// Leader 管理消息
	// ─────────────────────────────────────────────────────────────────────

	case *GetLeaderMsg:
		actor.TrySend(m.ReplyChan, t.leader)

	case *IsLeaderMsg:
		isLeader := t.leader != nil && t.leader.ID() == m.MemberID
		actor.TrySend(m.ReplyChan, isLeader)

	// ─────────────────────────────────────────────────────────────────────
	// 任务管理消息
	// ─────────────────────────────────────────────────────────────────────

	case *AddTaskMsg:
		taskID := t.handleAddTask(m.Task)
		actor.TrySend(m.ReplyChan, taskID)

	case *UpdateTaskStatusMsg:
		err := t.handleUpdateTaskStatus(m.TaskID, m.Status, m.MemberID)
		actor.TrySend(m.ReplyChan, err)

	case *AssignTaskMsg:
		err := t.handleAssignTask(m.TaskID, m.MemberID)
		actor.TrySend(m.ReplyChan, err)

	case *GetTaskMsg:
		task := t.taskBoard.GetByID(m.TaskID)
		found := task != nil
		actor.TrySend(m.ReplyChan, &GetTaskResult{Task: task, Found: found})

	case *GetTaskBoardMsg:
		board := t.getTaskBoardInternal()
		actor.TrySend(m.ReplyChan, board)

	case *GetMemberTasksMsg:
		tasks := t.taskBoard.GetByAssignee(m.MemberID)
		actor.TrySend(m.ReplyChan, tasks)

	// ─────────────────────────────────────────────────────────────────────
	// 进度追踪消息
	// ─────────────────────────────────────────────────────────────────────

	case *ReportProgressMsg:
		t.handleReportProgress(m.MemberID, m.TaskID, m.Message)

	case *GetProgressLogMsg:
		log := t.getProgressLogInternal(m.Limit)
		actor.TrySend(m.ReplyChan, log)

	// ─────────────────────────────────────────────────────────────────────
	// 报告管理消息
	// ─────────────────────────────────────────────────────────────────────

	case *AddReportMsg:
		reportID := t.handleAddReport(m.MemberID, m.Title, m.Content)
		actor.TrySend(m.ReplyChan, reportID)

	case *GetReportsMsg:
		reports := t.getReportsInternal()
		actor.TrySend(m.ReplyChan, reports)

	case *GetReportMsg:
		report, found := t.getReportInternal(m.ReportID)
		actor.TrySend(m.ReplyChan, &GetReportResult{Report: report, Found: found})

	// ─────────────────────────────────────────────────────────────────────
	// 团队信息与生命周期消息
	// ─────────────────────────────────────────────────────────────────────

	case *GetInfoMsg:
		info := TeamInfo{Name: t.name, Description: t.description}
		actor.TrySend(m.ReplyChan, info)

	case *GetSummaryMsg:
		summary := t.getSummaryInternal()
		actor.TrySend(m.ReplyChan, summary)

	case *ShutdownMsg:
		t.logger.Info("TeamActor shutdown requested", "team", t.name, "reason", m.Reason)
		t.handleShutdown()
		ctx.StopSelf()

	default:
		t.logger.Warn("TeamActor received unknown message", "type", msg.Kind())
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 内部处理方法（无锁，由 Actor 串行调用保证安全）
// ═══════════════════════════════════════════════════════════════════════════

// handleAddMember 添加成员
func (t *TeamActor) handleAddMember(m Member) error {
	if m == nil {
		return fmt.Errorf("member cannot be nil")
	}
	if m.ID() == "" {
		return fmt.Errorf("member ID cannot be empty")
	}

	// 检查 ID 是否已存在
	if _, exists := t.members[m.ID()]; exists {
		return fmt.Errorf("member with ID %s already exists", m.ID())
	}

	// 循环依赖检测
	if m.ParentID() != "" {
		if err := detectCycle(t.members, m.ID(), m.ParentID()); err != nil {
			return fmt.Errorf("cycle validation failed: %w", err)
		}
	}

	// 深度限制检查
	if m.ParentID() != "" {
		if err := checkDepthLimit(t.members, m.ParentID(), MaxMemberDepth); err != nil {
			return fmt.Errorf("depth validation failed: %w", err)
		}
	}

	// 第一个加入的成为 Leader
	if t.leader == nil {
		t.leader = m
	}

	t.members[m.ID()] = m

	// 记录进度
	t.progressLog = append(t.progressLog, Progress{
		AgentID:   m.ID(),
		AgentName: m.Name(),
		Message:   fmt.Sprintf("加入团队，当前成员数: %d", len(t.members)),
		Timestamp: time.Now(),
	})

	return nil
}

// handleRemoveMember 移除成员
func (t *TeamActor) handleRemoveMember(id string) {
	m, ok := t.members[id]
	if !ok {
		return
	}

	delete(t.members, id)

	// 如果移除的是 Leader，选举新 Leader
	if t.leader != nil && t.leader.ID() == id {
		t.leader = nil
		for _, member := range t.members {
			t.leader = member
			break
		}
	}

	// 记录进度
	t.progressLog = append(t.progressLog, Progress{
		AgentID:   id,
		AgentName: m.Name(),
		Message:   "离开团队",
		Timestamp: time.Now(),
	})
}

// handleCloseMember 关闭并移除成员
func (t *TeamActor) handleCloseMember(id string) error {
	m, ok := t.members[id]
	if !ok {
		return fmt.Errorf("member not found: %s", id)
	}

	delete(t.members, id)

	// 如果关闭的是 Leader，选举新 Leader
	if t.leader != nil && t.leader.ID() == id {
		t.leader = nil
		for _, member := range t.members {
			t.leader = member
			break
		}
	}

	// 如果实现了 Closeable，调用 Close
	if closer, ok := m.(Closeable); ok {
		return closer.Close()
	}

	return nil
}

// handleForkMember 克隆成员
func (t *TeamActor) handleForkMember(srcID, newID string, opts []any) error {
	// 检查新 ID 是否已存在
	if _, exists := t.members[newID]; exists {
		return fmt.Errorf("member with ID %s already exists", newID)
	}

	// 查找源成员
	srcMember, exists := t.members[srcID]
	if !exists {
		return fmt.Errorf("source member not found: %s", srcID)
	}

	// 类型断言：检查是否是 *agent.Agent
	srcAgent, ok := srcMember.(*agent.Agent)
	if !ok {
		return fmt.Errorf("member %s is not an *agent.Agent (got %T)", srcID, srcMember)
	}

	// 转换 opts 为 agent.Option
	agentOpts := make([]agent.Option, 0, len(opts)+1)
	agentOpts = append(agentOpts, agent.WithID(newID))

	for _, opt := range opts {
		if agentOpt, ok := opt.(agent.Option); ok {
			agentOpts = append(agentOpts, agentOpt)
		} else {
			return fmt.Errorf("invalid option type: %T (expected agent.Option)", opt)
		}
	}

	// 克隆 Agent
	newAgent, err := agent.CloneAgent(srcAgent, agentOpts...)
	if err != nil {
		return fmt.Errorf("clone agent failed: %w", err)
	}

	// 添加到团队
	return t.handleAddMember(newAgent)
}

// listMembersInternal 列出所有成员
func (t *TeamActor) listMembersInternal() []Member {
	members := make([]Member, 0, len(t.members))
	for _, m := range t.members {
		members = append(members, m)
	}
	return members
}

// listChildMembersInternal 列出直接子成员
func (t *TeamActor) listChildMembersInternal(parentID string) []Member {
	children := make([]Member, 0)
	for _, m := range t.members {
		if m.ParentID() == parentID {
			children = append(children, m)
		}
	}
	return children
}

// listDescendantMembersInternal 列出所有后代成员
func (t *TeamActor) listDescendantMembersInternal(parentID string) []Member {
	var descendants []Member
	var collect func(pid string)
	collect = func(pid string) {
		for _, m := range t.members {
			if m.ParentID() == pid {
				descendants = append(descendants, m)
				collect(m.ID())
			}
		}
	}
	collect(parentID)
	return descendants
}

// getMemberLineageInternal 获取成员血统
func (t *TeamActor) getMemberLineageInternal(id string) []string {
	var lineage []string
	currentID := id

	for currentID != "" {
		lineage = append([]string{currentID}, lineage...)
		if m, ok := t.members[currentID]; ok {
			currentID = m.ParentID()
		} else {
			break
		}
	}
	return lineage
}

// ═══════════════════════════════════════════════════════════════════════════
// 任务管理内部方法
// ═══════════════════════════════════════════════════════════════════════════

// handleAddTask 添加任务
func (t *TeamActor) handleAddTask(task Task) string {
	if task.ID == "" {
		task.ID = fmt.Sprintf("task-%d", len(t.taskBoard.Tasks)+1)
	}
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	if task.Status == "" {
		task.Status = TaskStatusPending
	}

	t.taskBoard.Tasks = append(t.taskBoard.Tasks, task)
	return task.ID
}

// handleUpdateTaskStatus 更新任务状态
func (t *TeamActor) handleUpdateTaskStatus(taskID string, status TaskStatus, memberID string) error {
	for i := range t.taskBoard.Tasks {
		if t.taskBoard.Tasks[i].ID == taskID {
			t.taskBoard.Tasks[i].Status = status
			t.taskBoard.Tasks[i].UpdatedAt = time.Now()
			if status == TaskStatusCompleted {
				now := time.Now()
				t.taskBoard.Tasks[i].CompletedAt = &now
			}

			// 记录进度
			memberName := ""
			if m, ok := t.members[memberID]; ok {
				memberName = m.Name()
			}
			t.progressLog = append(t.progressLog, Progress{
				AgentID:   memberID,
				AgentName: memberName,
				TaskID:    taskID,
				Message:   fmt.Sprintf("任务状态更新: %s", status),
				Timestamp: time.Now(),
			})
			return nil
		}
	}
	return fmt.Errorf("task not found: %s", taskID)
}

// handleAssignTask 分配任务
func (t *TeamActor) handleAssignTask(taskID, memberID string) error {
	// 验证成员存在
	if _, ok := t.members[memberID]; !ok {
		return fmt.Errorf("member not found: %s", memberID)
	}

	for i := range t.taskBoard.Tasks {
		if t.taskBoard.Tasks[i].ID == taskID {
			t.taskBoard.Tasks[i].AssignedTo = memberID
			t.taskBoard.Tasks[i].UpdatedAt = time.Now()

			memberName := ""
			if m, ok := t.members[memberID]; ok {
				memberName = m.Name()
			}
			t.progressLog = append(t.progressLog, Progress{
				AgentID:   memberID,
				AgentName: memberName,
				TaskID:    taskID,
				Message:   "任务已分配",
				Timestamp: time.Now(),
			})
			return nil
		}
	}
	return fmt.Errorf("task not found: %s", taskID)
}

// getTaskBoardInternal 获取任务看板副本
func (t *TeamActor) getTaskBoardInternal() TaskBoard {
	board := TaskBoard{
		Tasks: make([]Task, len(t.taskBoard.Tasks)),
	}
	copy(board.Tasks, t.taskBoard.Tasks)
	return board
}

// ═══════════════════════════════════════════════════════════════════════════
// 进度与报告内部方法
// ═══════════════════════════════════════════════════════════════════════════

// handleReportProgress 报告进度
func (t *TeamActor) handleReportProgress(memberID, taskID, message string) {
	memberName := ""
	if m, ok := t.members[memberID]; ok {
		memberName = m.Name()
	}

	t.progressLog = append(t.progressLog, Progress{
		AgentID:   memberID,
		AgentName: memberName,
		TaskID:    taskID,
		Message:   message,
		Timestamp: time.Now(),
	})
}

// getProgressLogInternal 获取进度日志
func (t *TeamActor) getProgressLogInternal(limit int) []Progress {
	if limit <= 0 || limit > len(t.progressLog) {
		limit = len(t.progressLog)
	}

	start := max(len(t.progressLog)-limit, 0)
	result := make([]Progress, limit)
	copy(result, t.progressLog[start:])
	return result
}

// handleAddReport 添加报告
func (t *TeamActor) handleAddReport(memberID, title, content string) string {
	memberName := ""
	if m, ok := t.members[memberID]; ok {
		memberName = m.Name()
	}

	report := Report{
		ID:        fmt.Sprintf("report-%d", len(t.reports)+1),
		Title:     title,
		Content:   content,
		CreatedBy: memberID,
		AgentName: memberName,
		CreatedAt: time.Now(),
	}

	t.reports = append(t.reports, report)
	return report.ID
}

// getReportsInternal 获取所有报告
func (t *TeamActor) getReportsInternal() []Report {
	result := make([]Report, len(t.reports))
	copy(result, t.reports)
	return result
}

// getReportInternal 获取指定报告
func (t *TeamActor) getReportInternal(reportID string) (*Report, bool) {
	for i := range t.reports {
		if t.reports[i].ID == reportID {
			report := t.reports[i]
			return &report, true
		}
	}
	return nil, false
}

// ═══════════════════════════════════════════════════════════════════════════
// 摘要与生命周期
// ═══════════════════════════════════════════════════════════════════════════

// getSummaryInternal 获取团队状态摘要
func (t *TeamActor) getSummaryInternal() Summary {
	summary := Summary{
		Name:        t.name,
		Description: t.description,
		MemberCount: len(t.members),
	}

	if t.leader != nil {
		summary.LeaderID = t.leader.ID()
		summary.LeaderName = t.leader.Name()
	}

	// 统计任务
	for _, task := range t.taskBoard.Tasks {
		summary.TotalTasks++
		switch task.Status {
		case TaskStatusPending:
			summary.PendingTasks++
		case TaskStatusInProgress:
			summary.InProgressTasks++
		case TaskStatusCompleted:
			summary.CompletedTasks++
		case TaskStatusBlocked:
			summary.BlockedTasks++
		}
	}

	// 最近 5 条进度
	limit := min(5, len(t.progressLog))
	if limit > 0 {
		start := len(t.progressLog) - limit
		summary.RecentProgress = make([]Progress, limit)
		copy(summary.RecentProgress, t.progressLog[start:])
	}

	return summary
}

// handleShutdown 解散团队
func (t *TeamActor) handleShutdown() {
	// 关闭所有实现了 Closeable 的成员
	for _, m := range t.members {
		if closer, ok := m.(Closeable); ok {
			_ = closer.Close()
		}
	}
	t.members = make(map[string]Member)
	t.leader = nil
}

// Stats 获取统计信息
func (t *TeamActor) Stats() *actor.ActorStats {
	return t.stats.Stats()
}
