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

// Exported handlers for integration tests

// HandleAdminCreateSpecialtyForTest exports handleAdminCreateSpecialty for integration tests
func HandleAdminCreateSpecialtyForTest(repo *schedule.Repository) gin.HandlerFunc {
	return handleAdminCreateSpecialty(repo)
}

// HandleAdminListSpecialtiesForTest exports handleAdminListSpecialties for integration tests
func HandleAdminListSpecialtiesForTest(repo *schedule.Repository) gin.HandlerFunc {
	return handleAdminListSpecialties(repo)
}

// HandleAdminUpdateSpecialtyForTest exports handleAdminUpdateSpecialty for integration tests
func HandleAdminUpdateSpecialtyForTest(repo *schedule.Repository) gin.HandlerFunc {
	return handleAdminUpdateSpecialty(repo)
}

// HandleAdminDeleteSpecialtyForTest exports handleAdminDeleteSpecialty for integration tests
func HandleAdminDeleteSpecialtyForTest(repo *schedule.Repository) gin.HandlerFunc {
	return handleAdminDeleteSpecialty(repo)
}

// Specialties

func handleAdminListSpecialties(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		var (
			rows []schedule.Specialty
			err  error
		)
		if p.Limit != nil {
			rows, err = repo.ListSpecialtiesPaged(p.Limit, p.Offset)
		} else {
			rows, err = repo.ListSpecialties()
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]specialtyDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toSpecialtyDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

func handleAdminCreateSpecialty(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schedule.Specialty
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		if req.Code == "" || req.Name == "" {
			writeValidationError(c, "", "code and name required")
			return
		}
		if err := repo.CreateSpecialty(&req); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "create", "specialties", strconv.Itoa(req.ID), req)
		c.JSON(http.StatusCreated, toSpecialtyDTO(req))
	}
}

func handleAdminUpdateSpecialty(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		var req schedule.Specialty
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		if req.Code == "" || req.Name == "" {
			writeValidationError(c, "", "code and name required")
			return
		}
		row, err := repo.UpdateSpecialty(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "update", "specialties", strconv.Itoa(id), row)
		c.JSON(http.StatusOK, toSpecialtyDTO(*row))
	}
}

func handleAdminDeleteSpecialty(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		if err := repo.DeleteSpecialty(id); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "delete", "specialties", strconv.Itoa(id), nil)
		c.Writer.WriteHeader(http.StatusNoContent)
	}
}

// Curricula

func handleAdminListCurricula(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		var filters schedule.CurriculumFilters
		if v := c.Query("specialty_id"); v != "" {
			id, err := strconv.Atoi(v)
			if err != nil {
				writeValidationError(c, "specialty_id", "invalid specialty_id")
				return
			}
			filters.SpecialtyID = &id
		}
		if v := c.Query("admission_year"); v != "" {
			y, err := strconv.Atoi(v)
			if err != nil {
				writeValidationError(c, "admission_year", "invalid admission_year")
				return
			}
			y16 := int16(y)
			filters.AdmissionYear = &y16
		}
		if v := c.Query("is_active"); v != "" {
			b, err := strconv.ParseBool(v)
			if err != nil {
				writeValidationError(c, "is_active", "invalid is_active")
				return
			}
			filters.IsActive = &b
		}
		var (
			rows []schedule.Curriculum
			err  error
		)
		if p.Limit != nil {
			rows, err = repo.ListCurriculaPaged(filters, p.Limit, p.Offset)
		} else {
			rows, err = repo.ListCurricula(filters)
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]curriculumDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toCurriculumDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

func handleAdminCreateCurriculum(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schedule.Curriculum
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		if req.SpecialtyID <= 0 || req.AdmissionYear <= 0 {
			writeValidationError(c, "", "specialty_id and admission_year required")
			return
		}
		if err := repo.CreateCurriculum(&req); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "create", "curricula", strconv.FormatInt(req.ID, 10), req)
		c.JSON(http.StatusCreated, toCurriculumDTO(req))
	}
}

func handleAdminUpdateCurriculum(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		var req schedule.Curriculum
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		if req.SpecialtyID <= 0 || req.AdmissionYear <= 0 {
			writeValidationError(c, "", "specialty_id and admission_year required")
			return
		}
		row, err := repo.UpdateCurriculum(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "update", "curricula", strconv.FormatInt(id, 10), row)
		c.JSON(http.StatusOK, toCurriculumDTO(*row))
	}
}

func handleAdminDeleteCurriculum(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		if err := repo.DeleteCurriculum(id); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "delete", "curricula", strconv.FormatInt(id, 10), nil)
		c.Writer.WriteHeader(http.StatusNoContent)
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
		p, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		currID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid curriculum id")
			return
		}
		var rows []schedule.AcademicCalendar
		if p.Limit != nil {
			rows, err = repo.ListAcademicCalendarsPaged(currID, p.Limit, p.Offset)
		} else {
			rows, err = repo.ListAcademicCalendars(currID)
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]academicCalendarDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toAcademicCalendarDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

