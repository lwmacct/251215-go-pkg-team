package team

import "time"

// ═══════════════════════════════════════════════════════════════════════════
// Task 相关类型
// ═══════════════════════════════════════════════════════════════════════════

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"     // 待处理
	TaskStatusInProgress TaskStatus = "in_progress" // 进行中
	TaskStatusCompleted  TaskStatus = "completed"   // 已完成
	TaskStatusBlocked    TaskStatus = "blocked"     // 阻塞
	TaskStatusCancelled  TaskStatus = "cancelled"   // 已取消
)

// Task 任务
type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	AssignedTo  string     `json:"assigned_to,omitempty"` // Agent ID
	Status      TaskStatus `json:"status"`
	Priority    int        `json:"priority,omitempty"` // 1-5, 5 最高
	CreatedBy   string     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// IsTerminal reports whether the task is in a terminal state.
func (t *Task) IsTerminal() bool {
	return t.Status == TaskStatusCompleted || t.Status == TaskStatusCancelled
}

// TaskBoard 任务看板
type TaskBoard struct {
	Tasks []Task `json:"tasks"`
}

// NewTaskBoard creates an empty TaskBoard.
func NewTaskBoard() *TaskBoard {
	return &TaskBoard{Tasks: make([]Task, 0)}
}

// GetByID returns the task with the given ID, or nil if not found.
func (b *TaskBoard) GetByID(id string) *Task {
	for i := range b.Tasks {
		if b.Tasks[i].ID == id {
			return &b.Tasks[i]
		}
	}
	return nil
}

// GetByAssignee returns all tasks assigned to the given agent.
func (b *TaskBoard) GetByAssignee(agentID string) []Task {
	result := make([]Task, 0)
	for _, task := range b.Tasks {
		if task.AssignedTo == agentID {
			result = append(result, task)
		}
	}
	return result
}

// CountByStatus returns the number of tasks with the given status.
func (b *TaskBoard) CountByStatus(status TaskStatus) int {
	count := 0
	for _, task := range b.Tasks {
		if task.Status == status {
			count++
		}
	}
	return count
}

// ═══════════════════════════════════════════════════════════════════════════
// Progress 相关类型
// ═══════════════════════════════════════════════════════════════════════════

// Progress 进度更新记录
type Progress struct {
	AgentID   string    `json:"agent_id"`
	AgentName string    `json:"agent_name"`
	TaskID    string    `json:"task_id,omitempty"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// NewProgress creates a new Progress record.
func NewProgress(agentID, agentName, message string) Progress {
	return Progress{
		AgentID:   agentID,
		AgentName: agentName,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// WithTaskID returns a copy of Progress with the given task ID.
func (p Progress) WithTaskID(taskID string) Progress {
	p.TaskID = taskID
	return p
}

// ═══════════════════════════════════════════════════════════════════════════
// Report 相关类型
// ═══════════════════════════════════════════════════════════════════════════

// Report 协作报告
type Report struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedBy string    `json:"created_by"`
	AgentName string    `json:"agent_name"`
	CreatedAt time.Time `json:"created_at"`
}

// NewReport creates a new Report.
func NewReport(id, title, content, createdBy, agentName string) Report {
	return Report{
		ID:        id,
		Title:     title,
		Content:   content,
		CreatedBy: createdBy,
		AgentName: agentName,
		CreatedAt: time.Now(),
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Summary 相关类型
// ═══════════════════════════════════════════════════════════════════════════

// Summary 团队状态摘要
type Summary struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MemberCount int    `json:"member_count"`
	LeaderID    string `json:"leader_id,omitempty"`
	LeaderName  string `json:"leader_name,omitempty"`

	// 任务统计
	TotalTasks      int `json:"total_tasks"`
	PendingTasks    int `json:"pending_tasks"`
	InProgressTasks int `json:"in_progress_tasks"`
	CompletedTasks  int `json:"completed_tasks"`
	BlockedTasks    int `json:"blocked_tasks"`

	// 最近活动
	RecentProgress []Progress `json:"recent_progress,omitempty"`
}
