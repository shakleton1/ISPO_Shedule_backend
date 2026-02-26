package httpapi

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ispo-schedule/internal/schedule"
)

// Specialties

func handleAdminListSpecialties(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := repo.ListSpecialties()
		if err != nil {
			writeDBError(c, err)
			return
		}
		c.JSON(http.StatusOK, rows)
	}
}

func handleAdminCreateSpecialty(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schedule.Specialty
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if req.Code == "" || req.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "code and name required"})
			return
		}
		if err := repo.CreateSpecialty(&req); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "create", "specialties", strconv.Itoa(req.ID), req)
		c.JSON(http.StatusCreated, req)
	}
}

func handleAdminUpdateSpecialty(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req schedule.Specialty
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if req.Code == "" || req.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "code and name required"})
			return
		}
		row, err := repo.UpdateSpecialty(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "update", "specialties", strconv.Itoa(id), row)
		c.JSON(http.StatusOK, row)
	}
}

func handleAdminDeleteSpecialty(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := repo.DeleteSpecialty(id); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "delete", "specialties", strconv.Itoa(id), nil)
		c.Status(http.StatusNoContent)
	}
}

// Curricula

func handleAdminListCurricula(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var filters schedule.CurriculumFilters
		if v := c.Query("specialty_id"); v != "" {
			id, err := strconv.Atoi(v)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid specialty_id"})
				return
			}
			filters.SpecialtyID = &id
		}
		if v := c.Query("admission_year"); v != "" {
			y, err := strconv.Atoi(v)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid admission_year"})
				return
			}
			y16 := int16(y)
			filters.AdmissionYear = &y16
		}
		if v := c.Query("is_active"); v != "" {
			b, err := strconv.ParseBool(v)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid is_active"})
				return
			}
			filters.IsActive = &b
		}

		rows, err := repo.ListCurricula(filters)
		if err != nil {
			writeDBError(c, err)
			return
		}
		c.JSON(http.StatusOK, rows)
	}
}

func handleAdminCreateCurriculum(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schedule.Curriculum
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if req.SpecialtyID <= 0 || req.AdmissionYear <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "specialty_id and admission_year required"})
			return
		}
		if err := repo.CreateCurriculum(&req); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "create", "curricula", strconv.FormatInt(req.ID, 10), req)
		c.JSON(http.StatusCreated, req)
	}
}

func handleAdminUpdateCurriculum(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req schedule.Curriculum
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if req.SpecialtyID <= 0 || req.AdmissionYear <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "specialty_id and admission_year required"})
			return
		}
		row, err := repo.UpdateCurriculum(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "update", "curricula", strconv.FormatInt(id, 10), row)
		c.JSON(http.StatusOK, row)
	}
}

func handleAdminDeleteCurriculum(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := repo.DeleteCurriculum(id); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "delete", "curricula", strconv.FormatInt(id, 10), nil)
		c.Status(http.StatusNoContent)
	}
}

// Academic calendars

type createAcademicCalendarReq struct {
	AcademicYearStart string  `json:"academic_year_start"`
	WeeksTotal        *int16  `json:"weeks_total"`
	Notes             *string `json:"notes"`
}

func handleAdminListAcademicCalendars(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		currID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid curriculum id"})
			return
		}
		rows, err := repo.ListAcademicCalendars(currID)
		if err != nil {
			writeDBError(c, err)
			return
		}
		c.JSON(http.StatusOK, rows)
	}
}

func handleAdminCreateAcademicCalendar(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		currID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid curriculum id"})
			return
		}
		var req createAcademicCalendarReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if req.AcademicYearStart == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "academic_year_start required"})
			return
		}
		start, err := time.Parse("2006-01-02", req.AcademicYearStart)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "academic_year_start must be YYYY-MM-DD"})
			return
		}
		weeksTotal := int16(52)
		if req.WeeksTotal != nil {
			weeksTotal = *req.WeeksTotal
		}
		row := schedule.AcademicCalendar{CurriculumID: currID, AcademicYearStart: start, WeeksTotal: weeksTotal, Notes: req.Notes}
		if err := repo.CreateAcademicCalendar(&row); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "create", "academic_calendars", strconv.FormatInt(row.ID, 10), row)
		c.JSON(http.StatusCreated, row)
	}
}

func handleAdminDeleteAcademicCalendar(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := repo.DeleteAcademicCalendar(id); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "delete", "academic_calendars", strconv.FormatInt(id, 10), nil)
		c.Status(http.StatusNoContent)
	}
}

func handleAdminListAcademicCalendarWeeks(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		rows, err := repo.ListAcademicCalendarWeeks(id)
		if err != nil {
			writeDBError(c, err)
			return
		}
		c.JSON(http.StatusOK, rows)
	}
}

