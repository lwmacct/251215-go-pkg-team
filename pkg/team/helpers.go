package team

import (
	"context"
	"time"

	"github.com/lwmacct/251215-go-pkg-team/pkg/actor"
)

// ═══════════════════════════════════════════════════════════════════════════
// 成员管理便捷函数
// ═══════════════════════════════════════════════════════════════════════════

// DoAddMember 向 TeamActor 添加成员
func DoAddMember(pid *actor.PID, m Member, timeout time.Duration) error {
	replyCh := make(chan error, 1)

	pid.Tell(&AddMemberMsg{
		Member:    m,
		ReplyChan: replyCh,
	})

	select {
	case err := <-replyCh:
		return err
	case <-time.After(timeout):
		return context.DeadlineExceeded
	}
}

// DoGetMember 从 TeamActor 获取成员
func DoGetMember(pid *actor.PID, memberID string, timeout time.Duration) (Member, bool, error) {
	replyCh := make(chan *GetMemberResult, 1)

	pid.Tell(&GetMemberMsg{
		MemberID:  memberID,
		ReplyChan: replyCh,
	})

	select {
	case result := <-replyCh:
		return result.Member, result.Found, nil
	case <-time.After(timeout):
		return nil, false, context.DeadlineExceeded
	}
}

// DoListMembers 列出所有成员
func DoListMembers(pid *actor.PID, timeout time.Duration) ([]Member, error) {
	replyCh := make(chan []Member, 1)

	pid.Tell(&ListMembersMsg{
		ReplyChan: replyCh,
	})

	select {
	case members := <-replyCh:
		return members, nil
	case <-time.After(timeout):
		return nil, context.DeadlineExceeded
	}
}

// DoCloseMember 关闭并移除成员
func DoCloseMember(pid *actor.PID, memberID string, timeout time.Duration) error {
	replyCh := make(chan error, 1)

	pid.Tell(&CloseMemberMsg{
		MemberID:  memberID,
		ReplyChan: replyCh,
	})

	select {
	case err := <-replyCh:
		return err
	case <-time.After(timeout):
		return context.DeadlineExceeded
	}
}

