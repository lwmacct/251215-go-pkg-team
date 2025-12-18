package team

import (
	"fmt"
	"sync"
	"time"

	"github.com/lwmacct/251215-go-pkg-agent/pkg/agent"
)

// BaseTeam Team 接口的通用实现
//
// BaseTeam 提供了 Team 接口的完整实现，使用通用的 Member 接口。
// 任何实现了 Member 接口的类型都可以被 BaseTeam 管理。
//
// Thread Safety: BaseTeam 是并发安全的，所有方法都使用读写锁保护。
type BaseTeam struct {
	mu sync.RWMutex

	// 团队信息
	name        string
	description string
	createdAt   time.Time

	// 成员管理
	leader  Member            // Leader（第一个加入的成员）
	members map[string]Member // 所有成员（包括 Leader）

	// 共享信息（节约上下文的关键）
	taskBoard   *TaskBoard // 任务看板
	progressLog []Progress // 进度日志
	reports     []Report   // 报告
}

// Option BaseTeam 配置选项
type Option func(*BaseTeam)

// WithName 设置团队名称
func WithName(name string) Option {
	return func(t *BaseTeam) {
		t.name = name
	}
}

// WithDescription 设置团队描述
func WithDescription(desc string) Option {
	return func(t *BaseTeam) {
		t.description = desc
	}
}

