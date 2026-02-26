package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"ispo-schedule/internal/auth"
	"ispo-schedule/internal/config"
	"ispo-schedule/internal/db"
	"ispo-schedule/internal/obs"
	"ispo-schedule/internal/schedule"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to config yaml (optional). If empty, uses configs/config.yaml then configs/config.example.yaml")
	flag.Parse()

	cfg, err := loadConfig(configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("load config")
	}

	obs.InitLogger(cfg.Log)

	gormDB, err := db.Open(cfg.DB)
	if err != nil {
		log.Fatal().Err(err).Msg("open db")
	}
	repo := schedule.NewRepository(gormDB)

	res, err := seedMinimal(repo)
	if err != nil {
		log.Fatal().Err(err).Msg("seed")
	}

	fmt.Println("Seed completed.")
	fmt.Printf("- specialty_id=%d curriculum_id=%d calendar_id=%d group_id=%d\n", res.SpecialtyID, res.CurriculumID, res.CalendarID, res.GroupID)
	fmt.Println("- users:")
	for _, u := range res.Users {
		fmt.Printf("  - %s (%s)\n", u.Login, u.Role)
	}
	fmt.Println("\nNote: default passwords are for dev only; override with env vars ISPO_SEED_*.")
}

type seedResult struct {
	SpecialtyID  int
	CurriculumID int64
	CalendarID   int64
	GroupID      int
	Users        []auth.User
}

func loadConfig(configPath string) (*config.Config, error) {
	if configPath != "" {
		return config.Load(config.LoadOptions{ConfigPath: configPath})
	}
	cfg, err := config.Load(config.LoadOptions{ConfigPath: "configs/config.yaml"})
	if err == nil {
		return cfg, nil
	}
	return config.Load(config.LoadOptions{ConfigPath: "configs/config.example.yaml"})
}

func seedMinimal(repo *schedule.Repository) (*seedResult, error) {
	// ----------- Dictionaries
	mathID, err := getOrCreateSubject(repo.DB(), schedule.Subject{Name: "Математика", ShortName: "Матем"})
	if err != nil {
		return nil, err
	}
	infID, err := getOrCreateSubject(repo.DB(), schedule.Subject{Name: "Информатика", ShortName: "Инф"})
	if err != nil {
		return nil, err
	}
	_, _ = infID, mathID

	loc101ID, err := getOrCreateLocation(repo.DB(), schedule.Location{Name: "Каб. 101", IsVirtual: false})
	if err != nil {
		return nil, err
	}
	locOnlineID, err := getOrCreateLocation(repo.DB(), schedule.Location{Name: "Дистанционно", IsVirtual: true})
	if err != nil {
		return nil, err
	}
	_, _ = loc101ID, locOnlineID

	// ----------- Curriculum
	specID, err := getOrCreateSpecialty(repo.DB(), schedule.Specialty{Code: "09.02.07", Name: "Информационные системы и программирование"})
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	admissionYear := defaultAdmissionYear(now)
	currID, err := getOrCreateCurriculum(repo.DB(), schedule.Curriculum{
		SpecialtyID:   specID,
		AdmissionYear: admissionYear,
		Variant:       "Базовый",
		Title:         "Учебный план (seed)",
		IsActive:      true,
	})
	if err != nil {
		return nil, err
	}

	acYearStart := time.Date(int(admissionYear), 9, 1, 0, 0, 0, 0, time.UTC)
	calID, err := getOrCreateAcademicCalendar(repo.DB(), schedule.AcademicCalendar{
		CurriculumID:      currID,
		AcademicYearStart: acYearStart,
		WeeksTotal:        52,
		Notes:             ptrString("Seed calendar")},
	)
	if err != nil {
		return nil, err
	}

	weeks := make([]schedule.AcademicCalendarWeek, 0, 4)
	for i := int16(1); i <= 4; i++ {
		start := acYearStart.AddDate(0, 0, int((i-1)*7))
		weeks = append(weeks, schedule.AcademicCalendarWeek{
			WeekNumber:    i,
			WeekStartDate: start,
			ActivityCode:  "EDU",
			ActivityName:  ptrString("Учебная неделя"),
			IsTeaching:    true,
		})
	}
	if _, err := repo.UpsertAcademicCalendarWeeks(calID, weeks); err != nil {
		return nil, err
	}

	// ----------- Group
	groupName := "ИСПО-11-1"
	groupID, err := getOrCreateGroup(repo.DB(), schedule.Group{
		Name:          groupName,
		Course:        1,
		CurriculumID:  &currID,
		AdmissionYear: &admissionYear,
		SpecialtyID:   &specID,
	})
	if err != nil {
		return nil, err
	}

	// ----------- Minimal curriculum items
	rootID, err := getOrCreateCurriculumItem(repo.DB(), schedule.CurriculumItem{
		CurriculumID: currID,
		ParentID:     nil,
		IndexCode:    ptrString("1"),
		ItemType:     "OTHER",
		Name:         "Обязательная часть",
		SubjectID:    nil,
	})
	if err != nil {
		return nil, err
	}

	mathItemID, err := getOrCreateCurriculumItem(repo.DB(), schedule.CurriculumItem{
		CurriculumID: currID,
		ParentID:     &rootID,
		IndexCode:    ptrString("1.1"),
		ItemType:     "DISCIPLINE",
		Name:         "Математика",
		SubjectID:    &mathID,
	})
	if err != nil {
		return nil, err
	}

	allocs := []schedule.CurriculumItemAllocation{
		{
			Semester:       1,
			Weeks:          ptrI16(16),
			HoursTotal:     ptrInt(64),
			HoursLectures:  ptrInt(32),
			HoursPractice:  ptrInt(32),
			AssessmentType: ptrString("EXAM"),
		},
	}
	if _, err := repo.UpsertCurriculumItemAllocations(mathItemID, allocs); err != nil {
		return nil, err
	}

	// ----------- Users
	if _, err := repo.GetSystemState(); err != nil {
		return nil, err
	}

	users, err := seedUsers(repo, groupID)
	if err != nil {
		return nil, err
	}

	return &seedResult{
		SpecialtyID:  specID,
		CurriculumID: currID,
		CalendarID:   calID,
		GroupID:      groupID,
		Users:        users,
	}, nil
}

