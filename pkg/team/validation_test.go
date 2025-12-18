package team

import (
	"fmt"
	"sync"
	"testing"
)

// mockMember 用于测试的 mock Member
type mockMember struct {
	id       string
	name     string
	parentID string
}

func (m *mockMember) ID() string       { return m.id }
func (m *mockMember) Name() string     { return m.name }
func (m *mockMember) ParentID() string { return m.parentID }

// TestDetectCycle_BasicCycle 测试基本的循环依赖：A → B → A
func TestDetectCycle_BasicCycle(t *testing.T) {
	members := map[string]Member{
		"B": &mockMember{id: "B", name: "B", parentID: "A"},
	}

	// 尝试添加 A，其 parentID 是 B，这会形成循环
	err := detectCycle(members, "A", "B")
	if err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}
	if err.Error() != "adding member A with parent B would create a cycle" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestDetectCycle_MultiLevelCycle 测试多层循环：A → B → C → A
func TestDetectCycle_MultiLevelCycle(t *testing.T) {
	members := map[string]Member{
		"B": &mockMember{id: "B", name: "B", parentID: "A"},
		"C": &mockMember{id: "C", name: "C", parentID: "B"},
	}

	// 尝试添加 A，其 parentID 是 C，这会形成循环
	err := detectCycle(members, "A", "C")
	if err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}
}

// TestDetectCycle_SelfReference 测试自引用：A.ParentID = A.ID
func TestDetectCycle_SelfReference(t *testing.T) {
	members := map[string]Member{}

	// 尝试添加 A，其 parentID 是自己
	err := detectCycle(members, "A", "A")
	if err == nil {
		t.Fatal("expected self-reference error, got nil")
	}
}

// TestDetectCycle_NormalHierarchy 测试正常层级：A → B → C（不应报错）
func TestDetectCycle_NormalHierarchy(t *testing.T) {
	members := map[string]Member{
		"A": &mockMember{id: "A", name: "A", parentID: ""},
		"B": &mockMember{id: "B", name: "B", parentID: "A"},
	}

	// 添加 C，其 parentID 是 B，不形成循环
	err := detectCycle(members, "C", "B")
	if err != nil {
		t.Fatalf("unexpected error for normal hierarchy: %v", err)
	}
}

// TestDetectCycle_RootNode 测试根节点（parentID 为空）
func TestDetectCycle_RootNode(t *testing.T) {
	members := map[string]Member{}

	// 根节点没有 parent，不应报错
	err := detectCycle(members, "A", "")
	if err != nil {
		t.Fatalf("unexpected error for root node: %v", err)
	}
}