// DoForkMember 克隆成员
func DoForkMember(pid *actor.PID, srcID, newID string, opts []any, timeout time.Duration) error {
	replyCh := make(chan error, 1)

	pid.Tell(&ForkMemberMsg{
		SourceID:  srcID,
		NewID:     newID,
		Options:   opts,
		ReplyChan: replyCh,
	})

	select {
	case err := <-replyCh:
		return err
	case <-time.After(timeout):
		return context.DeadlineExceeded
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Leader 管理便捷函数
// ═══════════════════════════════════════════════════════════════════════════

// DoGetLeader 获取 Leader
func DoGetLeader(pid *actor.PID, timeout time.Duration) (Member, error) {
	replyCh := make(chan Member, 1)

	pid.Tell(&GetLeaderMsg{
		ReplyChan: replyCh,
	})

	select {
	case leader := <-replyCh:
		return leader, nil
	case <-time.After(timeout):
		return nil, context.DeadlineExceeded
	}
}

// DoIsLeader 检查是否是 Leader
func DoIsLeader(pid *actor.PID, memberID string, timeout time.Duration) (bool, error) {
	replyCh := make(chan bool, 1)

	pid.Tell(&IsLeaderMsg{
		MemberID:  memberID,
		ReplyChan: replyCh,
	})

	select {
	case isLeader := <-replyCh:
		return isLeader, nil
	case <-time.After(timeout):
		return false, context.DeadlineExceeded
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 任务管理便捷函数
// ═══════════════════════════════════════════════════════════════════════════

// DoAddTask 添加任务
func DoAddTask(pid *actor.PID, task Task, timeout time.Duration) (string, error) {
	replyCh := make(chan string, 1)

	pid.Tell(&AddTaskMsg{
		Task:      task,
		ReplyChan: replyCh,
	})

	select {
	case taskID := <-replyCh:
		return taskID, nil
	case <-time.After(timeout):
		return "", context.DeadlineExceeded
	}
}

// DoUpdateTaskStatus 更新任务状态
func DoUpdateTaskStatus(pid *actor.PID, taskID string, status TaskStatus, memberID string, timeout time.Duration) error {
	replyCh := make(chan error, 1)

	pid.Tell(&UpdateTaskStatusMsg{
		TaskID:    taskID,
		Status:    status,
		MemberID:  memberID,
		ReplyChan: replyCh,
	})

	select {
	case err := <-replyCh:
		return err
	case <-time.After(timeout):
		return context.DeadlineExceeded
	}
}

// DoAssignTask 分配任务
func DoAssignTask(pid *actor.PID, taskID, memberID string, timeout time.Duration) error {
	replyCh := make(chan error, 1)

	pid.Tell(&AssignTaskMsg{
		TaskID:    taskID,
		MemberID:  memberID,
		ReplyChan: replyCh,
	})

	select {
	case err := <-replyCh:
		return err
	case <-time.After(timeout):
		return context.DeadlineExceeded
	}
}

// DoGetTask 获取任务
func DoGetTask(pid *actor.PID, taskID string, timeout time.Duration) (*Task, bool, error) {
	replyCh := make(chan *GetTaskResult, 1)

	pid.Tell(&GetTaskMsg{
		TaskID:    taskID,
		ReplyChan: replyCh,
	})

	select {
	case result := <-replyCh:
		return result.Task, result.Found, nil
	case <-time.After(timeout):
		return nil, false, context.DeadlineExceeded
	}
}

// DoGetTaskBoard 获取任务看板
func DoGetTaskBoard(pid *actor.PID, timeout time.Duration) (TaskBoard, error) {
	replyCh := make(chan TaskBoard, 1)

	pid.Tell(&GetTaskBoardMsg{
		ReplyChan: replyCh,
	})

	select {
	case board := <-replyCh:
		return board, nil
	case <-time.After(timeout):
		return TaskBoard{}, context.DeadlineExceeded
	}
}

// DoGetMemberTasks 获取成员任务
func DoGetMemberTasks(pid *actor.PID, memberID string, timeout time.Duration) ([]Task, error) {
	replyCh := make(chan []Task, 1)

	pid.Tell(&GetMemberTasksMsg{
		MemberID:  memberID,
		ReplyChan: replyCh,
	})

	select {
	case tasks := <-replyCh:
		return tasks, nil
	case <-time.After(timeout):
		return nil, context.DeadlineExceeded
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 进度追踪便捷函数
// ═══════════════════════════════════════════════════════════════════════════

// DoReportProgress 报告进度（fire-and-forget）
func DoReportProgress(pid *actor.PID, memberID, taskID, message string) {
	pid.Tell(&ReportProgressMsg{
		MemberID: memberID,
		TaskID:   taskID,
		Message:  message,
	})
}

// DoGetProgressLog 获取进度日志
func DoGetProgressLog(pid *actor.PID, limit int, timeout time.Duration) ([]Progress, error) {
	replyCh := make(chan []Progress, 1)

	pid.Tell(&GetProgressLogMsg{
		Limit:     limit,
		ReplyChan: replyCh,
	})

	select {
	case log := <-replyCh:
		return log, nil
	case <-time.After(timeout):
		return nil, context.DeadlineExceeded
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 报告管理便捷函数
// ═══════════════════════════════════════════════════════════════════════════

// DoAddReport 添加报告
func DoAddReport(pid *actor.PID, memberID, title, content string, timeout time.Duration) (string, error) {
	replyCh := make(chan string, 1)

	pid.Tell(&AddReportMsg{
		MemberID:  memberID,
		Title:     title,
		Content:   content,
		ReplyChan: replyCh,
	})

	select {
	case reportID := <-replyCh:
		return reportID, nil
	case <-time.After(timeout):
		return "", context.DeadlineExceeded
	}
}

// DoGetReports 获取所有报告
func DoGetReports(pid *actor.PID, timeout time.Duration) ([]Report, error) {
	replyCh := make(chan []Report, 1)

	pid.Tell(&GetReportsMsg{
		ReplyChan: replyCh,
	})

	select {
	case reports := <-replyCh:
		return reports, nil
	case <-time.After(timeout):
		return nil, context.DeadlineExceeded
	}
}

// DoGetReport 获取指定报告
func DoGetReport(pid *actor.PID, reportID string, timeout time.Duration) (*Report, bool, error) {
	replyCh := make(chan *GetReportResult, 1)

	pid.Tell(&GetReportMsg{
		ReportID:  reportID,
		ReplyChan: replyCh,
	})

	select {
	case result := <-replyCh:
		return result.Report, result.Found, nil
	case <-time.After(timeout):
		return nil, false, context.DeadlineExceeded
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 团队信息便捷函数
// ═══════════════════════════════════════════════════════════════════════════

// DoGetInfo 获取团队信息
func DoGetInfo(pid *actor.PID, timeout time.Duration) (TeamInfo, error) {
	replyCh := make(chan TeamInfo, 1)

	pid.Tell(&GetInfoMsg{
		ReplyChan: replyCh,
	})

	select {
	case info := <-replyCh:
		return info, nil
	case <-time.After(timeout):
		return TeamInfo{}, context.DeadlineExceeded
	}
}

// DoGetSummary 获取团队状态摘要
func DoGetSummary(pid *actor.PID, timeout time.Duration) (Summary, error) {
	replyCh := make(chan Summary, 1)

	pid.Tell(&GetSummaryMsg{
		ReplyChan: replyCh,
	})

	select {
	case summary := <-replyCh:
		return summary, nil
	case <-time.After(timeout):
		return Summary{}, context.DeadlineExceeded
	}
}

// DoShutdown 解散团队（fire-and-forget）
func DoShutdown(pid *actor.PID, reason string) {
	pid.Tell(&ShutdownMsg{
		Reason: reason,
	})
}
