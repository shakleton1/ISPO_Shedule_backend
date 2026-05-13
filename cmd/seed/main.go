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
	fmt.Printf("- specialty_id=%d curriculum_id=%d calendar_id=%d group_ids=%v\n", res.SpecialtyID, res.CurriculumID, res.CalendarID, res.GroupIDs)
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
	GroupIDs     []int
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
	db := repo.DB()

	mathID, err := getOrCreateSubject(db, schedule.Subject{Name: "Математика", ShortName: "Матем"})
	if err != nil {
		return nil, err
	}
	infID, err := getOrCreateSubject(db, schedule.Subject{Name: "Информатика", ShortName: "Инф"})
	if err != nil {
		return nil, err
	}
	englishID, err := getOrCreateSubject(db, schedule.Subject{Name: "Иностранный язык", ShortName: "Ин. яз."})
	if err != nil {
		return nil, err
	}
	peID, err := getOrCreateSubject(db, schedule.Subject{Name: "Физическая культура", ShortName: "Физра"})
	if err != nil {
		return nil, err
	}
	practiceID, err := getOrCreateSubject(db, schedule.Subject{Name: "Учебная практика", ShortName: "УП"})
	if err != nil {
		return nil, err
	}

	campusID, err := getOrCreateCampus(db, schedule.Campus{Name: "Приморская", Address: ptrString("Приморская площадка")})
	if err != nil {
		return nil, err
	}
	classroomTypeID, err := getOrCreateLocationType(db, schedule.LocationType{Code: "classroom", Name: "Обычная аудитория"})
	if err != nil {
		return nil, err
	}
	computerTypeID, err := getOrCreateLocationType(db, schedule.LocationType{Code: "computer_class", Name: "ВЦ / компьютерный класс"})
	if err != nil {
		return nil, err
	}
	gymTypeID, err := getOrCreateLocationType(db, schedule.LocationType{Code: "gym", Name: "Спортивный зал"})
	if err != nil {
		return nil, err
	}
	poolTypeID, err := getOrCreateLocationType(db, schedule.LocationType{Code: "pool", Name: "Бассейн"})
	if err != nil {
		return nil, err
	}
	onlineTypeID, err := getOrCreateLocationType(db, schedule.LocationType{Code: "online", Name: "Онлайн / дистант"})
	if err != nil {
		return nil, err
	}

	loc101ID, err := getOrCreateLocation(db, schedule.Location{Name: "Каб. 101", CampusID: &campusID, Kind: "physical", IsActive: true, Capacity: ptrI16(32)})
	if err != nil {
		return nil, err
	}
	loc102ID, err := getOrCreateLocation(db, schedule.Location{Name: "Каб. 102", CampusID: &campusID, Kind: "physical", IsActive: true, Capacity: ptrI16(32)})
	if err != nil {
		return nil, err
	}
	locComputerID, err := getOrCreateLocation(db, schedule.Location{Name: "ВЦ-1", CampusID: &campusID, Kind: "physical", IsActive: true, Capacity: ptrI16(30)})
	if err != nil {
		return nil, err
	}
	locGymID, err := getOrCreateLocation(db, schedule.Location{Name: "Зал", CampusID: &campusID, Kind: "physical", IsActive: true, Capacity: ptrI16(90)})
	if err != nil {
		return nil, err
	}
	locPoolID, err := getOrCreateLocation(db, schedule.Location{Name: "Бассейн", CampusID: &campusID, Kind: "physical", IsActive: true, Capacity: ptrI16(45)})
	if err != nil {
		return nil, err
	}
	locOnlineID, err := getOrCreateLocation(db, schedule.Location{Name: "Дистант", Kind: "virtual", IsActive: true})
	if err != nil {
		return nil, err
	}
	for _, link := range []struct{ locationID, typeID int }{
		{loc101ID, classroomTypeID},
		{loc102ID, classroomTypeID},
		{locComputerID, computerTypeID},
		{locGymID, gymTypeID},
		{locPoolID, poolTypeID},
		{locOnlineID, onlineTypeID},
	} {
		if err := ensureLocationTypeLink(db, link.locationID, link.typeID); err != nil {
			return nil, err
		}
	}

	mathTeacherID, err := getOrCreateTeacher(db, schedule.Teacher{Name: "Иванов И.И."})
	if err != nil {
		return nil, err
	}
	infTeacherID, err := getOrCreateTeacher(db, schedule.Teacher{Name: "Петров П.П."})
	if err != nil {
		return nil, err
	}
	englishTeacherID, err := getOrCreateTeacher(db, schedule.Teacher{Name: "Сидорова А.А."})
	if err != nil {
		return nil, err
	}
	peTeacherID, err := getOrCreateTeacher(db, schedule.Teacher{Name: "Кузнецов К.К."})
	if err != nil {
		return nil, err
	}
	teacherIDs := []int{mathTeacherID, infTeacherID, englishTeacherID, peTeacherID}
	for i := 5; i <= 80; i++ {
		id, err := getOrCreateTeacher(db, schedule.Teacher{Name: fmt.Sprintf("Тестовый преподаватель %02d", i)})
		if err != nil {
			return nil, err
		}
		teacherIDs = append(teacherIDs, id)
	}
	onlineTeacherIDs := teacherIDs[30:33]
	onlineTeacherID := onlineTeacherIDs[0]

	for _, ts := range []schedule.TeacherSubject{
		{TeacherID: mathTeacherID, SubjectID: mathID},
		{TeacherID: infTeacherID, SubjectID: infID},
		{TeacherID: englishTeacherID, SubjectID: englishID},
		{TeacherID: peTeacherID, SubjectID: peID},
		{TeacherID: infTeacherID, SubjectID: practiceID},
	} {
		if err := repo.CreateTeacherSubject(&ts); err != nil {
			return nil, err
		}
	}
	seedSubjects := []int{mathID, infID, englishID, peID, practiceID}
	for i, teacherID := range teacherIDs {
		if err := repo.CreateTeacherSubject(&schedule.TeacherSubject{TeacherID: teacherID, SubjectID: seedSubjects[i%len(seedSubjects)]}); err != nil {
			return nil, err
		}
	}
	for _, teacherID := range onlineTeacherIDs {
		if err := repo.CreateTeacherSubject(&schedule.TeacherSubject{TeacherID: teacherID, SubjectID: englishID}); err != nil {
			return nil, err
		}
	}

	specID, err := getOrCreateSpecialty(db, schedule.Specialty{Code: "09.02.07", Name: "Информационные системы и программирование"})
	if err != nil {
		return nil, err
	}
	admissionYear := int16(2022)
	currID, err := getOrCreateCurriculum(db, schedule.Curriculum{
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
	calID, err := getOrCreateAcademicCalendar(db, schedule.AcademicCalendar{
		CurriculumID:      currID,
		AcademicYearStart: acYearStart,
		WeeksTotal:        52,
		Notes:             ptrString("Seed calendar"),
	})
	if err != nil {
		return nil, err
	}
	academicWeeks := make([]schedule.AcademicCalendarWeek, 0, 4)
	for i := int16(1); i <= 4; i++ {
		start := acYearStart.AddDate(0, 0, int((i-1)*7))
		academicWeeks = append(academicWeeks, schedule.AcademicCalendarWeek{
			WeekNumber:    i,
			WeekStartDate: start,
			ActivityCode:  "TEACHING",
			ActivityName:  ptrString("Учебная неделя"),
			IsTeaching:    true,
		})
	}
	if _, err := repo.UpsertAcademicCalendarWeeks(calID, academicWeeks); err != nil {
		return nil, err
	}

	group1ID, err := getOrCreateGroup(db, schedule.Group{
		Name:          "22290907/1095",
		Course:        4,
		CurriculumID:  &currID,
		AdmissionYear: &admissionYear,
		SpecialtyID:   &specID,
	})
	if err != nil {
		return nil, err
	}
	group2ID, err := getOrCreateGroup(db, schedule.Group{
		Name:          "22290907/1096",
		Course:        4,
		CurriculumID:  &currID,
		AdmissionYear: &admissionYear,
		SpecialtyID:   &specID,
	})
	if err != nil {
		return nil, err
	}

	rootID, err := getOrCreateCurriculumItem(db, schedule.CurriculumItem{
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
	for _, seed := range []struct {
		index      string
		name       string
		subjectID  int
		hours      int
		assessment string
	}{
		{index: "1.1", name: "Математика", subjectID: mathID, hours: 64, assessment: "EXAM"},
		{index: "1.2", name: "Информатика", subjectID: infID, hours: 72, assessment: "GRADED_CREDIT"},
		{index: "1.3", name: "Иностранный язык", subjectID: englishID, hours: 48, assessment: "CREDIT"},
		{index: "1.4", name: "Физическая культура", subjectID: peID, hours: 48, assessment: "CREDIT"},
		{index: "1.5", name: "Учебная практика", subjectID: practiceID, hours: 36, assessment: "CREDIT"},
	} {
		subjectID := seed.subjectID
		itemID, err := getOrCreateCurriculumItem(db, schedule.CurriculumItem{
			CurriculumID: currID,
			ParentID:     &rootID,
			IndexCode:    ptrString(seed.index),
			ItemType:     "DISCIPLINE",
			Name:         seed.name,
			SubjectID:    &subjectID,
		})
		if err != nil {
			return nil, err
		}
		if _, err := repo.UpsertCurriculumItemAllocations(itemID, []schedule.CurriculumItemAllocation{{
			Semester:       1,
			Weeks:          ptrI16(16),
			HoursTotal:     ptrInt(seed.hours),
			HoursLectures:  ptrInt(seed.hours / 2),
			HoursPractice:  ptrInt(seed.hours / 2),
			AssessmentType: ptrString(seed.assessment),
		}}); err != nil {
			return nil, err
		}
	}

	teachingID, err := repo.GetOrCreateStudyActivity("TEACHING", "Учебные занятия", "TEACHING", true)
	if err != nil {
		return nil, err
	}
	practiceActivityID, err := repo.GetOrCreateStudyActivity("PRACTICE", "Практика", "PRACTICE", false)
	if err != nil {
		return nil, err
	}
	examActivityID, err := repo.GetOrCreateStudyActivity("EXAM", "Экзамен", "EXAM", true)
	if err != nil {
		return nil, err
	}
	for _, groupID := range []int{group1ID, group2ID} {
		weeks := []schedule.StudyCalendarWeek{
			{WeekNumber: 1, WeekStartDate: &acYearStart, ActivityID: &teachingID, AllowsLessons: true},
			{WeekNumber: 2, WeekStartDate: ptrTime(acYearStart.AddDate(0, 0, 7)), ActivityID: &practiceActivityID, AllowsLessons: false, Comment: ptrString("Производственная практика")},
			{WeekNumber: 3, WeekStartDate: ptrTime(acYearStart.AddDate(0, 0, 14)), ActivityID: &examActivityID, AllowsLessons: true, Comment: ptrString("Экзаменационная неделя")},
			{WeekNumber: 4, WeekStartDate: ptrTime(acYearStart.AddDate(0, 0, 21)), ActivityID: &practiceActivityID, AllowsLessons: false, Comment: ptrString("Учебная практика")},
		}
		if _, err := repo.UpsertStudyCalendarWeeks(groupID, weeks); err != nil {
			return nil, err
		}
	}

	for _, a := range []schedule.CourseAssignment{
		{GroupID: group1ID, Semester: 1, SubjectID: mathID, Status: schedule.StatusPublished, TeacherID: &mathTeacherID},
		{GroupID: group1ID, Semester: 1, SubjectID: infID, Status: schedule.StatusPublished, TeacherID: &infTeacherID},
		{GroupID: group1ID, Semester: 1, SubjectID: englishID, Status: schedule.StatusPublished, TeacherID: &onlineTeacherID},
		{GroupID: group1ID, Semester: 1, SubjectID: peID, Status: schedule.StatusPublished, TeacherID: &peTeacherID},
		{GroupID: group2ID, Semester: 1, SubjectID: mathID, Status: schedule.StatusPublished, TeacherID: &mathTeacherID},
		{GroupID: group2ID, Semester: 1, SubjectID: infID, Status: schedule.StatusPublished, TeacherID: &infTeacherID},
		{GroupID: group2ID, Semester: 1, SubjectID: englishID, Status: schedule.StatusPublished, TeacherID: &onlineTeacherID},
		{GroupID: group2ID, Semester: 1, SubjectID: peID, Status: schedule.StatusPublished, TeacherID: &peTeacherID},
	} {
		if _, err := getOrCreateCourseAssignment(db, a); err != nil {
			return nil, err
		}
	}

	online := "online"
	templateSeeds := []schedule.ScheduleTemplate{
		{GroupID: group1ID, DayOfWeek: 0, WeekParity: schedule.WeekParityBoth, PairNumber: 1, SubjectID: mathID, LocationID: &loc101ID, Status: schedule.StatusPublished, TeacherID: &mathTeacherID},
		{GroupID: group1ID, DayOfWeek: 0, WeekParity: schedule.WeekParityBoth, PairNumber: 2, SubjectID: infID, LocationID: &locComputerID, Status: schedule.StatusPublished, TeacherID: &infTeacherID},
		{GroupID: group1ID, DayOfWeek: 1, WeekParity: schedule.WeekParityBoth, PairNumber: 1, SubjectID: englishID, LocationID: &locOnlineID, LessonFormat: online, Status: schedule.StatusPublished, TeacherID: &onlineTeacherID},
		{GroupID: group1ID, DayOfWeek: 2, WeekParity: schedule.WeekParityBoth, PairNumber: 2, SubjectID: peID, LocationID: &locGymID, Status: schedule.StatusPublished, TeacherID: &peTeacherID},
		{GroupID: group2ID, DayOfWeek: 0, WeekParity: schedule.WeekParityBoth, PairNumber: 1, SubjectID: mathID, LocationID: &loc102ID, Status: schedule.StatusPublished, TeacherID: &mathTeacherID},
		{GroupID: group2ID, DayOfWeek: 0, WeekParity: schedule.WeekParityBoth, PairNumber: 2, SubjectID: infID, LocationID: &locComputerID, Status: schedule.StatusPublished, TeacherID: &infTeacherID, FlowKey: ptrString("stream-inf-1")},
		{GroupID: group2ID, DayOfWeek: 1, WeekParity: schedule.WeekParityBoth, PairNumber: 1, SubjectID: englishID, LocationID: &locOnlineID, LessonFormat: online, Status: schedule.StatusPublished, TeacherID: &onlineTeacherID},
		{GroupID: group2ID, DayOfWeek: 2, WeekParity: schedule.WeekParityBoth, PairNumber: 2, SubjectID: peID, LocationID: &locPoolID, Status: schedule.StatusPublished, TeacherID: &peTeacherID},
	}
	var infTemplateID int64
	for _, tpl := range templateSeeds {
		id, err := getOrCreateScheduleTemplate(db, tpl)
		if err != nil {
			return nil, err
		}
		if tpl.GroupID == group1ID && tpl.SubjectID == infID {
			infTemplateID = id
		}
	}

	weekStart := mondayOfWeekSeed(acYearStart)
	for _, availability := range []schedule.LocationWeekAvailability{
		{LocationID: loc101ID, IsAvailable: true},
		{LocationID: loc102ID, IsAvailable: true},
		{LocationID: locComputerID, IsAvailable: true, Comment: ptrString("ВЦ на неделю")},
		{LocationID: locGymID, IsAvailable: true},
		{LocationID: locPoolID, IsAvailable: true},
	} {
		if _, err := repo.UpsertLocationWeekAvailability(weekStart, []schedule.LocationWeekAvailability{availability}); err != nil {
			return nil, err
		}
	}

	if err := upsertTeacherDayConstraint(db, schedule.TeacherDayConstraint{
		TeacherID:            onlineTeacherID,
		TargetDate:           acYearStart.AddDate(0, 0, 1),
		Reason:               "Методический день",
		ConstraintLevel:      "warning",
		RequiresConfirmation: true,
	}); err != nil {
		return nil, err
	}
	if err := upsertTeacherLocationPreference(db, schedule.TeacherLocationPreference{
		TeacherID:  mathTeacherID,
		LocationID: loc101ID,
		Priority:   1,
		Comment:    ptrString("Закрепленный кабинет"),
	}); err != nil {
		return nil, err
	}
	primorskayaPreferenceLocations := []int{loc101ID, loc102ID, locComputerID}
	for i, teacherID := range teacherIDs[:30] {
		if err := upsertTeacherLocationPreference(db, schedule.TeacherLocationPreference{
			TeacherID:  teacherID,
			LocationID: primorskayaPreferenceLocations[i%len(primorskayaPreferenceLocations)],
			Priority:   1,
			Comment:    ptrString("Только пары на Приморской площадке"),
		}); err != nil {
			return nil, err
		}
	}
	for _, teacherID := range onlineTeacherIDs {
		if err := upsertTeacherLocationPreference(db, schedule.TeacherLocationPreference{
			TeacherID:  teacherID,
			LocationID: locOnlineID,
			Priority:   1,
			Comment:    ptrString("Онлайн-пары"),
		}); err != nil {
			return nil, err
		}
	}
	if err := upsertRoomRequest(db, schedule.RoomRequest{
		TeacherID:      &infTeacherID,
		SubjectID:      &infID,
		GroupID:        &group1ID,
		Semester:       ptrI16(1),
		RequiredTypeID: &computerTypeID,
		Priority:       1,
		Status:         "approved",
		Comment:        ptrString("Нужен ВЦ для практических занятий"),
	}); err != nil {
		return nil, err
	}
	if infTemplateID > 0 {
		if err := upsertRoomAssignment(db, schedule.RoomAssignment{
			ScheduleTemplateID: &infTemplateID,
			LocationID:         locComputerID,
			Source:             "request",
			Status:             schedule.StatusPublished,
		}); err != nil {
			return nil, err
		}
	}

	if _, err := repo.GetSystemState(); err != nil {
		return nil, err
	}
	users, err := seedUsers(repo, group1ID)
	if err != nil {
		return nil, err
	}

	return &seedResult{
		SpecialtyID:  specID,
		CurriculumID: currID,
		CalendarID:   calID,
		GroupIDs:     []int{group1ID, group2ID},
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
		row.ShortName = s.ShortName
		if err := db.Save(&row).Error; err != nil {
			return 0, err
		}
		return row.ID, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	if err := db.Create(&s).Error; err != nil {
		return 0, err
	}
	return s.ID, nil
}

func getOrCreateCampus(db *gorm.DB, c schedule.Campus) (int, error) {
	var row schedule.Campus
	if err := db.Where("name = ?", c.Name).First(&row).Error; err == nil {
		row.Address = c.Address
		if err := db.Save(&row).Error; err != nil {
			return 0, err
		}
		return row.ID, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	if err := db.Create(&c).Error; err != nil {
		return 0, err
	}
	return c.ID, nil
}

func getOrCreateLocationType(db *gorm.DB, t schedule.LocationType) (int, error) {
	var row schedule.LocationType
	if err := db.Where("code = ?", t.Code).First(&row).Error; err == nil {
		row.Name = t.Name
		if err := db.Save(&row).Error; err != nil {
			return 0, err
		}
		return row.ID, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	if err := db.Create(&t).Error; err != nil {
		return 0, err
	}
	return t.ID, nil
}

func getOrCreateLocation(db *gorm.DB, l schedule.Location) (int, error) {
	if l.Kind == "" {
		l.Kind = "physical"
	}
	var row schedule.Location
	if err := db.Where("name = ?", l.Name).First(&row).Error; err == nil {
		row.CampusID = l.CampusID
		row.Kind = l.Kind
		row.Capacity = l.Capacity
		row.IsActive = l.IsActive
		if err := db.Save(&row).Error; err != nil {
			return 0, err
		}
		return row.ID, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	if err := db.Create(&l).Error; err != nil {
		return 0, err
	}
	return l.ID, nil
}

func ensureLocationTypeLink(db *gorm.DB, locationID, typeID int) error {
	return db.Exec(
		"INSERT INTO location_type_links (location_id, type_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
		locationID, typeID,
	).Error
}

func getOrCreateTeacher(db *gorm.DB, t schedule.Teacher) (int, error) {
	var out struct {
		ID int `gorm:"column:id"`
	}
	err := db.Raw(
		"INSERT INTO teachers (name) VALUES (?) ON CONFLICT (name_key) DO UPDATE SET name = EXCLUDED.name, deleted_at = NULL RETURNING id",
		t.Name,
	).Scan(&out).Error
	if err != nil {
		return 0, err
	}
	return out.ID, nil
}

func getOrCreateSpecialty(db *gorm.DB, s schedule.Specialty) (int, error) {
	var row schedule.Specialty
	if err := db.Where("code = ?", s.Code).First(&row).Error; err == nil {
		row.Name = s.Name
		if err := db.Save(&row).Error; err != nil {
			return 0, err
		}
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
		row.Title = c.Title
		row.IsActive = c.IsActive
		row.Notes = c.Notes
		if err := db.Save(&row).Error; err != nil {
			return 0, err
		}
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
		row.WeeksTotal = ac.WeeksTotal
		row.Notes = ac.Notes
		if err := db.Save(&row).Error; err != nil {
			return 0, err
		}
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

func getOrCreateCourseAssignment(db *gorm.DB, a schedule.CourseAssignment) (int64, error) {
	q := db.Where("group_id = ? AND semester = ? AND subject_id = ? AND status = ?", a.GroupID, a.Semester, a.SubjectID, a.Status)
	if a.Subgroup == nil {
		q = q.Where("subgroup IS NULL")
	} else {
		q = q.Where("subgroup = ?", *a.Subgroup)
	}
	var row schedule.CourseAssignment
	if err := q.First(&row).Error; err == nil {
		row.TeacherID = a.TeacherID
		row.CurriculumItemID = a.CurriculumItemID
		row.Notes = a.Notes
		if err := db.Save(&row).Error; err != nil {
			return 0, err
		}
		return row.ID, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	if err := db.Create(&a).Error; err != nil {
		return 0, err
	}
	return a.ID, nil
}

func getOrCreateScheduleTemplate(db *gorm.DB, tpl schedule.ScheduleTemplate) (int64, error) {
	if tpl.LessonFormat == "" {
		tpl.LessonFormat = "offline"
	}
	q := db.Where(
		"group_id = ? AND day_of_week = ? AND week_parity = ? AND pair_number = ? AND subject_id = ? AND status = ?",
		tpl.GroupID, tpl.DayOfWeek, tpl.WeekParity, tpl.PairNumber, tpl.SubjectID, tpl.Status,
	)
	if tpl.Subgroup == nil {
		q = q.Where("subgroup IS NULL")
	} else {
		q = q.Where("subgroup = ?", *tpl.Subgroup)
	}
	var row schedule.ScheduleTemplate
	if err := q.First(&row).Error; err == nil {
		row.LocationID = tpl.LocationID
		row.LessonFormat = tpl.LessonFormat
		row.TeacherID = tpl.TeacherID
		row.TeacherManual = tpl.TeacherManual
		row.LocationManual = tpl.LocationManual
		row.FlowKey = tpl.FlowKey
		if err := db.Omit("TeacherName").Save(&row).Error; err != nil {
			return 0, err
		}
		return row.ID, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	if err := db.Omit("TeacherName").Create(&tpl).Error; err != nil {
		return 0, err
	}
	return tpl.ID, nil
}

func upsertTeacherDayConstraint(db *gorm.DB, d schedule.TeacherDayConstraint) error {
	if d.ConstraintLevel == "" {
		d.ConstraintLevel = "warning"
	}
	if d.ConstraintLevel == "warning" {
		d.RequiresConfirmation = true
	}
	return db.Exec(`
INSERT INTO teacher_day_constraints (teacher_id, target_date, reason, constraint_level, requires_confirmation)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT (teacher_id, target_date)
DO UPDATE SET reason = EXCLUDED.reason, constraint_level = EXCLUDED.constraint_level, requires_confirmation = EXCLUDED.requires_confirmation`,
		d.TeacherID, dateOnly(d.TargetDate), d.Reason, d.ConstraintLevel, d.RequiresConfirmation,
	).Error
}

func upsertTeacherLocationPreference(db *gorm.DB, p schedule.TeacherLocationPreference) error {
	var row schedule.TeacherLocationPreference
	err := db.Where("teacher_id = ? AND location_id = ? AND priority = ?", p.TeacherID, p.LocationID, p.Priority).First(&row).Error
	if err == nil {
		row.ValidFrom = p.ValidFrom
		row.ValidTo = p.ValidTo
		row.Comment = p.Comment
		return db.Save(&row).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return db.Create(&p).Error
}

func upsertRoomRequest(db *gorm.DB, r schedule.RoomRequest) error {
	q := db.Where("teacher_id = ? AND subject_id = ? AND group_id = ?", r.TeacherID, r.SubjectID, r.GroupID)
	if r.Semester == nil {
		q = q.Where("semester IS NULL")
	} else {
		q = q.Where("semester = ?", *r.Semester)
	}
	var row schedule.RoomRequest
	err := q.First(&row).Error
	if err == nil {
		row.RequiredTypeID = r.RequiredTypeID
		row.PreferredLocationID = r.PreferredLocationID
		row.Priority = r.Priority
		row.Comment = r.Comment
		row.Status = r.Status
		return db.Save(&row).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return db.Create(&r).Error
}

func upsertRoomAssignment(db *gorm.DB, a schedule.RoomAssignment) error {
	var row schedule.RoomAssignment
	q := db.Model(&schedule.RoomAssignment{})
	if a.ScheduleTemplateID != nil {
		q = q.Where("schedule_template_id = ?", *a.ScheduleTemplateID)
	} else if a.ScheduleOverrideID != nil {
		q = q.Where("schedule_override_id = ?", *a.ScheduleOverrideID)
	} else {
		return fmt.Errorf("room assignment owner required")
	}
	err := q.First(&row).Error
	if err == nil {
		row.LocationID = a.LocationID
		row.Source = a.Source
		row.Status = a.Status
		return db.Save(&row).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return db.Create(&a).Error
}

func defaultAdmissionYear(now time.Time) int16 {
	if now.Month() < time.September {
		return int16(now.Year() - 1)
	}
	return int16(now.Year())
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func mondayOfWeekSeed(t time.Time) time.Time {
	d := dateOnly(t)
	wd := int(d.Weekday())
	offset := (wd + 6) % 7
	return d.AddDate(0, 0, -offset)
}

func ptrString(v string) *string { return &v }
func ptrInt(v int) *int          { return &v }
func ptrI16(v int16) *int16      { return &v }
func ptrTime(v time.Time) *time.Time {
	d := dateOnly(v)
	return &d
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
