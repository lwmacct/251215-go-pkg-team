package team

// ═══════════════════════════════════════════════════════════════════════════
// 成员管理消息 (Runtime 接口)
// ═══════════════════════════════════════════════════════════════════════════

// AddMemberMsg 添加成员请求
//
// 第一个加入的成员自动成为 Leader。
type AddMemberMsg struct {
	Member    Member
	ReplyChan chan error
}

// Kind 实现 actor.Message 接口
func (m *AddMemberMsg) Kind() string { return "team.add_member" }

// RemoveMemberMsg 移除成员请求
type RemoveMemberMsg struct {
	MemberID string
}

// Kind 实现 actor.Message 接口
func (m *RemoveMemberMsg) Kind() string { return "team.remove_member" }

// CloseMemberMsg 关闭并移除成员请求
//
// 如果成员实现了 Closeable 接口，会先调用 Close()。
type CloseMemberMsg struct {
	MemberID  string
	ReplyChan chan error
}

// Kind 实现 actor.Message 接口
func (m *CloseMemberMsg) Kind() string { return "team.close_member" }

// GetMemberMsg 获取成员请求
type GetMemberMsg struct {
	MemberID  string
	ReplyChan chan *GetMemberResult
}

// Kind 实现 actor.Message 接口
func (m *GetMemberMsg) Kind() string { return "team.get_member" }

// GetMemberResult 获取成员结果
type GetMemberResult struct {
	Member Member
	Found  bool
}

// ListMembersMsg 列出所有成员请求
type ListMembersMsg struct {
	ReplyChan chan []Member
}

// Kind 实现 actor.Message 接口
func (m *ListMembersMsg) Kind() string { return "team.list_members" }

// ListChildMembersMsg 列出直接子成员请求
type ListChildMembersMsg struct {
	ParentID  string
	ReplyChan chan []Member
}

// Kind 实现 actor.Message 接口
func (m *ListChildMembersMsg) Kind() string { return "team.list_child_members" }

// ListDescendantMembersMsg 列出所有后代成员请求
type ListDescendantMembersMsg struct {
	ParentID  string
	ReplyChan chan []Member
}

// Kind 实现 actor.Message 接口
func (m *ListDescendantMembersMsg) Kind() string { return "team.list_descendant_members" }

// GetMemberLineageMsg 获取成员血统请求
type GetMemberLineageMsg struct {
	MemberID  string
	ReplyChan chan []string
}

// Kind 实现 actor.Message 接口
func (m *GetMemberLineageMsg) Kind() string { return "team.get_member_lineage" }

// ForkMemberMsg 克隆成员请求
//
// 从团队中查找源成员并克隆出新的成员实例。
// 只支持克隆 *agent.Agent 类型的成员。
type ForkMemberMsg struct {
	SourceID  string
	NewID     string
	Options   []any // agent.Option
	ReplyChan chan error
}

// Kind 实现 actor.Message 接口
func (m *ForkMemberMsg) Kind() string { return "team.fork_member" }

// ═══════════════════════════════════════════════════════════════════════════
// Leader 管理消息
// ═══════════════════════════════════════════════════════════════════════════

// GetLeaderMsg 获取 Leader 请求
type GetLeaderMsg struct {
	ReplyChan chan Member
}

// Kind 实现 actor.Message 接口
func (m *GetLeaderMsg) Kind() string { return "team.get_leader" }

// IsLeaderMsg 检查是否是 Leader 请求
type IsLeaderMsg struct {
	MemberID  string
	ReplyChan chan bool
}

// Kind 实现 actor.Message 接口
func (m *IsLeaderMsg) Kind() string { return "team.is_leader" }

// ═══════════════════════════════════════════════════════════════════════════
// 任务管理消息 (TaskManager 接口)
// ═══════════════════════════════════════════════════════════════════════════

// AddTaskMsg 添加任务请求
//
// 返回生成的任务 ID。
type AddTaskMsg struct {
	Task      Task
	ReplyChan chan string
}

// Kind 实现 actor.Message 接口
func (m *AddTaskMsg) Kind() string { return "team.add_task" }

// UpdateTaskStatusMsg 更新任务状态请求
type UpdateTaskStatusMsg struct {
	TaskID    string
	Status    TaskStatus
	MemberID  string
	ReplyChan chan error
}

// Kind 实现 actor.Message 接口
func (m *UpdateTaskStatusMsg) Kind() string { return "team.update_task_status" }

// AssignTaskMsg 分配任务请求
type AssignTaskMsg struct {
	TaskID    string
	MemberID  string
	ReplyChan chan error
}

// Kind 实现 actor.Message 接口
func (m *AssignTaskMsg) Kind() string { return "team.assign_task" }

// GetTaskMsg 获取任务请求
type GetTaskMsg struct {
	TaskID    string
	ReplyChan chan *GetTaskResult
}

// Kind 实现 actor.Message 接口
func (m *GetTaskMsg) Kind() string { return "team.get_task" }