func seedUsers(repo *schedule.Repository, groupID int) ([]auth.User, error) {
	adminLogin := firstNonEmpty(os.Getenv("ISPO_SEED_ADMIN_LOGIN"), "admin")
	adminPass := firstNonEmpty(os.Getenv("ISPO_SEED_ADMIN_PASSWORD"), "admin")
	dispatcherLogin := firstNonEmpty(os.Getenv("ISPO_SEED_DISPATCHER_LOGIN"), "dispatcher")
	dispatcherPass := firstNonEmpty(os.Getenv("ISPO_SEED_DISPATCHER_PASSWORD"), "dispatcher")
	studentLogin := firstNonEmpty(os.Getenv("ISPO_SEED_STUDENT_LOGIN"), "student1")
	studentPass := firstNonEmpty(os.Getenv("ISPO_SEED_STUDENT_PASSWORD"), "student1")
	viewerLogin := firstNonEmpty(os.Getenv("ISPO_SEED_VIEWER_LOGIN"), "viewer")
	viewerPass := firstNonEmpty(os.Getenv("ISPO_SEED_VIEWER_PASSWORD"), "viewer")

	seed := []struct {
		login    string
		password string
		role     auth.Role
		groupID  *int
		subgroup *int16
	}{
		{login: adminLogin, password: adminPass, role: auth.RoleAdmin},
		{login: dispatcherLogin, password: dispatcherPass, role: auth.RoleDispatcher},
		{login: viewerLogin, password: viewerPass, role: auth.RoleViewer},
		{login: studentLogin, password: studentPass, role: auth.RoleStudent, groupID: &groupID, subgroup: ptrI16(1)},
	}

	out := make([]auth.User, 0, len(seed))
	for _, s := range seed {
		u, err := ensureUser(repo, s.login, s.password, s.role, s.groupID, s.subgroup)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, nil
}

func ensureUser(repo *schedule.Repository, login, password string, role auth.Role, groupID *int, subgroup *int16) (*auth.User, error) {
	u, err := repo.GetUserByLogin(login)
	if err == nil {
		return u, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	h, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}

	u = &auth.User{
		Login:        login,
		PasswordHash: h,
		Role:         role,
		GroupID:      groupID,
		Subgroup:     subgroup,
	}
	if err := repo.CreateUser(u); err != nil {
		return nil, err
	}
	return u, nil
}

func getOrCreateSubject(db *gorm.DB, s schedule.Subject) (int, error) {
	var row schedule.Subject
	if err := db.Where("name = ?", s.Name).First(&row).Error; err == nil {
		return row.ID, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	if err := db.Create(&s).Error; err != nil {
		return 0, err
	}
	return s.ID, nil
}

func getOrCreateLocation(db *gorm.DB, l schedule.Location) (int, error) {
	var row schedule.Location
	if err := db.Where("name = ?", l.Name).First(&row).Error; err == nil {
		return row.ID, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	if err := db.Create(&l).Error; err != nil {
		return 0, err
	}
	return l.ID, nil
}

func getOrCreateSpecialty(db *gorm.DB, s schedule.Specialty) (int, error) {
	var row schedule.Specialty
	if err := db.Where("code = ?", s.Code).First(&row).Error; err == nil {
		return row.ID, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	if err := db.Create(&s).Error; err != nil {
		return 0, err
	}
	return s.ID, nil
}

func getOrCreateCurriculum(db *gorm.DB, c schedule.Curriculum) (int64, error) {
	var row schedule.Curriculum
	if err := db.Where("specialty_id = ? AND admission_year = ? AND variant = ?", c.SpecialtyID, c.AdmissionYear, c.Variant).First(&row).Error; err == nil {
		return row.ID, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	if err := db.Create(&c).Error; err != nil {
		return 0, err
	}
	return c.ID, nil
}

func getOrCreateAcademicCalendar(db *gorm.DB, ac schedule.AcademicCalendar) (int64, error) {
	var row schedule.AcademicCalendar
	if err := db.Where("curriculum_id = ? AND academic_year_start = ?", ac.CurriculumID, dateOnly(ac.AcademicYearStart)).First(&row).Error; err == nil {
		return row.ID, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	ac.AcademicYearStart = dateOnly(ac.AcademicYearStart)
	if err := db.Create(&ac).Error; err != nil {
		return 0, err
	}
	return ac.ID, nil
}

func getOrCreateGroup(db *gorm.DB, g schedule.Group) (int, error) {
	var row schedule.Group
	if err := db.Where("name = ?", g.Name).First(&row).Error; err == nil {
		// Keep it simple: patch key link fields on re-run.
		row.Course = g.Course
		row.CurriculumID = g.CurriculumID
		row.AdmissionYear = g.AdmissionYear
		row.SpecialtyID = g.SpecialtyID
		if err := db.Save(&row).Error; err != nil {
			return 0, err
		}
		return row.ID, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	if err := db.Create(&g).Error; err != nil {
		return 0, err
	}
	return g.ID, nil
}

func getOrCreateCurriculumItem(db *gorm.DB, it schedule.CurriculumItem) (int64, error) {
	q := db.Where("curriculum_id = ? AND item_type = ? AND name = ?", it.CurriculumID, it.ItemType, it.Name)
	if it.ParentID == nil {
		q = q.Where("parent_id IS NULL")
	} else {
		q = q.Where("parent_id = ?", *it.ParentID)
	}
	var row schedule.CurriculumItem
	if err := q.First(&row).Error; err == nil {
		// Patch index/subject links on re-run.
		row.IndexCode = it.IndexCode
		row.SubjectID = it.SubjectID
		if err := db.Save(&row).Error; err != nil {
			return 0, err
		}
		return row.ID, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	if err := db.Create(&it).Error; err != nil {
		return 0, err
	}
	return it.ID, nil
}

func defaultAdmissionYear(now time.Time) int16 {
	// If we are before September, admission year is previous year.
	if now.Month() < time.September {
		return int16(now.Year() - 1)
	}
	return int16(now.Year())
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func ptrString(v string) *string { return &v }
func ptrInt(v int) *int         { return &v }
func ptrI16(v int16) *int16     { return &v }

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
