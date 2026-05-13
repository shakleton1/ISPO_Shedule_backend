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
	if err := deactivateLegacySeedTeachers(db); err != nil {
		return nil, err
	}

	instrToolsID, err := getOrCreateSubject(db, schedule.Subject{Name: "МДК.02.02 Инстр. средства разр. ПО", ShortName: "МДК.02.02"})
	if err != nil {
		return nil, err
	}
	englishPDID, err := getOrCreateSubject(db, schedule.Subject{Name: "Ин. яз.в ПД", ShortName: "Ин. яз. ПД"})
	if err != nil {
		return nil, err
	}
	peID, err := getOrCreateSubject(db, schedule.Subject{Name: "Физ. культура", ShortName: "Физ-ра"})
	if err != nil {
		return nil, err
	}
	economicsID, err := getOrCreateSubject(db, schedule.Subject{Name: "Экономика отрасли", ShortName: "Экономика"})
	if err != nil {
		return nil, err
	}
	managementID, err := getOrCreateSubject(db, schedule.Subject{Name: "Менеджмент в проф. деятельности", ShortName: "Менеджмент"})
	if err != nil {
		return nil, err
	}
	mathModelID, err := getOrCreateSubject(db, schedule.Subject{Name: "МДК.02.03 Матем. моделирование", ShortName: "МДК.02.03"})
	if err != nil {
		return nil, err
	}
	standardsID, err := getOrCreateSubject(db, schedule.Subject{Name: "Стандарт., сертиф. и техн. докумен.", ShortName: "Стандартизация"})
	if err != nil {
		return nil, err
	}
	trpoID, err := getOrCreateSubject(db, schedule.Subject{Name: "МДК.02.01 ТРПО", ShortName: "МДК.02.01"})
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

	loc403ID, err := getOrCreateLocation(db, schedule.Location{Name: "403 П", CampusID: &campusID, Kind: "physical", IsActive: true, Capacity: ptrI16(30)})
	if err != nil {
		return nil, err
	}
	loc441ID, err := getOrCreateLocation(db, schedule.Location{Name: "441", CampusID: &campusID, Kind: "physical", IsActive: true, Capacity: ptrI16(18)})
	if err != nil {
		return nil, err
	}
	loc548ID, err := getOrCreateLocation(db, schedule.Location{Name: "548", CampusID: &campusID, Kind: "physical", IsActive: true, Capacity: ptrI16(18)})
	if err != nil {
		return nil, err
	}
	locSK5ID, err := getOrCreateLocation(db, schedule.Location{Name: "СК5", CampusID: &campusID, Kind: "physical", IsActive: true, Capacity: ptrI16(90)})
	if err != nil {
		return nil, err
	}
	locEconomicsID, err := getOrCreateLocation(db, schedule.Location{Name: "1#", CampusID: &campusID, Kind: "physical", IsActive: true, Capacity: ptrI16(32)})
	if err != nil {
		return nil, err
	}
	locModelingID, err := getOrCreateLocation(db, schedule.Location{Name: "!", CampusID: &campusID, Kind: "physical", IsActive: true, Capacity: ptrI16(32)})
	if err != nil {
		return nil, err
	}
	locTRPOID, err := getOrCreateLocation(db, schedule.Location{Name: "{{", CampusID: &campusID, Kind: "physical", IsActive: true, Capacity: ptrI16(32)})
	if err != nil {
		return nil, err
	}
	locComputerID, err := getOrCreateLocation(db, schedule.Location{Name: "ВЦ-1", CampusID: &campusID, Kind: "physical", IsActive: true, Capacity: ptrI16(30)})
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
		{loc403ID, computerTypeID},
		{loc441ID, classroomTypeID},
		{loc548ID, classroomTypeID},
		{locSK5ID, gymTypeID},
		{locEconomicsID, classroomTypeID},
		{locModelingID, classroomTypeID},
		{locTRPOID, classroomTypeID},
		{locComputerID, computerTypeID},
		{locPoolID, poolTypeID},
		{locOnlineID, onlineTypeID},
	} {
		if err := ensureLocationTypeLink(db, link.locationID, link.typeID); err != nil {
			return nil, err
		}
	}

	tuzovaID, err := getOrCreateTeacher(db, schedule.Teacher{Name: "Тузова Д.А."})
	if err != nil {
		return nil, err
	}
	kuznetsovaID, err := getOrCreateTeacher(db, schedule.Teacher{Name: "Кузнецова Л.И."})
	if err != nil {
		return nil, err
	}
	pshenitsynaID, err := getOrCreateTeacher(db, schedule.Teacher{Name: "Пшеницына М.А."})
	if err != nil {
		return nil, err
	}
	smirnovID, err := getOrCreateTeacher(db, schedule.Teacher{Name: "Смирнов А.Н."})
	if err != nil {
		return nil, err
	}
	vimbergID, err := getOrCreateTeacher(db, schedule.Teacher{Name: "Вимберг С.В."})
	if err != nil {
		return nil, err
	}
	zernovaID, err := getOrCreateTeacher(db, schedule.Teacher{Name: "Зернова Е.Н."})
	if err != nil {
		return nil, err
	}
	chelishchevaID, err := getOrCreateTeacher(db, schedule.Teacher{Name: "Челищева Л.Н."})
	if err != nil {
		return nil, err
	}
	teacherIDs := []int{tuzovaID, kuznetsovaID, pshenitsynaID, smirnovID, vimbergID, zernovaID, chelishchevaID}
	for i := 8; i <= 80; i++ {
		id, err := getOrCreateTeacher(db, schedule.Teacher{Name: fmt.Sprintf("Тестовый преподаватель %02d", i)})
		if err != nil {
			return nil, err
		}
		teacherIDs = append(teacherIDs, id)
	}
	onlineTeacherIDs := teacherIDs[77:80]
	onlineTeacherID := onlineTeacherIDs[0]
	if err := resetSeedTeacherPreferences(db, teacherIDs); err != nil {
		return nil, err
	}

	for _, ts := range []schedule.TeacherSubject{
		{TeacherID: tuzovaID, SubjectID: instrToolsID},
		{TeacherID: kuznetsovaID, SubjectID: englishPDID},
		{TeacherID: pshenitsynaID, SubjectID: englishPDID},
		{TeacherID: smirnovID, SubjectID: peID},
		{TeacherID: vimbergID, SubjectID: economicsID},
		{TeacherID: vimbergID, SubjectID: managementID},
		{TeacherID: zernovaID, SubjectID: mathModelID},
		{TeacherID: zernovaID, SubjectID: standardsID},
		{TeacherID: chelishchevaID, SubjectID: trpoID},
		{TeacherID: tuzovaID, SubjectID: practiceID},
	} {
		if err := repo.CreateTeacherSubject(&ts); err != nil {
			return nil, err
		}
	}
	seedSubjects := []int{instrToolsID, englishPDID, peID, economicsID, managementID, mathModelID, standardsID, trpoID, practiceID}
	for i, teacherID := range teacherIDs {
		if err := repo.CreateTeacherSubject(&schedule.TeacherSubject{TeacherID: teacherID, SubjectID: seedSubjects[i%len(seedSubjects)]}); err != nil {
			return nil, err
		}
	}
	for _, teacherID := range onlineTeacherIDs {
		if err := repo.CreateTeacherSubject(&schedule.TeacherSubject{TeacherID: teacherID, SubjectID: englishPDID}); err != nil {
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

	acYearStart := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	calID, err := getOrCreateAcademicCalendar(db, schedule.AcademicCalendar{
		CurriculumID:      currID,
		AcademicYearStart: acYearStart,
		WeeksTotal:        52,
		Notes:             ptrString("Seed calendar"),
	})
	if err != nil {
		return nil, err
	}
	academicWeeks := make([]schedule.AcademicCalendarWeek, 0, 52)
	for i := int16(1); i <= 52; i++ {
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

	if err := resetSeedGroupPlanning(db, []int{group1ID, group2ID}); err != nil {
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
	curriculumItemIDs := map[int]int64{}
	for _, seed := range []struct {
		index      string
		name       string
		subjectID  int
		hours      int
		assessment string
	}{
		{index: "МДК.02.02", name: "МДК.02.02 Инстр. средства разр. ПО", subjectID: instrToolsID, hours: 144, assessment: "GRADED_CREDIT"},
		{index: "ОГСЭ.03", name: "Ин. яз.в ПД", subjectID: englishPDID, hours: 72, assessment: "CREDIT"},
		{index: "ОГСЭ.04", name: "Физ. культура", subjectID: peID, hours: 72, assessment: "CREDIT"},
		{index: "ОП.09", name: "Экономика отрасли", subjectID: economicsID, hours: 64, assessment: "CREDIT"},
		{index: "ОП.10", name: "Менеджмент в проф. деятельности", subjectID: managementID, hours: 54, assessment: "CREDIT"},
		{index: "МДК.02.03", name: "МДК.02.03 Матем. моделирование", subjectID: mathModelID, hours: 108, assessment: "EXAM"},
		{index: "ОП.11", name: "Стандарт., сертиф. и техн. докумен.", subjectID: standardsID, hours: 72, assessment: "CREDIT"},
		{index: "МДК.02.01", name: "МДК.02.01 ТРПО", subjectID: trpoID, hours: 144, assessment: "GRADED_CREDIT"},
		{index: "УП.02", name: "Учебная практика", subjectID: practiceID, hours: 36, assessment: "CREDIT"},
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
		curriculumItemIDs[seed.subjectID] = itemID
		if _, err := repo.UpsertCurriculumItemAllocations(itemID, []schedule.CurriculumItemAllocation{{
			Semester:       8,
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
			{WeekNumber: 30, WeekStartDate: ptrTime(time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)), ActivityID: &teachingID, AllowsLessons: true},
			{WeekNumber: 31, WeekStartDate: ptrTime(time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)), ActivityID: &teachingID, AllowsLessons: true},
			{WeekNumber: 32, WeekStartDate: ptrTime(time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)), ActivityID: &practiceActivityID, AllowsLessons: false, Comment: ptrString("Производственная практика")},
			{WeekNumber: 33, WeekStartDate: ptrTime(time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)), ActivityID: &examActivityID, AllowsLessons: true, Comment: ptrString("Экзаменационная неделя")},
		}
		if _, err := repo.UpsertStudyCalendarWeeks(groupID, weeks); err != nil {
			return nil, err
		}
	}

	semester := int16(8)
	subgroup1 := int16(1)
	subgroup2 := int16(2)
	for _, a := range []schedule.CourseAssignment{
		{GroupID: group1ID, Semester: semester, SubjectID: instrToolsID, Status: schedule.StatusPublished, TeacherID: &tuzovaID, CurriculumItemID: ptrInt64(curriculumItemIDs[instrToolsID])},
		{GroupID: group1ID, Semester: semester, SubjectID: englishPDID, Status: schedule.StatusPublished, TeacherID: &kuznetsovaID, CurriculumItemID: ptrInt64(curriculumItemIDs[englishPDID]), Subgroup: &subgroup1},
		{GroupID: group1ID, Semester: semester, SubjectID: englishPDID, Status: schedule.StatusPublished, TeacherID: &pshenitsynaID, CurriculumItemID: ptrInt64(curriculumItemIDs[englishPDID]), Subgroup: &subgroup2},
		{GroupID: group1ID, Semester: semester, SubjectID: peID, Status: schedule.StatusPublished, TeacherID: &smirnovID, CurriculumItemID: ptrInt64(curriculumItemIDs[peID])},
		{GroupID: group1ID, Semester: semester, SubjectID: economicsID, Status: schedule.StatusPublished, TeacherID: &vimbergID, CurriculumItemID: ptrInt64(curriculumItemIDs[economicsID])},
		{GroupID: group1ID, Semester: semester, SubjectID: managementID, Status: schedule.StatusPublished, TeacherID: &vimbergID, CurriculumItemID: ptrInt64(curriculumItemIDs[managementID])},
		{GroupID: group1ID, Semester: semester, SubjectID: mathModelID, Status: schedule.StatusPublished, TeacherID: &zernovaID, CurriculumItemID: ptrInt64(curriculumItemIDs[mathModelID])},
		{GroupID: group1ID, Semester: semester, SubjectID: standardsID, Status: schedule.StatusPublished, TeacherID: &zernovaID, CurriculumItemID: ptrInt64(curriculumItemIDs[standardsID])},
		{GroupID: group1ID, Semester: semester, SubjectID: trpoID, Status: schedule.StatusPublished, TeacherID: &chelishchevaID, CurriculumItemID: ptrInt64(curriculumItemIDs[trpoID])},
		{GroupID: group2ID, Semester: semester, SubjectID: instrToolsID, Status: schedule.StatusPublished, TeacherID: &tuzovaID, CurriculumItemID: ptrInt64(curriculumItemIDs[instrToolsID])},
		{GroupID: group2ID, Semester: semester, SubjectID: englishPDID, Status: schedule.StatusPublished, TeacherID: &onlineTeacherID, CurriculumItemID: ptrInt64(curriculumItemIDs[englishPDID])},
		{GroupID: group2ID, Semester: semester, SubjectID: peID, Status: schedule.StatusPublished, TeacherID: &smirnovID, CurriculumItemID: ptrInt64(curriculumItemIDs[peID])},
		{GroupID: group2ID, Semester: semester, SubjectID: economicsID, Status: schedule.StatusPublished, TeacherID: &vimbergID, CurriculumItemID: ptrInt64(curriculumItemIDs[economicsID])},
		{GroupID: group2ID, Semester: semester, SubjectID: mathModelID, Status: schedule.StatusPublished, TeacherID: &zernovaID, CurriculumItemID: ptrInt64(curriculumItemIDs[mathModelID])},
		{GroupID: group2ID, Semester: semester, SubjectID: trpoID, Status: schedule.StatusPublished, TeacherID: &chelishchevaID, CurriculumItemID: ptrInt64(curriculumItemIDs[trpoID])},
	} {
		if _, err := getOrCreateCourseAssignment(db, a); err != nil {
			return nil, err
		}
	}

	online := "online"
	templateSeeds := []schedule.ScheduleTemplate{
		{GroupID: group1ID, DayOfWeek: 0, WeekParity: schedule.WeekParityDenominator, PairNumber: 3, SubjectID: instrToolsID, Status: schedule.StatusPublished, TeacherID: &tuzovaID},
		{GroupID: group1ID, DayOfWeek: 0, WeekParity: schedule.WeekParityDenominator, PairNumber: 4, SubjectID: instrToolsID, LocationID: &loc403ID, Status: schedule.StatusPublished, TeacherID: &tuzovaID},
		{GroupID: group1ID, DayOfWeek: 1, WeekParity: schedule.WeekParityDenominator, PairNumber: 2, SubjectID: englishPDID, LocationID: &loc548ID, Status: schedule.StatusPublished, TeacherID: &pshenitsynaID, Subgroup: &subgroup2},
		{GroupID: group1ID, DayOfWeek: 1, WeekParity: schedule.WeekParityDenominator, PairNumber: 3, SubjectID: peID, LocationID: &locSK5ID, Status: schedule.StatusPublished, TeacherID: &smirnovID},
		{GroupID: group1ID, DayOfWeek: 1, WeekParity: schedule.WeekParityDenominator, PairNumber: 4, SubjectID: economicsID, LocationID: &locEconomicsID, Status: schedule.StatusPublished, TeacherID: &vimbergID},
		{GroupID: group1ID, DayOfWeek: 2, WeekParity: schedule.WeekParityDenominator, PairNumber: 1, SubjectID: instrToolsID, LocationID: &loc403ID, Status: schedule.StatusPublished, TeacherID: &tuzovaID},
		{GroupID: group1ID, DayOfWeek: 2, WeekParity: schedule.WeekParityDenominator, PairNumber: 2, SubjectID: instrToolsID, LocationID: &loc403ID, Status: schedule.StatusPublished, TeacherID: &tuzovaID},
		{GroupID: group1ID, DayOfWeek: 3, WeekParity: schedule.WeekParityDenominator, PairNumber: 1, SubjectID: englishPDID, LocationID: &loc441ID, Status: schedule.StatusPublished, TeacherID: &kuznetsovaID, Subgroup: &subgroup1},
		{GroupID: group1ID, DayOfWeek: 3, WeekParity: schedule.WeekParityDenominator, PairNumber: 1, SubjectID: englishPDID, LocationID: &loc548ID, Status: schedule.StatusPublished, TeacherID: &pshenitsynaID, Subgroup: &subgroup2},
		{GroupID: group1ID, DayOfWeek: 3, WeekParity: schedule.WeekParityDenominator, PairNumber: 2, SubjectID: managementID, LocationID: &locEconomicsID, Status: schedule.StatusPublished, TeacherID: &vimbergID},
		{GroupID: group1ID, DayOfWeek: 4, WeekParity: schedule.WeekParityDenominator, PairNumber: 1, SubjectID: mathModelID, LocationID: &locModelingID, Status: schedule.StatusPublished, TeacherID: &zernovaID},
		{GroupID: group1ID, DayOfWeek: 4, WeekParity: schedule.WeekParityDenominator, PairNumber: 2, SubjectID: mathModelID, LocationID: &locModelingID, Status: schedule.StatusPublished, TeacherID: &zernovaID},
		{GroupID: group1ID, DayOfWeek: 4, WeekParity: schedule.WeekParityDenominator, PairNumber: 3, SubjectID: standardsID, LocationID: &locModelingID, Status: schedule.StatusPublished, TeacherID: &zernovaID},
		{GroupID: group1ID, DayOfWeek: 5, WeekParity: schedule.WeekParityDenominator, PairNumber: 1, SubjectID: trpoID, LocationID: &locTRPOID, Status: schedule.StatusPublished, TeacherID: &chelishchevaID},
		{GroupID: group1ID, DayOfWeek: 5, WeekParity: schedule.WeekParityDenominator, PairNumber: 2, SubjectID: trpoID, LocationID: &locTRPOID, Status: schedule.StatusPublished, TeacherID: &chelishchevaID},

		{GroupID: group1ID, DayOfWeek: 0, WeekParity: schedule.WeekParityNumerator, PairNumber: 2, SubjectID: englishPDID, LocationID: &loc441ID, Status: schedule.StatusPublished, TeacherID: &kuznetsovaID, Subgroup: &subgroup1},
		{GroupID: group1ID, DayOfWeek: 0, WeekParity: schedule.WeekParityNumerator, PairNumber: 3, SubjectID: economicsID, LocationID: &locEconomicsID, Status: schedule.StatusPublished, TeacherID: &vimbergID},
		{GroupID: group1ID, DayOfWeek: 0, WeekParity: schedule.WeekParityNumerator, PairNumber: 4, SubjectID: mathModelID, LocationID: &locModelingID, Status: schedule.StatusPublished, TeacherID: &zernovaID},
		{GroupID: group1ID, DayOfWeek: 1, WeekParity: schedule.WeekParityNumerator, PairNumber: 1, SubjectID: peID, LocationID: &locSK5ID, Status: schedule.StatusPublished, TeacherID: &smirnovID},
		{GroupID: group1ID, DayOfWeek: 1, WeekParity: schedule.WeekParityNumerator, PairNumber: 2, SubjectID: mathModelID, LocationID: &locModelingID, Status: schedule.StatusPublished, TeacherID: &zernovaID},
		{GroupID: group1ID, DayOfWeek: 2, WeekParity: schedule.WeekParityNumerator, PairNumber: 2, SubjectID: englishPDID, LocationID: &loc548ID, Status: schedule.StatusPublished, TeacherID: &pshenitsynaID, Subgroup: &subgroup2},
		{GroupID: group1ID, DayOfWeek: 2, WeekParity: schedule.WeekParityNumerator, PairNumber: 3, SubjectID: trpoID, LocationID: &locTRPOID, Status: schedule.StatusPublished, TeacherID: &chelishchevaID},
		{GroupID: group1ID, DayOfWeek: 2, WeekParity: schedule.WeekParityNumerator, PairNumber: 4, SubjectID: trpoID, LocationID: &locTRPOID, Status: schedule.StatusPublished, TeacherID: &chelishchevaID},
		{GroupID: group1ID, DayOfWeek: 3, WeekParity: schedule.WeekParityNumerator, PairNumber: 1, SubjectID: instrToolsID, LocationID: &loc403ID, Status: schedule.StatusPublished, TeacherID: &tuzovaID},
		{GroupID: group1ID, DayOfWeek: 3, WeekParity: schedule.WeekParityNumerator, PairNumber: 2, SubjectID: instrToolsID, LocationID: &loc403ID, Status: schedule.StatusPublished, TeacherID: &tuzovaID},
		{GroupID: group1ID, DayOfWeek: 4, WeekParity: schedule.WeekParityNumerator, PairNumber: 3, SubjectID: trpoID, LocationID: &locTRPOID, Status: schedule.StatusPublished, TeacherID: &chelishchevaID},
		{GroupID: group1ID, DayOfWeek: 4, WeekParity: schedule.WeekParityNumerator, PairNumber: 4, SubjectID: trpoID, LocationID: &locTRPOID, Status: schedule.StatusPublished, TeacherID: &chelishchevaID},
		{GroupID: group1ID, DayOfWeek: 4, WeekParity: schedule.WeekParityNumerator, PairNumber: 5, SubjectID: standardsID, LocationID: &locModelingID, Status: schedule.StatusPublished, TeacherID: &zernovaID},
		{GroupID: group1ID, DayOfWeek: 5, WeekParity: schedule.WeekParityNumerator, PairNumber: 1, SubjectID: economicsID, LocationID: &locEconomicsID, Status: schedule.StatusPublished, TeacherID: &vimbergID},
		{GroupID: group1ID, DayOfWeek: 5, WeekParity: schedule.WeekParityNumerator, PairNumber: 2, SubjectID: economicsID, LocationID: &locEconomicsID, Status: schedule.StatusPublished, TeacherID: &vimbergID},
		{GroupID: group1ID, DayOfWeek: 5, WeekParity: schedule.WeekParityNumerator, PairNumber: 3, SubjectID: mathModelID, LocationID: &locModelingID, Status: schedule.StatusPublished, TeacherID: &zernovaID},

		{GroupID: group2ID, DayOfWeek: 0, WeekParity: schedule.WeekParityDenominator, PairNumber: 1, SubjectID: trpoID, LocationID: &locTRPOID, Status: schedule.StatusPublished, TeacherID: &chelishchevaID},
		{GroupID: group2ID, DayOfWeek: 0, WeekParity: schedule.WeekParityDenominator, PairNumber: 2, SubjectID: instrToolsID, LocationID: &loc403ID, Status: schedule.StatusPublished, TeacherID: &tuzovaID, FlowKey: ptrString("stream-403-tools")},
		{GroupID: group2ID, DayOfWeek: 1, WeekParity: schedule.WeekParityDenominator, PairNumber: 1, SubjectID: englishPDID, LessonFormat: online, Status: schedule.StatusPublished, TeacherID: &onlineTeacherIDs[0]},
		{GroupID: group2ID, DayOfWeek: 1, WeekParity: schedule.WeekParityDenominator, PairNumber: 2, SubjectID: peID, LocationID: &locPoolID, Status: schedule.StatusPublished, TeacherID: &smirnovID},
		{GroupID: group2ID, DayOfWeek: 2, WeekParity: schedule.WeekParityDenominator, PairNumber: 1, SubjectID: mathModelID, LocationID: &locModelingID, Status: schedule.StatusPublished, TeacherID: &zernovaID},
		{GroupID: group2ID, DayOfWeek: 3, WeekParity: schedule.WeekParityNumerator, PairNumber: 1, SubjectID: englishPDID, LessonFormat: online, Status: schedule.StatusPublished, TeacherID: &onlineTeacherIDs[1]},
		{GroupID: group2ID, DayOfWeek: 4, WeekParity: schedule.WeekParityNumerator, PairNumber: 2, SubjectID: englishPDID, LessonFormat: online, Status: schedule.StatusPublished, TeacherID: &onlineTeacherIDs[2]},
		{GroupID: group2ID, DayOfWeek: 5, WeekParity: schedule.WeekParityNumerator, PairNumber: 1, SubjectID: economicsID, LocationID: &locEconomicsID, Status: schedule.StatusPublished, TeacherID: &vimbergID},
	}
	var requestTemplateID int64
	for _, tpl := range templateSeeds {
		id, err := getOrCreateScheduleTemplate(db, tpl)
		if err != nil {
			return nil, err
		}
		if tpl.GroupID == group1ID && tpl.SubjectID == instrToolsID && tpl.DayOfWeek == 0 && tpl.WeekParity == schedule.WeekParityDenominator && tpl.PairNumber == 3 {
			requestTemplateID = id
		}
	}

	weekStart := time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)
	for _, availability := range []schedule.LocationWeekAvailability{
		{LocationID: loc403ID, IsAvailable: true, Comment: ptrString("403 П на неделю")},
		{LocationID: loc441ID, IsAvailable: true},
		{LocationID: loc548ID, IsAvailable: true},
		{LocationID: locSK5ID, IsAvailable: true},
		{LocationID: locEconomicsID, IsAvailable: true},
		{LocationID: locModelingID, IsAvailable: true},
		{LocationID: locTRPOID, IsAvailable: true},
		{LocationID: locComputerID, IsAvailable: true, Comment: ptrString("ВЦ на неделю")},
		{LocationID: locPoolID, IsAvailable: true},
	} {
		if _, err := repo.UpsertLocationWeekAvailability(weekStart, []schedule.LocationWeekAvailability{availability}); err != nil {
			return nil, err
		}
	}

	if err := upsertTeacherDayConstraint(db, schedule.TeacherDayConstraint{
		TeacherID:            zernovaID,
		TargetDate:           time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
		Reason:               "Методический день",
		ConstraintLevel:      "warning",
		RequiresConfirmation: true,
	}); err != nil {
		return nil, err
	}
	if err := upsertTeacherLocationPreference(db, schedule.TeacherLocationPreference{
		TeacherID:  tuzovaID,
		LocationID: loc403ID,
		Priority:   1,
		Comment:    ptrString("Закрепленный кабинет"),
	}); err != nil {
		return nil, err
	}
	primorskayaPreferenceLocations := []int{loc403ID, loc441ID, loc548ID, locSK5ID, locEconomicsID, locModelingID, locTRPOID, locComputerID}
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
		TeacherID:      &tuzovaID,
		SubjectID:      &instrToolsID,
		GroupID:        &group1ID,
		Semester:       ptrI16(semester),
		RequiredTypeID: &computerTypeID,
		Priority:       1,
		Status:         "approved",
		Comment:        ptrString("Нужен компьютерный класс для МДК.02.02"),
	}); err != nil {
		return nil, err
	}
	if requestTemplateID > 0 {
		if err := upsertRoomAssignment(db, schedule.RoomAssignment{
			ScheduleTemplateID: &requestTemplateID,
			LocationID:         loc403ID,
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

func deactivateLegacySeedTeachers(db *gorm.DB) error {
	legacyNames := []string{
		"Иванов И.И.",
		"Петров П.П.",
		"Сидорова А.А.",
		"Кузнецов К.К.",
		"Тестовый преподаватель 05",
		"Тестовый преподаватель 06",
		"Тестовый преподаватель 07",
	}
	var legacyIDs []int
	if err := db.Table("teachers").Where("name IN ?", legacyNames).Pluck("id", &legacyIDs).Error; err != nil {
		return err
	}
	if len(legacyIDs) > 0 {
		if err := db.Exec("DELETE FROM teacher_location_preferences WHERE teacher_id IN ?", legacyIDs).Error; err != nil {
			return err
		}
		if err := db.Exec("DELETE FROM room_requests WHERE teacher_id IN ?", legacyIDs).Error; err != nil {
			return err
		}
	}
	return db.Exec("UPDATE teachers SET deleted_at = now() WHERE name IN ? AND deleted_at IS NULL", legacyNames).Error
}

func resetSeedTeacherPreferences(db *gorm.DB, teacherIDs []int) error {
	if len(teacherIDs) == 0 {
		return nil
	}
	return db.Exec("DELETE FROM teacher_location_preferences WHERE teacher_id IN ?", teacherIDs).Error
}

func resetSeedGroupPlanning(db *gorm.DB, groupIDs []int) error {
	if len(groupIDs) == 0 {
		return nil
	}
	statements := []string{
		"DELETE FROM schedule_replacements WHERE group_id IN ?",
		"DELETE FROM schedule_overrides WHERE group_id IN ?",
		"DELETE FROM schedule_templates WHERE group_id IN ?",
		"DELETE FROM course_assignments WHERE group_id IN ?",
		"DELETE FROM study_calendar_weeks WHERE group_id IN ?",
		"DELETE FROM room_requests WHERE group_id IN ?",
		"DELETE FROM schedule_day_overlays WHERE group_id IN ?",
	}
	for _, stmt := range statements {
		if err := db.Exec(stmt, groupIDs).Error; err != nil {
			return err
		}
	}
	return nil
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
		"group_id = ? AND day_of_week = ? AND week_parity = ? AND pair_number = ? AND status = ?",
		tpl.GroupID, tpl.DayOfWeek, tpl.WeekParity, tpl.PairNumber, tpl.Status,
	)
	if tpl.Subgroup == nil {
		q = q.Where("subgroup IS NULL")
	} else {
		q = q.Where("subgroup = ?", *tpl.Subgroup)
	}
	var row schedule.ScheduleTemplate
	if err := q.First(&row).Error; err == nil {
		row.SubjectID = tpl.SubjectID
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

func ptrString(v string) *string { return &v }
func ptrInt(v int) *int          { return &v }
func ptrI16(v int16) *int16      { return &v }
func ptrInt64(v int64) *int64    { return &v }
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
