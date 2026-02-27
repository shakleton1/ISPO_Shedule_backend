package schedule

import "fmt"

// scheduleInheritanceChainIDs returns group IDs in inheritance order (base -> leaf).
// Leaf is always the requested groupID.
func (s *Service) scheduleInheritanceChainIDs(groupID int) ([]int, error) {
	if groupID <= 0 {
		return nil, fmt.Errorf("group_id required")
	}
	leaf, err := s.repo.GetGroup(groupID)
	if err != nil {
		return nil, err
	}

	seen := map[int]bool{}
	idsRev := make([]int, 0, 2)
	cur := leaf
	for depth := 0; depth < 10; depth++ {
		if cur == nil {
			break
		}
		if seen[cur.ID] {
			return nil, fmt.Errorf("schedule_source_group_id cycle detected")
		}
		seen[cur.ID] = true
		idsRev = append(idsRev, cur.ID)
		if cur.ScheduleSourceGroupID == nil {
			break
		}
		next, err := s.repo.GetGroup(*cur.ScheduleSourceGroupID)
		if err != nil {
			return nil, err
		}
		cur = next
	}

	// Reverse to base -> leaf.
	ids := make([]int, 0, len(idsRev))
	for i := len(idsRev) - 1; i >= 0; i-- {
		ids = append(ids, idsRev[i])
	}
	return ids, nil
}