func handleAdminCreateAcademicCalendar(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		currID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid curriculum id")
			return
		}
		var req createAcademicCalendarReq
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		if req.AcademicYearStart == "" {
			writeValidationError(c, "academic_year_start", "academic_year_start required")
			return
		}
		start, err := time.Parse("2006-01-02", req.AcademicYearStart)
		if err != nil {
			writeValidationError(c, "academic_year_start", "academic_year_start must be YYYY-MM-DD")
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
		c.JSON(http.StatusCreated, toAcademicCalendarDTO(row))
	}
}

func handleAdminDeleteAcademicCalendar(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		if err := repo.DeleteAcademicCalendar(id); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "delete", "academic_calendars", strconv.FormatInt(id, 10), nil)
		c.Writer.WriteHeader(http.StatusNoContent)
	}
}

func handleAdminListAcademicCalendarWeeks(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		var rows []schedule.AcademicCalendarWeek
		if p.Limit != nil {
			rows, err = repo.ListAcademicCalendarWeeksPaged(id, p.Limit, p.Offset)
		} else {
			rows, err = repo.ListAcademicCalendarWeeks(id)
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]academicCalendarWeekDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toAcademicCalendarWeekDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

func handleAdminUpsertAcademicCalendarWeeks(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		var req []schedule.AcademicCalendarWeek
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		rows, err := repo.UpsertAcademicCalendarWeeks(id, req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "upsert", "academic_calendar_weeks", strconv.FormatInt(id, 10), gin.H{"count": len(req)})
		out := make([]academicCalendarWeekDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toAcademicCalendarWeekDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

// Curriculum items + allocations

type curriculumItemTreeNode struct {
	curriculumItemDTO
	Children []*curriculumItemTreeNode `json:"children"`
}

func buildCurriculumItemTree(items []schedule.CurriculumItem) []*curriculumItemTreeNode {
	nodesByID := make(map[int64]*curriculumItemTreeNode, len(items))
	for i := range items {
		it := items[i]
		nodesByID[it.ID] = &curriculumItemTreeNode{
			curriculumItemDTO: toCurriculumItemDTO(it),
			Children:          []*curriculumItemTreeNode{},
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
		p, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		currID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid curriculum id")
			return
		}
		var rows []schedule.CurriculumItem
		if p.Limit != nil {
			rows, err = repo.ListCurriculumItemsPaged(currID, p.Limit, p.Offset)
		} else {
			rows, err = repo.ListCurriculumItems(currID)
		}
		if err != nil {
			writeDBError(c, err)
			return
		}

		tree := c.Query("tree")
		if tree == "true" || tree == "1" {
			c.JSON(http.StatusOK, buildCurriculumItemTree(rows))
			return
		}
		out := make([]curriculumItemDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toCurriculumItemDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

func handleAdminCreateCurriculumItem(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		currID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid curriculum id")
			return
		}
		var req schedule.CurriculumItem
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		req.CurriculumID = currID
		if req.ItemType == "" || req.Name == "" {
			writeValidationError(c, "", "item_type and name required")
			return
		}
		if err := repo.CreateCurriculumItem(&req); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "create", "curriculum_items", strconv.FormatInt(req.ID, 10), req)
		c.JSON(http.StatusCreated, toCurriculumItemDTO(req))
	}
}

func handleAdminUpdateCurriculumItem(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		var req schedule.CurriculumItem
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		if req.ItemType == "" || req.Name == "" {
			writeValidationError(c, "", "item_type and name required")
			return
		}
		row, err := repo.UpdateCurriculumItem(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "update", "curriculum_items", strconv.FormatInt(id, 10), row)
		c.JSON(http.StatusOK, toCurriculumItemDTO(*row))
	}
}

func handleAdminDeleteCurriculumItem(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		if err := repo.DeleteCurriculumItem(id); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "delete", "curriculum_items", strconv.FormatInt(id, 10), nil)
		c.Writer.WriteHeader(http.StatusNoContent)
	}
}

func handleAdminListCurriculumItemAllocations(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		var rows []schedule.CurriculumItemAllocation
		if p.Limit != nil {
			rows, err = repo.ListCurriculumItemAllocationsPaged(id, p.Limit, p.Offset)
		} else {
			rows, err = repo.ListCurriculumItemAllocations(id)
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]curriculumItemAllocationDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toCurriculumItemAllocationDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

func handleAdminUpsertCurriculumItemAllocations(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		var req []schedule.CurriculumItemAllocation
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		rows, err := repo.UpsertCurriculumItemAllocations(id, req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "upsert", "curriculum_item_allocations", strconv.FormatInt(id, 10), gin.H{"count": len(req)})
		out := make([]curriculumItemAllocationDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toCurriculumItemAllocationDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}
