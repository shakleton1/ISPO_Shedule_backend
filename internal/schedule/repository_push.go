package schedule

import "fmt"

func (r *Repository) UpsertDeviceToken(groupID int, token string) error {
	if groupID <= 0 {
		return fmt.Errorf("group_id required")
	}
	token = normalizeToken(token)
	if token == "" {
		return fmt.Errorf("token required")
	}
	return r.db.Exec(
		`INSERT INTO device_tokens (group_id, token) VALUES (?, ?)
		 ON CONFLICT (token) DO UPDATE SET group_id = EXCLUDED.group_id, updated_at = now()`,
		groupID, token,
	).Error
}

func (r *Repository) DeleteDeviceToken(token string) error {
	token = normalizeToken(token)
	if token == "" {
		return fmt.Errorf("token required")
	}
	return r.db.Where("token = ?", token).Delete(&DeviceToken{}).Error
}

func (r *Repository) ListDeviceTokensByGroup(groupID int) ([]DeviceToken, error) {
	var rows []DeviceToken
	err := r.db.Where("group_id = ?", groupID).Order("id asc").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListSubscribedGroupIDs() ([]int, error) {
	var ids []int
	err := r.db.Model(&DeviceToken{}).Distinct("group_id").Order("group_id asc").Pluck("group_id", &ids).Error
	return ids, err
}

func normalizeToken(s string) string {
	// tokens are opaque; just trim spaces and forbid huge values.
	for len(s) > 0 {
		c := s[0]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			break
		}
		s = s[1:]
	}
	for len(s) > 0 {
		c := s[len(s)-1]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			break
		}
		s = s[:len(s)-1]
	}
	if len(s) > 4096 {
		return ""
	}
	return s
}
