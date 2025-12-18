package agent

import (
	"time"

	baseagent "github.com/lwmacct/251215-go-pkg-agent/pkg/agent"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-team/pkg/actor"
)

// Factory Agent Actor 工厂
//
// 用于批量创建配置相似的 AgentActor 实例。
type Factory struct {
	provider    llm.Provider
	defaultOpts []baseagent.Option
}

// NewFactory 创建工厂
func NewFactory(p llm.Provider, defaultOpts ...baseagent.Option) *Factory {
	return &Factory{
		provider:    p,
		defaultOpts: defaultOpts,
	}
}

// Create 创建 AgentActor
func (f *Factory) Create(opts ...baseagent.Option) (*AgentActor, error) {
	// 合并默认选项，并添加 Provider
	allOpts := make([]baseagent.Option, 0, len(f.defaultOpts)+len(opts)+1)
	allOpts = append(allOpts, f.defaultOpts...)
	allOpts = append(allOpts, opts...)
	if f.provider != nil {
		allOpts = append(allOpts, baseagent.WithProvider(f.provider))
	}

	ag, err := baseagent.NewAgent(allOpts...)
	if err != nil {
		return nil, err
	}

	return New(ag), nil
}

// CreateAndSpawn 创建并在 Actor 系统中启动
func (f *Factory) CreateAndSpawn(
	system *actor.System,
	name string,
	opts ...baseagent.Option,
) (*actor.PID, error) {
	agentActor, err := f.Create(opts...)
	if err != nil {
		return nil, err
	}

	// 设置监督策略
	props := actor.DefaultProps(name).
		WithMailboxSize(100).
		WithSupervisor(actor.NewOneForOneStrategy(3, time.Minute, actor.DefaultDecider))

	pid := system.SpawnWithProps(agentActor, props)
	return pid, nil
}
