package main

import (
	"github.com/lwmacct/251215-go-pkg-team/pkg/actor"
	"github.com/lwmacct/251215-go-pkg-team/pkg/team"
)

// Agent 角色常量
const (
	RoleTriage   = "triage"
	RoleTech     = "tech-support"
	RoleBilling  = "billing-support"
	RoleEscalate = "escalation"
)

// Agent 名称
var AgentNames = map[string]string{
	RoleTriage:   "分诊专员",
	RoleTech:     "技术支持",
	RoleBilling:  "账单支持",
	RoleEscalate: "升级处理",
}

// Agent Prompt 定义
var AgentPrompts = map[string]string{
	RoleTriage: `你是一名客服分诊专员。
你的职责是分析客户问题并进行分类。
请始终以 JSON 格式回复，包含以下字段：
- type: "technical"（技术问题）、"billing"（账单问题）或 "escalate"（需要升级）
- priority: "low"（低）、"medium"（中）或 "high"（高）
- summary: 问题简要描述`,

	RoleTech: `你是一名技术支持专家。
你负责处理以下技术问题：
- 账户访问问题
- 密码重置
- 系统错误
- 功能使用咨询
请提供清晰、分步骤的解决方案。`,

	RoleBilling: `你是一名账单支持专家。
你负责处理以下账单问题：
- 支付问题
- 退款申请
- 订阅变更
- 发票咨询
请保持专业和耐心。`,

	RoleEscalate: `你是一名高级升级处理专家。
你负责处理需要特殊关注的复杂问题：
- 跨部门协调
- 特殊授权
- 紧急情况
- VIP 客户服务
请确认问题并提供明确的处理方案。`,
}

// Mock 响应（用于演示）
var MockResponses = map[string]string{
	RoleTriage:   `{"type": "technical", "priority": "medium", "summary": "用户无法访问账户"}`,
	RoleTech:     "我已检查您的账户状态。您的密码已重置，请查收邮件中的重置链接，按照说明创建新密码即可。",
	RoleBilling:  "我理解您对账单问题的担忧。我已处理您的退款申请，金额将在 3-5 个工作日内退回您的账户。",
	RoleEscalate: "我已将您的问题升级至高级工程师团队。专家将在 24 小时内与您联系。您的工单编号是 ESC-2024-001。",
}

// AgentMember 包装 AgentActor PID 实现 team.Member 接口
type AgentMember struct {
	id       string
	name     string
	parentID string
	pid      *actor.PID
}

// NewAgentMember 创建 AgentMember
func NewAgentMember(role string, pid *actor.PID, parentID string) *AgentMember {
	return &AgentMember{
		id:       role,
		name:     AgentNames[role],
		parentID: parentID,
		pid:      pid,
	}
}

// ID 返回成员 ID
func (m *AgentMember) ID() string { return m.id }

// Name 返回成员名称
func (m *AgentMember) Name() string { return m.name }

// ParentID 返回父成员 ID
func (m *AgentMember) ParentID() string { return m.parentID }

// PID 返回 Actor PID
func (m *AgentMember) PID() *actor.PID { return m.pid }

// 确保 AgentMember 实现了 Member 接口
var _ team.Member = (*AgentMember)(nil)
