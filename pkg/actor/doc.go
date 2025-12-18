// Package actor 提供轻量级 Actor 模型实现
//
// Actor 模式是一种并发计算模型，每个 Actor 是独立的计算单元：
// • 拥有私有状态（无需锁保护）
// • 通过消息邮箱（mailbox）接收消息
// • 消息处理串行化（一次处理一条）
// • 可以创建子 Actor、发送消息、修改自身状态
//
// # 核心组件
//
// [System] 是 Actor 系统的入口，管理所有 Actor 的生命周期：
//
//	sys := actor.NewSystem("my-system")
//	defer sys.Shutdown()
//
// [Actor] 接口定义消息处理行为，[ActorFunc] 提供函数式快捷方式。
//
// [PID] 是 Actor 的唯一标识，用于消息发送。[PID.Tell] 异步发送消息（fire-and-forget），
// [PID.Request] 同步请求并等待响应。
//
// [Context] 提供 Actor 运行时上下文，支持回复消息、创建子 Actor、监控等操作。
//
// # 监督策略
//
// 监督策略决定子 Actor 失败时的处理方式。[OneForOneStrategy] 只重启失败的 Actor，
// [AllForOneStrategy] 重启所有子 Actor，[ExponentialBackoffStrategy] 指数退避重启。
//
// 监督指令 [Directive]：DirectiveResume 恢复运行，DirectiveRestart 重启 Actor，
// DirectiveStop 停止 Actor，DirectiveEscalate 上报给父 Actor。
//
// # 系统消息
//
// Actor 生命周期中会收到以下系统消息：[Started] 启动完成，[Stopping] 正在停止，
// [Stopped] 已停止，[Restarting] 正在重启，[PoisonPill] 优雅停止请求，
// [Terminated] 被监控的 Actor 终止。
//
// # 最佳实践
//
// 1. 消息不可变，发送后不要修改消息内容
// 2. 避免阻塞，Receive 方法中不要执行长时间操作
// 3. 使用监督，为关键 Actor 配置监督策略
// 4. 处理系统消息，正确处理生命周期消息
// 5. 合理设置邮箱，根据负载调整邮箱大小
//
// 完整使用示例请参考 example_test.go 或运行 go doc -all。
package actor