// New 创建新的 BaseTeam
func New(opts ...Option) *BaseTeam {
	t := &BaseTeam{
		name:        "default-team",
		createdAt:   time.Now(),
		members:     make(map[string]Member),
		taskBoard:   NewTaskBoard(),
		progressLog: make([]Progress, 0),
		reports:     make([]Report, 0),
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// ═══════════════════════════════════════════════════════════════════════════
// Team 信息
// ═══════════════════════════════════════════════════════════════════════════

// Info 返回团队信息
func (t *BaseTeam) Info() TeamInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return TeamInfo{
		Name:        t.name,
		Description: t.description,
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 成员管理 (Runtime 接口实现)
// ═══════════════════════════════════════════════════════════════════════════

// AddMember 添加成员到团队
//
// 第一个加入的成员自动成为 Leader。
func (t *BaseTeam) AddMember(m Member) error {
	if m == nil {
		return fmt.Errorf("member cannot be nil")
	}
	if m.ID() == "" {
		return fmt.Errorf("member ID cannot be empty")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

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

// RemoveMember 从团队移除成员
func (t *BaseTeam) RemoveMember(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()

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

// CloseMember 关闭并移除成员
//
// 如果成员实现了 Closeable 接口，会先调用 Close()。
func (t *BaseTeam) CloseMember(id string) error {
	t.mu.Lock()
	m, ok := t.members[id]
	if !ok {
		t.mu.Unlock()
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
	t.mu.Unlock()

	// 如果实现了 Closeable，调用 Close
	if closer, ok := m.(Closeable); ok {
		return closer.Close()
	}

	return nil
}

// GetMember 获取成员
func (t *BaseTeam) GetMember(id string) (Member, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	m, ok := t.members[id]
	return m, ok
}

// ListMembers 列出所有成员
func (t *BaseTeam) ListMembers() []Member {
	t.mu.RLock()
	defer t.mu.RUnlock()

	members := make([]Member, 0, len(t.members))
	for _, m := range t.members {
		members = append(members, m)
	}
	return members
}

// ═══════════════════════════════════════════════════════════════════════════
// 层级管理
// ═══════════════════════════════════════════════════════════════════════════

// ListChildMembers 列出直接子成员
func (t *BaseTeam) ListChildMembers(parentID string) []Member {
	t.mu.RLock()
	defer t.mu.RUnlock()

	children := make([]Member, 0)
	for _, m := range t.members {
		if m.ParentID() == parentID {
			children = append(children, m)
		}
	}
	return children
}

// ListDescendantMembers 列出所有后代成员
func (t *BaseTeam) ListDescendantMembers(parentID string) []Member {
	t.mu.RLock()
	defer t.mu.RUnlock()

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

// GetMemberLineage 获取成员血统（从根到当前的路径）
func (t *BaseTeam) GetMemberLineage(id string) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

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

// ForkMember 从现有成员克隆新成员
//
// 从团队注册表中查找源成员，克隆后自动添加到团队。
// 只支持克隆 *agent.Agent 类型的成员。
func (t *BaseTeam) ForkMember(srcID, newID string, opts ...any) error {
	// 检查新 ID 是否已存在
	t.mu.RLock()
	if _, exists := t.members[newID]; exists {
		t.mu.RUnlock()
		return fmt.Errorf("member with ID %s already exists", newID)
	}

	// 查找源成员
	srcMember, exists := t.members[srcID]
	t.mu.RUnlock()

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
	return t.AddMember(newAgent)
}

// ═══════════════════════════════════════════════════════════════════════════
// Leader 管理
// ═══════════════════════════════════════════════════════════════════════════

// GetLeader 获取 Leader
func (t *BaseTeam) GetLeader() Member {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.leader
}

// IsLeader 检查是否是 Leader
func (t *BaseTeam) IsLeader(id string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.leader != nil && t.leader.ID() == id
}

// ═══════════════════════════════════════════════════════════════════════════
// 任务管理 (TaskManager 接口实现)
// ═══════════════════════════════════════════════════════════════════════════

// AddTask 添加任务到看板
func (t *BaseTeam) AddTask(task Task) string {
	t.mu.Lock()
	defer t.mu.Unlock()

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

// UpdateTaskStatus 更新任务状态
func (t *BaseTeam) UpdateTaskStatus(taskID string, status TaskStatus, memberID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

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

// AssignTask 分配任务给成员
func (t *BaseTeam) AssignTask(taskID, memberID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

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

// GetTask 获取任务
func (t *BaseTeam) GetTask(taskID string) (*Task, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for i := range t.taskBoard.Tasks {
		if t.taskBoard.Tasks[i].ID == taskID {
			task := t.taskBoard.Tasks[i]
			return &task, true
		}
	}
	return nil, false
}

// GetTaskBoard 获取任务看板（只读副本）
func (t *BaseTeam) GetTaskBoard() TaskBoard {
	t.mu.RLock()
	defer t.mu.RUnlock()

	board := TaskBoard{
		Tasks: make([]Task, len(t.taskBoard.Tasks)),
	}
	copy(board.Tasks, t.taskBoard.Tasks)
	return board
}

// GetMemberTasks 获取成员的所有任务
func (t *BaseTeam) GetMemberTasks(memberID string) []Task {
	t.mu.RLock()
	defer t.mu.RUnlock()

	tasks := make([]Task, 0)
	for _, task := range t.taskBoard.Tasks {
		if task.AssignedTo == memberID {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// ═══════════════════════════════════════════════════════════════════════════
// 进度追踪 (ProgressTracker 接口实现)
// ═══════════════════════════════════════════════════════════════════════════

// ReportProgress 报告进度
func (t *BaseTeam) ReportProgress(memberID, taskID, message string) {
	t.mu.Lock()
	defer t.mu.Unlock()

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

// GetProgressLog 获取进度日志（最近 N 条）
func (t *BaseTeam) GetProgressLog(limit int) []Progress {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if limit <= 0 || limit > len(t.progressLog) {
		limit = len(t.progressLog)
	}

	// 返回最近的记录
	start := max(len(t.progressLog)-limit, 0)

	result := make([]Progress, limit)
	copy(result, t.progressLog[start:])
	return result
}

// ═══════════════════════════════════════════════════════════════════════════
// 报告管理 (ReportManager 接口实现)
// ═══════════════════════════════════════════════════════════════════════════

// AddReport 添加报告
func (t *BaseTeam) AddReport(memberID, title, content string) string {
	t.mu.Lock()
	defer t.mu.Unlock()

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

// GetReports 获取所有报告
func (t *BaseTeam) GetReports() []Report {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]Report, len(t.reports))
	copy(result, t.reports)
	return result
}

// GetReport 获取指定报告
func (t *BaseTeam) GetReport(reportID string) (*Report, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for i := range t.reports {
		if t.reports[i].ID == reportID {
			report := t.reports[i]
			return &report, true
		}
	}
	return nil, false
}

// ═══════════════════════════════════════════════════════════════════════════
// 状态与生命周期
// ═══════════════════════════════════════════════════════════════════════════

// GetSummary 获取团队状态摘要
func (t *BaseTeam) GetSummary() Summary {
	t.mu.RLock()
	defer t.mu.RUnlock()

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

// Shutdown 解散团队，关闭所有成员
func (t *BaseTeam) Shutdown() {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 关闭所有实现了 Closeable 的成员
	for _, m := range t.members {
		if closer, ok := m.(Closeable); ok {
			_ = closer.Close()
		}
	}
	t.members = make(map[string]Member)
	t.leader = nil
}

// 确保 BaseTeam 实现了 Team 接口
var _ Team = (*BaseTeam)(nil)
