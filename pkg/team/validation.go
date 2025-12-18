package team

import "fmt"

// MaxMemberDepth 最大成员层级深度
const MaxMemberDepth = 5

// detectCycle 使用 DFS 检测从 newMemberID 出发是否会形成循环
//
// 算法：从 parentID 向上遍历祖先链，如果遇到 newMemberID 则存在循环
// 时间复杂度：O(N)，N = 层级深度（最大为 MaxMemberDepth）
func detectCycle(members map[string]Member, newMemberID, parentID string) error {
	if parentID == "" {
		return nil // 根节点，无循环
	}

	visited := make(map[string]bool)
	currentID := parentID

	// 向上遍历祖先链
	for currentID != "" {
		// 检测到循环
		if currentID == newMemberID {
			return fmt.Errorf("adding member %s with parent %s would create a cycle", newMemberID, parentID)
		}

		// 防止异常情况下的无限循环
		if visited[currentID] {
			return fmt.Errorf("detected existing cycle in member hierarchy at %s", currentID)
		}
		visited[currentID] = true

		// 父成员不存在，结束遍历
		parent, ok := members[currentID]
		if !ok {
			return fmt.Errorf("parent member %s not found", currentID)
		}

		currentID = parent.ParentID()
	}

	return nil
}

// checkDepthLimit 检查添加新成员后是否超过深度限制
func checkDepthLimit(members map[string]Member, parentID string, maxDepth int) error {
	depth := calculateDepth(members, parentID)
	// 新成员的深度会是 depth + 1
	if depth+1 > maxDepth {
		return fmt.Errorf("hierarchy depth %d would exceed maximum %d", depth+1, maxDepth)
	}
	return nil
}

// calculateDepth 计算从 memberID 到根节点的深度
func calculateDepth(members map[string]Member, memberID string) int {
	if memberID == "" {
		return 0
	}

	depth := 0
	currentID := memberID
	visited := make(map[string]bool)

	// 向上遍历，限制最大迭代次数防止异常
	for currentID != "" && depth <= MaxMemberDepth {
		if visited[currentID] {
			// 检测到循环，返回当前深度
			return depth
		}
		visited[currentID] = true

		member, ok := members[currentID]
		if !ok {
			// 父成员不存在
			return depth
		}

		currentID = member.ParentID()
		depth++
	}

	return depth
}