func handleAdminUpsertAcademicCalendarWeeks(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req []schedule.AcademicCalendarWeek
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		rows, err := repo.UpsertAcademicCalendarWeeks(id, req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "upsert", "academic_calendar_weeks", strconv.FormatInt(id, 10), gin.H{"count": len(req)})
		c.JSON(http.StatusOK, rows)
	}
}

// Curriculum items + allocations

type curriculumItemTreeNode struct {
	schedule.CurriculumItem
	Children []*curriculumItemTreeNode `json:"children"`
}

func buildCurriculumItemTree(items []schedule.CurriculumItem) []*curriculumItemTreeNode {
	nodesByID := make(map[int64]*curriculumItemTreeNode, len(items))
	for i := range items {
		it := items[i]
		nodesByID[it.ID] = &curriculumItemTreeNode{
			CurriculumItem: it,
			Children:       []*curriculumItemTreeNode{},
		}
	}

	roots := []*curriculumItemTreeNode{}
	for _, node := range nodesByID {
		if node.ParentID == nil {
			roots = append(roots, node)
			continue
		}
		if *node.ParentID == node.ID {
			roots = append(roots, node)
			continue
		}
		parent, ok := nodesByID[*node.ParentID]
		if !ok {
			roots = append(roots, node)
			continue
		}
		parent.Children = append(parent.Children, node)
	}

	less := func(a, b *curriculumItemTreeNode) bool {
		aHasCode := a.IndexCode != nil && strings.TrimSpace(*a.IndexCode) != ""
		bHasCode := b.IndexCode != nil && strings.TrimSpace(*b.IndexCode) != ""
		if aHasCode != bHasCode {
			return aHasCode
		}
		aCode := ""
		bCode := ""
		if a.IndexCode != nil {
			aCode = strings.TrimSpace(*a.IndexCode)
		}
		if b.IndexCode != nil {
			bCode = strings.TrimSpace(*b.IndexCode)
		}
		if cmp := strings.Compare(aCode, bCode); cmp != 0 {
			return cmp < 0
		}
		if cmp := strings.Compare(a.Name, b.Name); cmp != 0 {
			return cmp < 0
		}
		return a.ID < b.ID
	}

	var sortRec func(nodes []*curriculumItemTreeNode)
	sortRec = func(nodes []*curriculumItemTreeNode) {
		sort.SliceStable(nodes, func(i, j int) bool { return less(nodes[i], nodes[j]) })
		for _, n := range nodes {
			sortRec(n.Children)
		}
	}

	sortRec(roots)
	return roots
}

func handleAdminListCurriculumItems(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		currID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid curriculum id"})
			return
		}
		rows, err := repo.ListCurriculumItems(currID)
		if err != nil {
			writeDBError(c, err)
			return
		}

		tree := c.Query("tree")
		if tree == "true" || tree == "1" {
			c.JSON(http.StatusOK, buildCurriculumItemTree(rows))
			return
		}
		c.JSON(http.StatusOK, rows)
	}
}

func handleAdminCreateCurriculumItem(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		currID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid curriculum id"})
			return
		}
		var req schedule.CurriculumItem
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		req.CurriculumID = currID
		if req.ItemType == "" || req.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "item_type and name required"})
			return
		}
		if err := repo.CreateCurriculumItem(&req); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "create", "curriculum_items", strconv.FormatInt(req.ID, 10), req)
		c.JSON(http.StatusCreated, req)
	}
}

func handleAdminUpdateCurriculumItem(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req schedule.CurriculumItem
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if req.ItemType == "" || req.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "item_type and name required"})
			return
		}
		row, err := repo.UpdateCurriculumItem(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "update", "curriculum_items", strconv.FormatInt(id, 10), row)
		c.JSON(http.StatusOK, row)
	}
}

func handleAdminDeleteCurriculumItem(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := repo.DeleteCurriculumItem(id); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "delete", "curriculum_items", strconv.FormatInt(id, 10), nil)
		c.Status(http.StatusNoContent)
	}
}

func handleAdminListCurriculumItemAllocations(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		rows, err := repo.ListCurriculumItemAllocations(id)
		if err != nil {
			writeDBError(c, err)
			return
		}
		c.JSON(http.StatusOK, rows)
	}
}

func handleAdminUpsertCurriculumItemAllocations(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req []schedule.CurriculumItemAllocation
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		rows, err := repo.UpsertCurriculumItemAllocations(id, req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "upsert", "curriculum_item_allocations", strconv.FormatInt(id, 10), gin.H{"count": len(req)})
		c.JSON(http.StatusOK, rows)
	}
}
