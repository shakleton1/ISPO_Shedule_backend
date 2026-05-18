package schedule

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ispo-schedule/internal/auth"

	"gorm.io/gorm"
)

func normalizeTeacherName(name string) string {
	return strings.TrimSpace(name)
}

func ensureTeacherID(tx *gorm.DB, name string) (*int, error) {
	name = normalizeTeacherName(name)
	if name == "" {
		return nil, nil
	}
	var out struct {
		ID int `gorm:"column:id"`
	}
	err := tx.Raw(
		"INSERT INTO teachers (name) VALUES (?) ON CONFLICT (name_key) DO UPDATE SET name = EXCLUDED.name, deleted_at = NULL RETURNING id",
		name,
	).Scan(&out).Error
	if err != nil {
		return nil, err
	}
	return &out.ID, nil
}

// Groups

func (r *Repository) ListGroups() ([]Group, error) {
	var rows []Group
	err := r.db.Order("id asc").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListGroupsPaged(limit, offset *int) ([]Group, error) {
	var rows []Group
	q := r.db.Order("id asc")
	q = applyLimitOffset(q, limit, offset)
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) GetGroup(id int) (*Group, error) {
	var row Group
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) CreateGroup(g *Group) error {
	return r.db.Create(g).Error
}

func (r *Repository) UpdateGroup(id int, patch *Group) (*Group, error) {
	var row Group
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.Name = patch.Name
	row.Course = patch.Course
	if patch.ScheduleSourceGroupID != nil {
		if *patch.ScheduleSourceGroupID == row.ID {
			return nil, fmt.Errorf("schedule_source_group_id must not point to self")
		}
		// Cycle check: follow schedule_source_group_id pointers.
		seen := map[int]bool{row.ID: true}
		next := *patch.ScheduleSourceGroupID
		for depth := 0; depth < 10; depth++ {
			if next <= 0 {
				break
			}
			if seen[next] {
				return nil, fmt.Errorf("schedule_source_group_id cycle detected")
			}
			seen[next] = true
			var g Group
			if err := r.db.First(&g, next).Error; err != nil {
				return nil, err
			}
			if g.ScheduleSourceGroupID == nil {
				break
			}
			next = *g.ScheduleSourceGroupID
		}
		row.ScheduleSourceGroupID = patch.ScheduleSourceGroupID
	}
	if patch.CurriculumID != nil {
		row.CurriculumID = patch.CurriculumID
	}
	if patch.AdmissionYear != nil {
		row.AdmissionYear = patch.AdmissionYear
	}
	if patch.SpecialtyID != nil {
		row.SpecialtyID = patch.SpecialtyID
	}
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteGroup(id int) error {
	return r.db.Delete(&Group{}, id).Error
}

// Subjects

func (r *Repository) ListSubjects() ([]Subject, error) {
	var rows []Subject
	err := r.db.Where("deleted_at IS NULL").Order("id asc").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListSubjectsPaged(limit, offset *int) ([]Subject, error) {
	var rows []Subject
	q := r.db.Where("deleted_at IS NULL").Order("id asc")
	q = applyLimitOffset(q, limit, offset)
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateSubject(s *Subject) error {
	return r.db.Create(s).Error
}

func (r *Repository) UpdateSubject(id int, patch *Subject) (*Subject, error) {
	var row Subject
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.Name = patch.Name
	row.ShortName = patch.ShortName
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	// If it was soft-deleted earlier, restore.
	if err := r.db.Exec("UPDATE subjects SET deleted_at = NULL WHERE id = ?", id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteSubject(id int) error {
	res := r.db.Exec("UPDATE subjects SET deleted_at = now() WHERE id = ? AND deleted_at IS NULL", id)
	return res.Error
}

// Locations

func (r *Repository) ListLocations() ([]Location, error) {
	var rows []Location
	err := r.db.Order("id asc").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListLocationsPaged(limit, offset *int) ([]Location, error) {
	var rows []Location
	q := r.db.Order("id asc")
	q = applyLimitOffset(q, limit, offset)
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) GetLocation(id int) (*Location, error) {
	var row Location
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) CreateLocation(l *Location) error {
	if err := normalizeLocation(l); err != nil {
		return err
	}
	return r.db.Create(l).Error
}

func (r *Repository) UpdateLocation(id int, patch *Location) (*Location, error) {
	if err := normalizeLocation(patch); err != nil {
		return nil, err
	}
	var row Location
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.Name = patch.Name
	row.CampusID = patch.CampusID
	row.Kind = patch.Kind
	row.Capacity = patch.Capacity
	row.IsActive = patch.IsActive
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteLocation(id int) error {
	return r.db.Delete(&Location{}, id).Error
}

// Overlay

func (r *Repository) UpsertOverlay(groupID int, date time.Time, text string, stylePreset string) (*ScheduleDayOverlay, error) {
	date = dateOnly(date)
	if stylePreset == "" {
		stylePreset = "standard"
	}

	var row ScheduleDayOverlay
	err := r.db.Where("group_id = ? AND target_date = ?", groupID, date).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			row = ScheduleDayOverlay{GroupID: groupID, TargetDate: date, Text: text, StylePreset: stylePreset}
			if err2 := r.db.Create(&row).Error; err2 != nil {
				return nil, err2
			}
			return &row, nil
		}
		return nil, err
	}
	row.Text = text
	row.StylePreset = stylePreset
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func ParseInt64Param(v string) (int64, error) {
	return strconv.ParseInt(v, 10, 64)
}

// Users (auth)

func (r *Repository) GetUserByLogin(login string) (*auth.User, error) {
	var u auth.User
	if err := r.db.Where("login = ?", login).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) GetUserByID(id int64) (*auth.User, error) {
	var u auth.User
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) CreateUser(u *auth.User) error {
	return r.db.Create(u).Error
}

func (r *Repository) UpdateUser(u *auth.User) error {
	return r.db.Save(u).Error
}

func (r *Repository) CountAdmins() (int64, error) {
	var cnt int64
	err := r.db.Model(&auth.User{}).Where("role = ?", auth.RoleAdmin).Count(&cnt).Error
	return cnt, err
}
