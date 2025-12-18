// Package team 提供多 Agent 协作的核心接口和实现
//
// # Overview
//
// team 包提供多 Agent 协作场景所需的接口、数据类型和实现，包括：
//   - [Member]: Agent 成员最小接口
//   - [Runtime]: Agent 运行时管理接口
//   - [Team]: 完整的协作团队接口
//   - [BaseTeam]: Team 接口的 RWMutex 实现（传统方式）
//   - [TeamActor]: Team 的 Actor 实现（推荐，lock-free）
//   - [Task], [Progress], [Report], [Summary]: 协作数据类型
//
// # Design Philosophy
//
// Agent 是独立的一等公民，不需要 Team 就能工作。
// Team 是可选的协作层，用于多 Agent 协作场景：
//   - 成员管理：Agent 的加入、移除、查找
//   - 层级关系：父子关系、汇报链
//   - 任务协调：任务看板、进度共享、报告汇总
//
// # Two Implementations
//
// 本包提供两种 Team 实现：
//
//	BaseTeam   - RWMutex 保护，适合简单场景
//	TeamActor  - Actor 模型，lock-free 并发，推荐用于高并发场景
//
// # Interface Hierarchy
//
//	Member   - Agent 成员最小接口（ID、名称、父级）
//	Runtime  - 运行时管理接口（成员管理 + 层级查询）
//	Team     - 完整团队接口（Runtime + 任务/进度/报告）
//
// # Usage: BaseTeam (Traditional)
//
// RWMutex-based 实现：
//
//	// 创建团队
//	myTeam := team.New(
//	    team.WithName("dev-team"),
//	    team.WithDescription("Development team"),
//	)
//
//	// 添加成员
//	myTeam.AddMember(agent1)
//	myTeam.AddMember(agent2)
//
//	// 任务管理
//	taskID := myTeam.AddTask(team.Task{
//	    Title: "Implement feature X",
//	})
//	myTeam.AssignTask(taskID, agent1.ID())
//
//	// 获取状态摘要
//	summary := myTeam.GetSummary()
//
// # Usage: TeamActor (Recommended)
//
// Actor-based 实现，通过消息传递实现 lock-free 并发：
//
//	// 创建 Actor 系统
//	sys := actor.NewSystem("team-system")
//	defer sys.Shutdown()
//
//	// 创建 TeamActor
//	teamActor := team.NewTeamActor("dev-team",
//	    team.WithDescription("Development team"),
//	)
//	teamPID := sys.Spawn(teamActor, "team")
//
//	// 添加成员（通过消息）
//	err := team.DoAddMember(teamPID, agent1, 5*time.Second)
//
//	// 添加任务
//	taskID, err := team.DoAddTask(teamPID, team.Task{
//	    Title: "Implement feature X",
//	}, 5*time.Second)
//
//	// 分配任务
//	err = team.DoAssignTask(teamPID, taskID, agent1.ID(), 5*time.Second)
//
//	// 报告进度（fire-and-forget）
//	team.DoReportProgress(teamPID, agent1.ID(), taskID, "Started working")
//
//	// 获取摘要
//	summary, err := team.DoGetSummary(teamPID, 5*time.Second)
//
// # Thread Safety
//
// [BaseTeam] 使用 RWMutex，并发安全但存在锁竞争。
// [TeamActor] 使用 Actor 模型，天然 lock-free 并发安全。
//
// 对于高并发场景，推荐使用 TeamActor。
package team