// GetTaskResult 获取任务结果
type GetTaskResult struct {
	Task  *Task
	Found bool
}

// GetTaskBoardMsg 获取任务看板请求
type GetTaskBoardMsg struct {
	ReplyChan chan TaskBoard
}

// Kind 实现 actor.Message 接口
func (m *GetTaskBoardMsg) Kind() string { return "team.get_task_board" }

// GetMemberTasksMsg 获取成员任务请求
type GetMemberTasksMsg struct {
	MemberID  string
	ReplyChan chan []Task
}

// Kind 实现 actor.Message 接口
func (m *GetMemberTasksMsg) Kind() string { return "team.get_member_tasks" }

// ═══════════════════════════════════════════════════════════════════════════
// 进度追踪消息 (ProgressTracker 接口)
// ═══════════════════════════════════════════════════════════════════════════

// ReportProgressMsg 报告进度请求
type ReportProgressMsg struct {
	MemberID string
	TaskID   string
	Message  string
}

// Kind 实现 actor.Message 接口
func (m *ReportProgressMsg) Kind() string { return "team.report_progress" }

// GetProgressLogMsg 获取进度日志请求
type GetProgressLogMsg struct {
	Limit     int
	ReplyChan chan []Progress
}

// Kind 实现 actor.Message 接口
func (m *GetProgressLogMsg) Kind() string { return "team.get_progress_log" }

// ═══════════════════════════════════════════════════════════════════════════
// 报告管理消息 (ReportManager 接口)
// ═══════════════════════════════════════════════════════════════════════════

// AddReportMsg 添加报告请求
//
// 返回生成的报告 ID。
type AddReportMsg struct {
	MemberID  string
	Title     string
	Content   string
	ReplyChan chan string
}

// Kind 实现 actor.Message 接口
func (m *AddReportMsg) Kind() string { return "team.add_report" }

// GetReportsMsg 获取所有报告请求
type GetReportsMsg struct {
	ReplyChan chan []Report
}

// Kind 实现 actor.Message 接口
func (m *GetReportsMsg) Kind() string { return "team.get_reports" }

// GetReportMsg 获取指定报告请求
type GetReportMsg struct {
	ReportID  string
	ReplyChan chan *GetReportResult
}

// Kind 实现 actor.Message 接口
func (m *GetReportMsg) Kind() string { return "team.get_report" }

// GetReportResult 获取报告结果
type GetReportResult struct {
	Report *Report
	Found  bool
}

// ═══════════════════════════════════════════════════════════════════════════
// 团队信息与生命周期消息
// ═══════════════════════════════════════════════════════════════════════════

// GetInfoMsg 获取团队信息请求
type GetInfoMsg struct {
	ReplyChan chan TeamInfo
}

// Kind 实现 actor.Message 接口
func (m *GetInfoMsg) Kind() string { return "team.get_info" }

// GetSummaryMsg 获取团队状态摘要请求
type GetSummaryMsg struct {
	ReplyChan chan Summary
}

// Kind 实现 actor.Message 接口
func (m *GetSummaryMsg) Kind() string { return "team.get_summary" }

// ShutdownMsg 解散团队请求
//
// 关闭所有成员，清理资源。
type ShutdownMsg struct {
	Reason string
}

// Kind 实现 actor.Message 接口
func (m *ShutdownMsg) Kind() string { return "team.shutdown" }

// ═══════════════════════════════════════════════════════════════════════════
// 事件通知消息
// ═══════════════════════════════════════════════════════════════════════════

// MemberJoinedEvent 成员加入事件
type MemberJoinedEvent struct {
	MemberID   string
	MemberName string
	IsLeader   bool
}

// Kind 实现 actor.Message 接口
func (m *MemberJoinedEvent) Kind() string { return "team.event.member_joined" }

// MemberLeftEvent 成员离开事件
type MemberLeftEvent struct {
	MemberID   string
	MemberName string
	Reason     string
}

// Kind 实现 actor.Message 接口
func (m *MemberLeftEvent) Kind() string { return "team.event.member_left" }

// TaskStatusChangedEvent 任务状态变更事件
type TaskStatusChangedEvent struct {
	TaskID    string
	OldStatus TaskStatus
	NewStatus TaskStatus
	MemberID  string
}

// Kind 实现 actor.Message 接口
func (m *TaskStatusChangedEvent) Kind() string { return "team.event.task_status_changed" }

// TaskAssignedEvent 任务分配事件
type TaskAssignedEvent struct {
	TaskID   string
	MemberID string
}

// Kind 实现 actor.Message 接口
func (m *TaskAssignedEvent) Kind() string { return "team.event.task_assigned" }

// ProgressReportedEvent 进度报告事件
type ProgressReportedEvent struct {
	Progress Progress
}

// Kind 实现 actor.Message 接口
func (m *ProgressReportedEvent) Kind() string { return "team.event.progress_reported" }
