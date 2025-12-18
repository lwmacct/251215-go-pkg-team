package team

import (
	"context"
)

// Member Agent 成员最小接口
//
// Member 定义了 Agent 作为 Team 成员所需的最小属性。
// 任何实现此接口的类型都可以被 Team 管理。
type Member interface {
	// ID 返回成员唯一标识
	ID() string

	// Name 返回成员名称
	Name() string

	// ParentID 返回父成员 ID，用于构建层级关系
	// 返回空字符串表示没有父成员（顶级成员）
	ParentID() string
}

// Runtime Agent 运行时管理接口
//
// Runtime 提供 Agent 成员的生命周期管理和查询功能。
// 这是一个通用接口，不依赖具体的 Agent 实现。
//
// Agent 是独立的一等公民，不需要 Runtime 就能工作。
// Runtime 是可选的协作层，用于多 Agent 协作场景。
type Runtime interface {
	// AddMember 添加成员到协作组
	AddMember(m Member) error

	// RemoveMember 从协作组移除成员
	RemoveMember(id string)

	// CloseMember 关闭并移除成员
	// 对于实现了 io.Closer 的成员，会先调用 Close()
	CloseMember(id string) error

	// GetMember 获取成员
	GetMember(id string) (Member, bool)

	// ListMembers 列出所有成员
	ListMembers() []Member

	// ListChildMembers 列出直接子成员
	ListChildMembers(parentID string) []Member

	// ListDescendantMembers 列出所有后代成员
	ListDescendantMembers(parentID string) []Member

	// GetMemberLineage 获取成员血统（从根到当前的路径）
	GetMemberLineage(id string) []string

	// ForkMember 从现有成员克隆新成员
	//
	// 从团队中查找源成员并克隆出新的成员实例。
	// 新成员会自动添加到团队中。
	//
	// 参数：
	//   - srcID: 源成员 ID（从团队注册表查找）
	//   - newID: 新成员 ID
	//   - opts: 配置选项（agent.Option），用于覆盖克隆的配置
	//
	// 示例：
	//   // 克隆基础 Agent 为专门的 Worker
	//   team.ForkMember("base-agent", "worker-1",
	//       agent.WithName("Worker 1"),
	//       agent.WithPrompt("Focus on testing"),
	//   )
	//
	// 错误：
	//   - 源成员不存在
	//   - 源成员不是 *agent.Agent 类型
	//   - 新 ID 已存在
	//   - 克隆失败
	ForkMember(srcID, newID string, opts ...any) error
}

// Team 完整的协作团队接口
//
// Team 在 Runtime 基础上增加了任务管理、进度追踪和报告功能。
// 核心价值：节约上下文，通过结构化信息协调工作。
type Team interface {
	Runtime

	// Info 团队信息
	Info() TeamInfo

	// GetLeader 获取 Leader（第一个加入的成员）
	GetLeader() Member

	// IsLeader 检查是否是 Leader
	IsLeader(id string) bool

	// TaskManager 任务管理
	TaskManager

	// ProgressTracker 进度追踪
	ProgressTracker

	// ReportManager 报告管理
	ReportManager

	// GetSummary 获取团队状态摘要
	GetSummary() Summary

	// Shutdown 解散团队，关闭所有成员
	Shutdown()
}

// TeamInfo 团队基本信息
type TeamInfo struct {
	Name        string
	Description string
}

// TaskManager 任务管理接口
type TaskManager interface {
	// AddTask 添加任务到看板，返回任务 ID
	AddTask(task Task) string

	// UpdateTaskStatus 更新任务状态
	UpdateTaskStatus(taskID string, status TaskStatus, agentID string) error

	// AssignTask 分配任务给成员
	AssignTask(taskID, memberID string) error

	// GetTask 获取任务
	GetTask(taskID string) (*Task, bool)

	// GetTaskBoard 获取任务看板（只读副本）
	GetTaskBoard() TaskBoard

	// GetMemberTasks 获取成员的所有任务
	GetMemberTasks(memberID string) []Task
}

// ProgressTracker 进度追踪接口
type ProgressTracker interface {
	// ReportProgress 报告进度
	ReportProgress(memberID, taskID, message string)

	// GetProgressLog 获取进度日志（最近 N 条）
	GetProgressLog(limit int) []Progress
}

// ReportManager 报告管理接口
type ReportManager interface {
	// AddReport 添加报告，返回报告 ID
	AddReport(memberID, title, content string) string

	// GetReports 获取所有报告
	GetReports() []Report

	// GetReport 获取指定报告
	GetReport(reportID string) (*Report, bool)
}

// Chateable 可对话的成员接口
//
// 如果 Member 实现了此接口，则可以进行同步对话。
type Chateable interface {
	// Chat 同步对话
	Chat(ctx context.Context, text string) (any, error)
}

// Closeable 可关闭的成员接口
//
// 如果 Member 实现了此接口，在 CloseMember 时会调用 Close()。
type Closeable interface {
	// Close 关闭成员
	Close() error
}