// TestDetectCycle_ParentNotFound 测试父成员不存在的情况
func TestDetectCycle_ParentNotFound(t *testing.T) {
	members := map[string]Member{}

	// 尝试添加 B，其 parentID 是不存在的 A
	err := detectCycle(members, "B", "A")
	if err == nil {
		t.Fatal("expected parent not found error, got nil")
	}
	if err.Error() != "parent member A not found" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestDetectCycle_ExistingCycle 测试已存在的异常循环
func TestDetectCycle_ExistingCycle(t *testing.T) {
	// 构建一个异常的循环结构（测试防御性编程）
	memberA := &mockMember{id: "A", name: "A", parentID: "B"}
	memberB := &mockMember{id: "B", name: "B", parentID: "A"}

	members := map[string]Member{
		"A": memberA,
		"B": memberB,
	}

	// 尝试添加 C，parentID 是 A，应该检测到已存在的循环
	err := detectCycle(members, "C", "A")
	if err == nil {
		t.Fatal("expected existing cycle detection error, got nil")
	}
}

// TestCheckDepthLimit_BoundaryDepth 测试边界深度
func TestCheckDepthLimit_BoundaryDepth(t *testing.T) {
	// 构建深度接近限制的结构：root → A → B → C
	members := map[string]Member{
		"root": &mockMember{id: "root", name: "root", parentID: ""},
		"A":    &mockMember{id: "A", name: "A", parentID: "root"},
		"B":    &mockMember{id: "B", name: "B", parentID: "A"},
		"C":    &mockMember{id: "C", name: "C", parentID: "B"},
	}

	// 添加 D（parentID="C"），C 的深度是 4，D 的深度会是 5，应该通过
	err := checkDepthLimit(members, "C", MaxMemberDepth)
	if err != nil {
		t.Fatalf("unexpected error for adding D (depth 5): %v", err)
	}

	// 添加 E（parentID="D"），D 的深度是 5，E 的深度会是 6，应该拒绝
	members["D"] = &mockMember{id: "D", name: "D", parentID: "C"}
	err = checkDepthLimit(members, "D", MaxMemberDepth)
	if err == nil {
		t.Fatal("expected depth limit error for adding E (depth 6), got nil")
	}
}

// TestCalculateDepth 测试深度计算
func TestCalculateDepth(t *testing.T) {
	tests := []struct {
		name     string
		members  map[string]Member
		memberID string
		want     int
	}{
		{
			name:     "root node",
			members:  map[string]Member{},
			memberID: "",
			want:     0,
		},
		{
			name: "depth 1",
			members: map[string]Member{
				"A": &mockMember{id: "A", name: "A", parentID: ""},
			},
			memberID: "A",
			want:     1,
		},
		{
			name: "depth 3",
			members: map[string]Member{
				"A": &mockMember{id: "A", name: "A", parentID: ""},
				"B": &mockMember{id: "B", name: "B", parentID: "A"},
				"C": &mockMember{id: "C", name: "C", parentID: "B"},
			},
			memberID: "C",
			want:     3,
		},
		{
			name: "parent not found",
			members: map[string]Member{
				"B": &mockMember{id: "B", name: "B", parentID: "A"},
			},
			memberID: "B",
			want:     1, // 遇到不存在的 parent 时停止
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateDepth(tt.members, tt.memberID)
			if got != tt.want {
				t.Errorf("calculateDepth() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestAddMember_CycleDetection 集成测试：通过 AddMember 测试循环检测
func TestAddMember_CycleDetection(t *testing.T) {
	team := New()

	// 添加正常的层级结构
	memberA := &mockMember{id: "A", name: "A", parentID: ""}
	memberB := &mockMember{id: "B", name: "B", parentID: "A"}

	if err := team.AddMember(memberA); err != nil {
		t.Fatalf("failed to add member A: %v", err)
	}
	if err := team.AddMember(memberB); err != nil {
		t.Fatalf("failed to add member B: %v", err)
	}

	// 尝试添加 C，其 parentID 指向 B，形成正常层级（应通过）
	memberC := &mockMember{id: "C", name: "C", parentID: "B"}
	if err := team.AddMember(memberC); err != nil {
		t.Fatalf("failed to add member C: %v", err)
	}

	// 尝试重新添加 A，其 parentID 指向 C，会形成循环（应拒绝）
	memberACycle := &mockMember{id: "A2", name: "A2", parentID: "C"}
	// 先设置 parentID 为 A（这会尝试检测 A2 → C → B → A）
	memberACycle.parentID = "A" // 正常添加
	if err := team.AddMember(memberACycle); err != nil {
		t.Fatalf("failed to add member A2: %v", err)
	}
}

// TestAddMember_DepthLimit 集成测试：通过 AddMember 测试深度限制
func TestAddMember_DepthLimit(t *testing.T) {
	team := New()

	// 构建深度为 MaxMemberDepth 的结构
	var prevID string
	for i := 0; i < MaxMemberDepth; i++ {
		memberID := fmt.Sprintf("member%d", i)
		member := &mockMember{id: memberID, name: memberID, parentID: prevID}
		if err := team.AddMember(member); err != nil {
			t.Fatalf("failed to add member at depth %d: %v", i, err)
		}
		prevID = memberID
	}

	// 尝试添加超过深度限制的成员（应拒绝）
	tooDeepMember := &mockMember{id: "tooDeep", name: "tooDeep", parentID: prevID}
	err := team.AddMember(tooDeepMember)
	if err == nil {
		t.Fatal("expected depth limit error, got nil")
	}
}

// TestAddMember_DuplicateID 测试重复 ID 检测
func TestAddMember_DuplicateID(t *testing.T) {
	team := New()

	member1 := &mockMember{id: "A", name: "A", parentID: ""}
	if err := team.AddMember(member1); err != nil {
		t.Fatalf("failed to add first member: %v", err)
	}

	// 尝试添加相同 ID 的成员（应拒绝）
	member2 := &mockMember{id: "A", name: "A2", parentID: ""}
	err := team.AddMember(member2)
	if err == nil {
		t.Fatal("expected duplicate ID error, got nil")
	}
	if err.Error() != "member with ID A already exists" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestAddMember_ConcurrentAdd 测试并发添加成员
func TestAddMember_ConcurrentAdd(t *testing.T) {
	team := New()

	// 先添加根成员
	root := &mockMember{id: "root", name: "root", parentID: ""}
	if err := team.AddMember(root); err != nil {
		t.Fatalf("failed to add root: %v", err)
	}

	// 并发添加多个成员
	const numGoroutines = 50
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			member := &mockMember{
				id:       fmt.Sprintf("member%d", id),
				name:     fmt.Sprintf("member%d", id),
				parentID: "root",
			}
			if err := team.AddMember(member); err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// 检查是否有错误
	for err := range errors {
		t.Errorf("concurrent add failed: %v", err)
	}

	// 验证所有成员都被添加
	members := team.ListMembers()
	if len(members) != numGoroutines+1 { // +1 for root
		t.Errorf("expected %d members, got %d", numGoroutines+1, len(members))
	}
}
