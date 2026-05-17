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
	gormlogger "gorm.io/gorm/logger"
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
	gormDB = gormDB.Session(&gorm.Session{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
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
	engelsCampusID, err := getOrCreateCampus(db, schedule.Campus{Name: "Энгельса", Address: ptrString("Площадка на Энгельса")})
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
	labTypeID, err := getOrCreateLocationType(db, schedule.LocationType{Code: "lab", Name: "Лаборатория"})
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

	primorskayaLocations, err := seedLocationCatalog(db, campusID, []seedLocationRoom{
		{Name: "101"}, {Name: "102"}, {Name: "103"}, {Name: "104"}, {Name: "105"}, {Name: "106"}, {Name: "107"}, {Name: "109"}, {Name: "110"}, {Name: "113"}, {Name: "115"}, {Name: "117"},
		{Name: "201"}, {Name: "202"}, {Name: "203"}, {Name: "204"}, {Name: "205"}, {Name: "206"}, {Name: "207"}, {Name: "208"}, {Name: "209"}, {Name: "210"}, {Name: "211"}, {Name: "212"}, {Name: "214"}, {Name: "216"}, {Name: "217"}, {Name: "218"}, {Name: "219"}, {Name: "221"}, {Name: "223"}, {Name: "225"}, {Name: "227"}, {Name: "229"}, {Name: "231"}, {Name: "233"}, {Name: "235"}, {Name: "237"}, {Name: "241"}, {Name: "243"}, {Name: "245"}, {Name: "247"}, {Name: "249"}, {Name: "251"}, {Name: "253"}, {Name: "255"}, {Name: "257"}, {Name: "259"},
		{Name: "301"}, {Name: "302"}, {Name: "303"}, {Name: "304"}, {Name: "305"}, {Name: "306"}, {Name: "307"}, {Name: "308"}, {Name: "310"}, {Name: "311"}, {Name: "312"}, {Name: "314"}, {Name: "315"}, {Name: "316"}, {Name: "318"}, {Name: "319"}, {Name: "320"}, {Name: "321"}, {Name: "322"}, {Name: "323"}, {Name: "324"}, {Name: "325"}, {Name: "326"}, {Name: "327"}, {Name: "329"}, {Name: "331"}, {Name: "333"}, {Name: "335"}, {Name: "337"}, {Name: "339"}, {Name: "341"}, {Name: "343"}, {Name: "345"}, {Name: "347"}, {Name: "349"},
		{Name: "401"}, {Name: "402"}, {Name: "403", Capacity: 30, TypeIDs: []int{computerTypeID}}, {Name: "404", Capacity: 30, TypeIDs: []int{computerTypeID}}, {Name: "405"}, {Name: "406"}, {Name: "408"}, {Name: "409"}, {Name: "410"}, {Name: "411"}, {Name: "413"}, {Name: "415"}, {Name: "417"}, {Name: "428"}, {Name: "430"}, {Name: "432"}, {Name: "434"}, {Name: "436"}, {Name: "438"}, {Name: "439"}, {Name: "441", Capacity: 18}, {Name: "443"}, {Name: "445"},
		{Name: "501"}, {Name: "502"}, {Name: "503"}, {Name: "504"}, {Name: "505"}, {Name: "506"}, {Name: "508"}, {Name: "509"}, {Name: "510"}, {Name: "511"}, {Name: "513"}, {Name: "515"}, {Name: "541"}, {Name: "542"}, {Name: "544"}, {Name: "545"}, {Name: "546"}, {Name: "547"}, {Name: "548", Capacity: 18}, {Name: "549"}, {Name: "550"}, {Name: "551"},
		{Name: "Спортивный зал", Capacity: 90, TypeIDs: []int{gymTypeID}},
		{Name: "Бассейн", Capacity: 45, TypeIDs: []int{poolTypeID}},
	}, classroomTypeID)
	if err != nil {
		return nil, err
	}
	engelsLocations, err := seedLocationCatalog(db, engelsCampusID, []seedLocationRoom{
		{Name: "101"}, {Name: "102"}, {Name: "103"}, {Name: "104"}, {Name: "105"}, {Name: "106"}, {Name: "107"}, {Name: "109"}, {Name: "110"}, {Name: "113"}, {Name: "115"}, {Name: "117"},
		{Name: "200"}, {Name: "201"}, {Name: "202"}, {Name: "203"}, {Name: "204"}, {Name: "206"}, {Name: "207"}, {Name: "208"}, {Name: "209"}, {Name: "210"}, {Name: "211"}, {Name: "212"}, {Name: "214"}, {Name: "215"}, {Name: "216", Capacity: 30, TypeIDs: []int{computerTypeID}}, {Name: "217"}, {Name: "218"}, {Name: "219"}, {Name: "221"}, {Name: "222"}, {Name: "223"}, {Name: "224"}, {Name: "225"}, {Name: "226"}, {Name: "227"}, {Name: "229"}, {Name: "231"}, {Name: "233"}, {Name: "235"}, {Name: "237"}, {Name: "243"}, {Name: "245"}, {Name: "247"}, {Name: "249"}, {Name: "251"}, {Name: "253"}, {Name: "255"}, {Name: "257"}, {Name: "259"},
		{Name: "301"}, {Name: "302"}, {Name: "303"}, {Name: "304"}, {Name: "305"}, {Name: "306"}, {Name: "307"}, {Name: "308"}, {Name: "310"}, {Name: "311"}, {Name: "312"}, {Name: "314", Capacity: 24, TypeIDs: []int{classroomTypeID, labTypeID}}, {Name: "315"}, {Name: "316"}, {Name: "318"}, {Name: "319"}, {Name: "319а"}, {Name: "320"}, {Name: "321"}, {Name: "322"}, {Name: "323"}, {Name: "324"}, {Name: "325", Capacity: 24, TypeIDs: []int{classroomTypeID, labTypeID}}, {Name: "326"}, {Name: "327"}, {Name: "329"}, {Name: "331"}, {Name: "331а"}, {Name: "333"}, {Name: "335"}, {Name: "337"}, {Name: "339"}, {Name: "341"}, {Name: "343"}, {Name: "345"}, {Name: "347"}, {Name: "349"},
		{Name: "401"}, {Name: "402"}, {Name: "403"}, {Name: "404"}, {Name: "405"}, {Name: "406"}, {Name: "408"}, {Name: "409"}, {Name: "410"}, {Name: "411"}, {Name: "413"}, {Name: "415"}, {Name: "417"}, {Name: "428"}, {Name: "430"}, {Name: "432"}, {Name: "434"}, {Name: "436"}, {Name: "438"}, {Name: "439"}, {Name: "441"}, {Name: "443"}, {Name: "445"},
		{Name: "501"}, {Name: "502"}, {Name: "503"}, {Name: "504"}, {Name: "505"}, {Name: "506"}, {Name: "508"}, {Name: "509"}, {Name: "510"}, {Name: "511"}, {Name: "513"}, {Name: "515"}, {Name: "541"}, {Name: "542"}, {Name: "544"}, {Name: "545"}, {Name: "546"}, {Name: "547"}, {Name: "548"}, {Name: "549"}, {Name: "550"}, {Name: "551"}, {Name: "599А"}, {Name: "599Б"}, {Name: "600"},
		{Name: "Конференц-зал", Capacity: 80},
		{Name: "Библиотека", Capacity: 40},
		{Name: "Квант", Capacity: 30, TypeIDs: []int{classroomTypeID, labTypeID}},
		{Name: "Спортивный зал", Capacity: 90, TypeIDs: []int{gymTypeID}},
		{Name: "Бассейн", Capacity: 45, TypeIDs: []int{poolTypeID}},
	}, classroomTypeID)
	if err != nil {
		return nil, err
	}
	loc403ID := primorskayaLocations["403"]
	loc441ID := primorskayaLocations["441"]
	loc548ID := primorskayaLocations["548"]
	locSK5ID := primorskayaLocations["Спортивный зал"]
	locEconomicsID := primorskayaLocations["405"]
	locModelingID := primorskayaLocations["409"]
	locTRPOID := primorskayaLocations["411"]
	locComputerID := primorskayaLocations["403"]
	locComputer2ID := primorskayaLocations["404"]
	locPoolID := primorskayaLocations["Бассейн"]
	locEngels201ID := engelsLocations["201"]
	locEngels202ID := engelsLocations["202"]
	locEngelsVCID := engelsLocations["216"]
	locEngelsLabID := engelsLocations["314"]
	locOnlineID, err := getOrCreateLocation(db, schedule.Location{Name: "Дистант", Kind: "virtual", IsActive: true})
	if err != nil {
		return nil, err
	}
	if err := ensureLocationTypeLink(db, locOnlineID, onlineTypeID); err != nil {
		return nil, err
	}
	if err := deactivateLegacySeedLocations(db); err != nil {
		return nil, err
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
	if err := seedAcademicCalendarWeeks(repo, calID, acYearStart); err != nil {
		return nil, err
	}
	if err := upsertAcademicCalendarDayOverride(db, schedule.AcademicCalendarDayOverride{
		CalendarID:   calID,
		CourseNumber: 1,
		WeekNumber:   4,
		DayOfWeek:    4,
		ActivityCode: "TEACHING",
		ActivityName: ptrString("Учебные занятия"),
		IsTeaching:   true,
		Comment:      ptrString("Seed day override inside academic calendar week"),
	}); err != nil {
		return nil, err
	}
	curriculumByAdmission := map[int16]int64{admissionYear: currID}
	calendarByCurriculum := map[int64]int64{currID: calID}
	for _, year := range []int16{2023, 2024, 2025} {
		id, err := getOrCreateCurriculum(db, schedule.Curriculum{
			SpecialtyID:   specID,
			AdmissionYear: year,
			Variant:       "Базовый",
			Title:         fmt.Sprintf("Учебный план %d (seed)", year),
			IsActive:      true,
		})
		if err != nil {
			return nil, err
		}
		curriculumByAdmission[year] = id
		cid, err := getOrCreateAcademicCalendar(db, schedule.AcademicCalendar{
			CurriculumID:      id,
			AcademicYearStart: acYearStart,
			WeeksTotal:        52,
			Notes:             ptrString("Seed calendar"),
		})
		if err != nil {
			return nil, err
		}
		calendarByCurriculum[id] = cid
		if err := seedAcademicCalendarWeeks(repo, cid, acYearStart); err != nil {
			return nil, err
		}
	}
	incompleteAdmissionYear := int16(2024)
	incompleteCurrID, err := getOrCreateCurriculum(db, schedule.Curriculum{
		SpecialtyID:   specID,
		AdmissionYear: incompleteAdmissionYear,
		Variant:       "Неполный",
		Title:         "Неполный учебный план (seed)",
		IsActive:      true,
		Notes:         ptrString("Для проверки: заполнен 3 семестр, 4 семестр отсутствует"),
	})
	if err != nil {
		return nil, err
	}
	incompleteCalID, err := getOrCreateAcademicCalendar(db, schedule.AcademicCalendar{
		CurriculumID:      incompleteCurrID,
		AcademicYearStart: acYearStart,
		WeeksTotal:        52,
		Notes:             ptrString("Seed incomplete calendar"),
	})
	if err != nil {
		return nil, err
	}
	calendarByCurriculum[incompleteCurrID] = incompleteCalID
	if err := seedAcademicCalendarWeeks(repo, incompleteCalID, acYearStart); err != nil {
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
	group3ID, err := getOrCreateGroup(db, schedule.Group{
		Name:          "22290907/1097",
		Course:        4,
		CurriculumID:  &currID,
		AdmissionYear: &admissionYear,
		SpecialtyID:   &specID,
	})
	if err != nil {
		return nil, err
	}
	groupHalfID, err := getOrCreateGroup(db, schedule.Group{
		Name:          "22290907/1098",
		Course:        4,
		CurriculumID:  &currID,
		AdmissionYear: &admissionYear,
		SpecialtyID:   &specID,
	})
	if err != nil {
		return nil, err
	}
	admission2023 := int16(2023)
	curr2023 := curriculumByAdmission[admission2023]
	groupCourse3ID, err := getOrCreateGroup(db, schedule.Group{
		Name:          "23290907/1093",
		Course:        3,
		CurriculumID:  &curr2023,
		AdmissionYear: &admission2023,
		SpecialtyID:   &specID,
	})
	if err != nil {
		return nil, err
	}
	admission2024 := int16(2024)
	curr2024 := curriculumByAdmission[admission2024]
	groupCourse2ID, err := getOrCreateGroup(db, schedule.Group{
		Name:          "24290907/1092",
		Course:        2,
		CurriculumID:  &curr2024,
		AdmissionYear: &admission2024,
		SpecialtyID:   &specID,
	})
	if err != nil {
		return nil, err
	}
	groupIncompleteID, err := getOrCreateGroup(db, schedule.Group{
		Name:          "24290907/1099",
		Course:        2,
		CurriculumID:  &incompleteCurrID,
		AdmissionYear: &incompleteAdmissionYear,
		SpecialtyID:   &specID,
	})
	if err != nil {
		return nil, err
	}
	admission2025 := int16(2025)
	curr2025 := curriculumByAdmission[admission2025]
	groupCourse1ID, err := getOrCreateGroup(db, schedule.Group{
		Name:          "25290907/1091",
		Course:        1,
		CurriculumID:  &curr2025,
		AdmissionYear: &admission2025,
		SpecialtyID:   &specID,
	})
	if err != nil {
		return nil, err
	}
	seedGroupIDs := []int{group1ID, group2ID, group3ID, groupHalfID, groupCourse1ID, groupCourse2ID, groupCourse3ID, groupIncompleteID}

	if err := resetSeedGroupPlanning(db, seedGroupIDs); err != nil {
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
		if _, err := repo.UpsertCurriculumItemAllocations(itemID, fullSeedAllocations(seed.hours, seed.assessment)); err != nil {
			return nil, err
		}
	}
	curriculumPlans := map[int64]map[int]int64{currID: curriculumItemIDs}
	for _, id := range []int64{curriculumByAdmission[2023], curriculumByAdmission[2024], curriculumByAdmission[2025]} {
		items, err := seedCurriculumPlan(repo, db, id, []seedCurriculumSubject{
			{index: "МДК.02.02", name: "МДК.02.02 Инстр. средства разр. ПО", subjectID: instrToolsID, hours: 144, assessment: "GRADED_CREDIT"},
			{index: "ОГСЭ.03", name: "Ин. яз.в ПД", subjectID: englishPDID, hours: 72, assessment: "CREDIT"},
			{index: "ОГСЭ.04", name: "Физ. культура", subjectID: peID, hours: 72, assessment: "CREDIT"},
			{index: "ОП.09", name: "Экономика отрасли", subjectID: economicsID, hours: 64, assessment: "CREDIT"},
			{index: "ОП.10", name: "Менеджмент в проф. деятельности", subjectID: managementID, hours: 54, assessment: "CREDIT"},
			{index: "МДК.02.03", name: "МДК.02.03 Матем. моделирование", subjectID: mathModelID, hours: 108, assessment: "EXAM"},
			{index: "ОП.11", name: "Стандарт., сертиф. и техн. докумен.", subjectID: standardsID, hours: 72, assessment: "CREDIT"},
			{index: "МДК.02.01", name: "МДК.02.01 ТРПО", subjectID: trpoID, hours: 144, assessment: "GRADED_CREDIT"},
			{index: "УП.02", name: "Учебная практика", subjectID: practiceID, hours: 36, assessment: "CREDIT"},
		}, false)
		if err != nil {
			return nil, err
		}
		curriculumPlans[id] = items
	}
	incompleteItems, err := seedCurriculumPlan(repo, db, incompleteCurrID, []seedCurriculumSubject{
		{index: "МДК.02.02", name: "МДК.02.02 Инстр. средства разр. ПО", subjectID: instrToolsID, hours: 72, assessment: "GRADED_CREDIT"},
		{index: "ОГСЭ.03", name: "Ин. яз.в ПД", subjectID: englishPDID, hours: 36, assessment: "CREDIT"},
		{index: "ОП.09", name: "Экономика отрасли", subjectID: economicsID, hours: 36, assessment: "CREDIT"},
	}, true)
	if err != nil {
		return nil, err
	}
	curriculumPlans[incompleteCurrID] = incompleteItems

	teachingID, err := repo.GetOrCreateStudyActivity("TEACHING", "Учебные занятия", "TEACHING", true)
	if err != nil {
		return nil, err
	}
	practiceEduActivityID, err := repo.GetOrCreateStudyActivity("PRACTICE_EDU", "Учебная практика", "PRACTICE", false)
	if err != nil {
		return nil, err
	}
	practiceProdActivityID, err := repo.GetOrCreateStudyActivity("PRACTICE_PROD", "Производственная практика", "PRACTICE", false)
	if err != nil {
		return nil, err
	}
	practicePregradActivityID, err := repo.GetOrCreateStudyActivity("PRACTICE_PREGRAD", "Преддипломная практика", "PRACTICE", false)
	if err != nil {
		return nil, err
	}
	examActivityID, err := repo.GetOrCreateStudyActivity("EXAM", "Экзамен", "EXAM", true)
	if err != nil {
		return nil, err
	}
	giaActivityID, err := repo.GetOrCreateStudyActivity("GIA", "Государственная итоговая аттестация", "GIA", false)
	if err != nil {
		return nil, err
	}
	vacationActivityID, err := repo.GetOrCreateStudyActivity("VACATION", "Каникулы", "VACATION", false)
	if err != nil {
		return nil, err
	}
	for _, groupID := range seedGroupIDs {
		if err := seedFullStudyCalendar(repo, groupID, acYearStart, teachingID, practiceEduActivityID, practiceProdActivityID, practicePregradActivityID, examActivityID, giaActivityID, vacationActivityID); err != nil {
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
		if a.CampusID == nil {
			a.CampusID = &campusID
		}
		if _, err := getOrCreateCourseAssignment(db, a); err != nil {
			return nil, err
		}
	}
	teacherBySubject := map[int]*int{
		instrToolsID: &tuzovaID,
		englishPDID:  &kuznetsovaID,
		peID:         &smirnovID,
		economicsID:  &vimbergID,
		managementID: &vimbergID,
		mathModelID:  &zernovaID,
		standardsID:  &zernovaID,
		trpoID:       &chelishchevaID,
		practiceID:   &tuzovaID,
	}
	for _, cfg := range []struct {
		groupID  int
		currID   int64
		semester int16
	}{
		{groupID: group2ID, currID: currID, semester: 8},
		{groupID: group3ID, currID: currID, semester: 8},
		{groupID: groupCourse3ID, currID: curr2023, semester: 6},
		{groupID: groupCourse2ID, currID: curr2024, semester: 4},
		{groupID: groupCourse1ID, currID: curr2025, semester: 2},
	} {
		assignmentCampusID := &engelsCampusID
		if cfg.groupID == group2ID || cfg.groupID == group3ID {
			assignmentCampusID = &campusID
		}
		if err := seedCourseAssignments(db, cfg.groupID, cfg.semester, curriculumPlans[cfg.currID], teacherBySubject, false, assignmentCampusID); err != nil {
			return nil, err
		}
	}
	if err := seedCourseAssignments(db, groupHalfID, 8, curriculumPlans[currID], teacherBySubject, true, &engelsCampusID); err != nil {
		return nil, err
	}
	if err := seedCourseAssignments(db, groupIncompleteID, 3, curriculumPlans[incompleteCurrID], teacherBySubject, false, &engelsCampusID); err != nil {
		return nil, err
	}

	online := "online"
	type lessonSeed struct {
		GroupID      int
		DayOfWeek    int16
		WeekParity   schedule.WeekParity
		PairNumber   int16
		SubjectID    int
		LocationID   *int
		LessonFormat string
		Status       schedule.EntityStatus
		TeacherID    *int
		Subgroup     *int16
		FlowKey      *string
	}
	templateSeeds := []lessonSeed{
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
	for _, extra := range []struct {
		groupID  int
		baseRoom int
		half     bool
	}{
		{groupID: group3ID, baseRoom: locComputer2ID},
		{groupID: groupCourse1ID, baseRoom: locEngels201ID},
		{groupID: groupCourse2ID, baseRoom: locEngels202ID},
		{groupID: groupCourse3ID, baseRoom: locEngelsVCID},
		{groupID: groupHalfID, baseRoom: locEngelsLabID, half: true},
		{groupID: groupIncompleteID, baseRoom: locEngels201ID, half: true},
	} {
		var maybeZernova *int = &zernovaID
		var maybeChelishcheva *int = &chelishchevaID
		if extra.half {
			maybeZernova = nil
			maybeChelishcheva = nil
		}
		baseRoom := extra.baseRoom
		templateSeeds = append(templateSeeds,
			lessonSeed{GroupID: extra.groupID, DayOfWeek: 0, WeekParity: schedule.WeekParityDenominator, PairNumber: 1, SubjectID: instrToolsID, LocationID: &baseRoom, Status: schedule.StatusPublished, TeacherID: &tuzovaID},
			lessonSeed{GroupID: extra.groupID, DayOfWeek: 0, WeekParity: schedule.WeekParityDenominator, PairNumber: 2, SubjectID: englishPDID, LocationID: &baseRoom, Status: schedule.StatusPublished, TeacherID: &kuznetsovaID},
			lessonSeed{GroupID: extra.groupID, DayOfWeek: 1, WeekParity: schedule.WeekParityDenominator, PairNumber: 1, SubjectID: peID, LocationID: &locSK5ID, Status: schedule.StatusPublished, TeacherID: &smirnovID},
			lessonSeed{GroupID: extra.groupID, DayOfWeek: 2, WeekParity: schedule.WeekParityNumerator, PairNumber: 1, SubjectID: mathModelID, LocationID: &baseRoom, Status: schedule.StatusPublished, TeacherID: maybeZernova},
			lessonSeed{GroupID: extra.groupID, DayOfWeek: 3, WeekParity: schedule.WeekParityNumerator, PairNumber: 2, SubjectID: trpoID, LocationID: &baseRoom, Status: schedule.StatusPublished, TeacherID: maybeChelishcheva},
		)
	}
	var requestLessonID int64
	for _, tpl := range templateSeeds {
		lessonDate, ok := seedLessonDate(tpl.WeekParity, tpl.DayOfWeek)
		if !ok {
			continue
		}
		subjectID := tpl.SubjectID
		lesson := schedule.ScheduleLesson{
			GroupID:      tpl.GroupID,
			LessonDate:   lessonDate,
			PairNumber:   tpl.PairNumber,
			Subgroup:     tpl.Subgroup,
			SubjectID:    &subjectID,
			TeacherID:    tpl.TeacherID,
			LessonFormat: tpl.LessonFormat,
			Status:       tpl.Status,
			Source:       "manual",
			FlowKey:      tpl.FlowKey,
		}
		id, err := getOrCreateScheduleLesson(db, lesson, tpl.LocationID, "manual")
		if err != nil {
			return nil, err
		}
		if tpl.GroupID == group1ID && tpl.SubjectID == instrToolsID && tpl.DayOfWeek == 0 && tpl.WeekParity == schedule.WeekParityDenominator && tpl.PairNumber == 3 {
			requestLessonID = id
		}
	}

	weekStart := time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)
	for _, availability := range []schedule.LocationWeekAvailability{
		{LocationID: loc403ID, IsAvailable: true, Comment: ptrString("403 на неделю")},
		{LocationID: loc441ID, IsAvailable: true},
		{LocationID: loc548ID, IsAvailable: true},
		{LocationID: locSK5ID, IsAvailable: true},
		{LocationID: locEconomicsID, IsAvailable: true},
		{LocationID: locModelingID, IsAvailable: true},
		{LocationID: locTRPOID, IsAvailable: true},
		{LocationID: locComputerID, IsAvailable: true, Comment: ptrString("ВЦ на неделю")},
		{LocationID: locComputer2ID, IsAvailable: true},
		{LocationID: locPoolID, IsAvailable: true},
		{LocationID: locEngels201ID, IsAvailable: true},
		{LocationID: locEngels202ID, IsAvailable: true},
		{LocationID: locEngelsVCID, IsAvailable: true},
		{LocationID: locEngelsLabID, IsAvailable: true},
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
	for _, constraint := range []schedule.CalendarDayConstraint{
		{
			TargetDate:           time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC),
			Title:                "Праздничный день",
			Reason:               ptrString("Тестовый глобальный выходной"),
			ConstraintType:       "blocked",
			AffectsLessons:       true,
			RequiresConfirmation: false,
			StylePreset:          "danger",
		},
		{
			TargetDate:           time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
			Title:                "Мероприятие",
			Reason:               ptrString("Большое мероприятие в колледже"),
			ConstraintType:       "warning",
			AffectsLessons:       true,
			RequiresConfirmation: true,
			StylePreset:          "warning",
		},
		{
			TargetDate:           time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC),
			Title:                "Информационная пометка",
			Reason:               ptrString("Пары можно ставить"),
			ConstraintType:       "info",
			AffectsLessons:       false,
			RequiresConfirmation: false,
			StylePreset:          "info",
		},
	} {
		if err := upsertCalendarDayConstraint(db, constraint); err != nil {
			return nil, err
		}
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
	if requestLessonID > 0 {
		if err := upsertRoomAssignment(db, schedule.RoomAssignment{
			ScheduleLessonID: requestLessonID,
			LocationID:       loc403ID,
			Source:           "request",
			Status:           schedule.StatusPublished,
		}); err != nil {
			return nil, err
		}
	}

	seedSvc := schedule.NewService(schedule.ServiceDeps{Repo: repo, SemesterStartDate: "2026-02-23", Now: time.Now})
	replaceDate, _ := seedLessonDate(schedule.WeekParityDenominator, 0)
	if replaceLessonID, err := findScheduleLessonID(db, group2ID, replaceDate, 1, nil); err == nil {
		expected := 1
		if err := applySeedOverrideOnce(db, seedSvc, "seed replace example", schedule.ApplyScheduleOverrideRequest{
			ScheduleLessonID:        &replaceLessonID,
			GroupID:                 group2ID,
			LessonDate:              replaceDate,
			PairNumber:              1,
			ActionType:              string(schedule.OverrideReplace),
			ReplacementSubjectID:    &mathModelID,
			ReplacementTeacherID:    &zernovaID,
			ReplacementLocationID:   &locModelingID,
			ReplacementLessonFormat: ptrString("offline"),
			ExpectedLessonVersion:   &expected,
			ConfirmConstraints:      true,
		}); err != nil {
			return nil, err
		}
	}
	cancelDate, _ := seedLessonDate(schedule.WeekParityDenominator, 1)
	if cancelLessonID, err := findScheduleLessonID(db, group2ID, cancelDate, 2, nil); err == nil {
		expected := 1
		if err := applySeedOverrideOnce(db, seedSvc, "seed cancel example", schedule.ApplyScheduleOverrideRequest{
			ScheduleLessonID:      &cancelLessonID,
			GroupID:               group2ID,
			LessonDate:            cancelDate,
			PairNumber:            2,
			ActionType:            string(schedule.OverrideCancel),
			ExpectedLessonVersion: &expected,
			ConfirmConstraints:    true,
		}); err != nil {
			return nil, err
		}
	}
	addDate, _ := seedLessonDate(schedule.WeekParityDenominator, 2)
	if err := applySeedOverrideOnce(db, seedSvc, "seed add example", schedule.ApplyScheduleOverrideRequest{
		GroupID:                 group2ID,
		LessonDate:              addDate,
		PairNumber:              3,
		ActionType:              string(schedule.OverrideAdd),
		ReplacementSubjectID:    &standardsID,
		ReplacementTeacherID:    &zernovaID,
		ReplacementLocationID:   &locModelingID,
		ReplacementLessonFormat: ptrString("offline"),
		ConfirmConstraints:      true,
	}); err != nil {
		return nil, err
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
		GroupIDs:     seedGroupIDs,
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

func deactivateLegacySeedLocations(db *gorm.DB) error {
	legacyNames := []string{
		"403 П",
		"СК5",
		"1#",
		"!",
		"{{",
		"ВЦ-1",
		"ВЦ-2",
		"201 Э",
		"202 Э",
		"ВЦ-Э",
		"Лаб. Э-1",
	}
	return db.Exec("UPDATE locations SET is_active = false WHERE name IN ?", legacyNames).Error
}

func resetSeedGroupPlanning(db *gorm.DB, groupIDs []int) error {
	if len(groupIDs) == 0 {
		return nil
	}
	statements := []string{
		"DELETE FROM schedule_replacements WHERE group_id IN ?",
		"DELETE FROM schedule_overrides WHERE group_id IN ?",
		"DELETE FROM schedule_lessons WHERE group_id IN ?",
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

type seedLocationRoom struct {
	Name     string
	Capacity int16
	TypeIDs  []int
}

func seedLocationCatalog(db *gorm.DB, campusID int, rooms []seedLocationRoom, defaultTypeID int) (map[string]int, error) {
	out := make(map[string]int, len(rooms))
	for _, room := range rooms {
		capacity := room.Capacity
		if capacity == 0 {
			capacity = 30
		}
		locationID, err := getOrCreateLocation(db, schedule.Location{
			Name:     room.Name,
			CampusID: &campusID,
			Kind:     "physical",
			IsActive: true,
			Capacity: ptrI16(capacity),
		})
		if err != nil {
			return nil, err
		}
		typeIDs := room.TypeIDs
		if len(typeIDs) == 0 {
			typeIDs = []int{defaultTypeID}
		}
		for _, typeID := range typeIDs {
			if err := ensureLocationTypeLink(db, locationID, typeID); err != nil {
				return nil, err
			}
		}
		out[room.Name] = locationID
	}
	return out, nil
}

func getOrCreateLocation(db *gorm.DB, l schedule.Location) (int, error) {
	if l.Kind == "" {
		l.Kind = "physical"
	}
	var row schedule.Location
	query := db.Where("name = ?", l.Name)
	if l.CampusID == nil {
		query = query.Where("campus_id IS NULL")
	} else {
		query = query.Where("campus_id = ?", *l.CampusID)
	}
	if err := query.First(&row).Error; err == nil {
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

func upsertAcademicCalendarDayOverride(db *gorm.DB, row schedule.AcademicCalendarDayOverride) error {
	if row.CourseNumber == 0 {
		row.CourseNumber = 1
	}
	return db.Exec(`
INSERT INTO academic_calendar_day_overrides
  (calendar_id, course_number, week_number, day_of_week, activity_code, activity_name, is_teaching, comment)
VALUES
  (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (calendar_id, course_number, week_number, day_of_week)
DO UPDATE SET
  activity_code = EXCLUDED.activity_code,
  activity_name = EXCLUDED.activity_name,
  is_teaching = EXCLUDED.is_teaching,
  comment = EXCLUDED.comment`,
		row.CalendarID,
		row.CourseNumber,
		row.WeekNumber,
		row.DayOfWeek,
		row.ActivityCode,
		row.ActivityName,
		row.IsTeaching,
		row.Comment,
	).Error
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
		row.CampusID = a.CampusID
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

func getOrCreateScheduleLesson(db *gorm.DB, lesson schedule.ScheduleLesson, locationID *int, roomSource string) (int64, error) {
	if lesson.LessonFormat == "" {
		lesson.LessonFormat = "offline"
	}
	if lesson.Status == "" {
		lesson.Status = schedule.StatusPublished
	}
	if lesson.Source == "" {
		lesson.Source = "manual"
	}
	lesson.LessonDate = dateOnly(lesson.LessonDate)
	q := db.Where(
		"group_id = ? AND lesson_date = ? AND pair_number = ? AND status <> ?",
		lesson.GroupID, lesson.LessonDate, lesson.PairNumber, schedule.StatusCancelled,
	)
	if lesson.Subgroup == nil {
		q = q.Where("subgroup IS NULL")
	} else {
		q = q.Where("subgroup = ?", *lesson.Subgroup)
	}
	var row schedule.ScheduleLesson
	if err := q.First(&row).Error; err == nil {
		row.SubjectID = lesson.SubjectID
		row.TeacherID = lesson.TeacherID
		row.LessonFormat = lesson.LessonFormat
		row.Status = lesson.Status
		row.Source = lesson.Source
		row.FlowKey = lesson.FlowKey
		row.Comment = lesson.Comment
		if err := db.Save(&row).Error; err != nil {
			return 0, err
		}
		if locationID != nil {
			if err := upsertRoomAssignment(db, schedule.RoomAssignment{
				ScheduleLessonID: row.ID,
				LocationID:       *locationID,
				Source:           roomSource,
				Status:           schedule.StatusPublished,
			}); err != nil {
				return 0, err
			}
		}
		return row.ID, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	if err := db.Create(&lesson).Error; err != nil {
		return 0, err
	}
	if locationID != nil {
		if err := upsertRoomAssignment(db, schedule.RoomAssignment{
			ScheduleLessonID: lesson.ID,
			LocationID:       *locationID,
			Source:           roomSource,
			Status:           schedule.StatusPublished,
		}); err != nil {
			return 0, err
		}
	}
	return lesson.ID, nil
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

func upsertCalendarDayConstraint(db *gorm.DB, d schedule.CalendarDayConstraint) error {
	var reason any
	if d.Reason != nil {
		reason = *d.Reason
	}
	return db.Exec(`
INSERT INTO calendar_day_constraints
  (target_date, title, reason, constraint_type, affects_lessons, requires_confirmation, style_preset)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (target_date)
DO UPDATE SET
  title = EXCLUDED.title,
  reason = EXCLUDED.reason,
  constraint_type = EXCLUDED.constraint_type,
  affects_lessons = EXCLUDED.affects_lessons,
  requires_confirmation = EXCLUDED.requires_confirmation,
	style_preset = EXCLUDED.style_preset`,
		dateOnly(d.TargetDate),
		d.Title,
		reason,
		d.ConstraintType,
		d.AffectsLessons,
		d.RequiresConfirmation,
		d.StylePreset,
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
	if a.ScheduleLessonID <= 0 {
		return fmt.Errorf("schedule_lesson_id required")
	}
	q := db.Model(&schedule.RoomAssignment{}).Where("schedule_lesson_id = ?", a.ScheduleLessonID)
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

func findScheduleLessonID(db *gorm.DB, groupID int, lessonDate time.Time, pairNumber int16, subgroup *int16) (int64, error) {
	q := db.Model(&schedule.ScheduleLesson{}).
		Where("group_id = ? AND lesson_date = ? AND pair_number = ? AND status <> ?", groupID, dateOnly(lessonDate), pairNumber, schedule.StatusCancelled)
	if subgroup == nil {
		q = q.Where("subgroup IS NULL")
	} else {
		q = q.Where("subgroup = ?", *subgroup)
	}
	var row schedule.ScheduleLesson
	if err := q.First(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

func applySeedOverrideOnce(db *gorm.DB, svc *schedule.Service, reason string, req schedule.ApplyScheduleOverrideRequest) error {
	var cnt int64
	if err := db.Model(&schedule.ScheduleOverride{}).
		Where("group_id = ? AND lesson_date = ? AND pair_number = ? AND action_type = ? AND reason = ?",
			req.GroupID, dateOnly(req.LessonDate), req.PairNumber, schedule.OverrideAction(req.ActionType), reason).
		Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return nil
	}
	req.Reason = &reason
	_, err := svc.ApplyScheduleOverride(req)
	return err
}

type seedCurriculumSubject struct {
	index      string
	name       string
	subjectID  int
	hours      int
	assessment string
}

func seedCurriculumPlan(repo *schedule.Repository, db *gorm.DB, currID int64, subjects []seedCurriculumSubject, incomplete bool) (map[int]int64, error) {
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
	out := map[int]int64{}
	for _, seed := range subjects {
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
		out[seed.subjectID] = itemID
		allocations := fullSeedAllocations(seed.hours, seed.assessment)
		if incomplete {
			allocations = []schedule.CurriculumItemAllocation{{
				Semester:       3,
				Weeks:          ptrI16(16),
				HoursTotal:     ptrInt(seed.hours),
				HoursLectures:  ptrInt(seed.hours / 2),
				HoursPractice:  ptrInt(seed.hours / 2),
				AssessmentType: ptrString(seed.assessment),
			}}
		}
		if _, err := repo.UpsertCurriculumItemAllocations(itemID, allocations); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func fullSeedAllocations(hours int, assessment string) []schedule.CurriculumItemAllocation {
	if hours <= 0 {
		hours = 36
	}
	perSemester := hours / 4
	if perSemester < 18 {
		perSemester = 18
	}
	out := make([]schedule.CurriculumItemAllocation, 0, 8)
	for semester := int16(1); semester <= 8; semester++ {
		a := schedule.CurriculumItemAllocation{
			Semester:       semester,
			Weeks:          ptrI16(16),
			HoursTotal:     ptrInt(perSemester),
			HoursLectures:  ptrInt(perSemester / 2),
			HoursPractice:  ptrInt(perSemester / 2),
			AssessmentType: ptrString("CREDIT"),
		}
		if semester == 8 {
			a.HoursTotal = ptrInt(hours)
			a.HoursLectures = ptrInt(hours / 2)
			a.HoursPractice = ptrInt(hours / 2)
			a.AssessmentType = ptrString(assessment)
		}
		out = append(out, a)
	}
	return out
}

func seedAcademicCalendarWeeks(repo *schedule.Repository, calendarID int64, academicYearStart time.Time) error {
	weeks := make([]schedule.AcademicCalendarWeek, 0, 52*6)
	for course := int16(1); course <= 6; course++ {
		courseStart := academicYearStart.AddDate(int(course-1), 0, 0)
		for week := int16(1); week <= 52; week++ {
			start := courseStart.AddDate(0, 0, int((week-1)*7))
			code, name, teaching := seedAcademicWeekActivity(week)
			weeks = append(weeks, schedule.AcademicCalendarWeek{
				CourseNumber:  course,
				WeekNumber:    week,
				WeekStartDate: start,
				ActivityCode:  code,
				ActivityName:  ptrString(name),
				IsTeaching:    teaching,
			})
		}
	}
	_, err := repo.UpsertAcademicCalendarWeeks(calendarID, weeks)
	return err
}

func seedAcademicWeekActivity(week int16) (string, string, bool) {
	switch week {
	case 32:
		return "PRACTICE_PROD", "Производственная практика", false
	case 33:
		return "EXAM", "Экзаменационная сессия", true
	case 45:
		return "PRACTICE_EDU", "Учебная практика", false
	case 48:
		return "PRACTICE_PREGRAD", "Преддипломная практика", false
	case 50:
		return "GIA", "Государственная итоговая аттестация", false
	case 52:
		return "VACATION", "Каникулы", false
	default:
		return "TEACHING", "Учебные занятия", true
	}
}

func seedFullStudyCalendar(repo *schedule.Repository, groupID int, academicYearStart time.Time, teachingID, practiceEduID, practiceProdID, practicePregradID, examID, giaID, vacationID int) error {
	weeks := make([]schedule.StudyCalendarWeek, 0, 52)
	for week := int16(1); week <= 52; week++ {
		activityID := teachingID
		allowsLessons := true
		comment := ""
		switch week {
		case 32:
			activityID = practiceProdID
			allowsLessons = false
			comment = "Производственная практика"
		case 33:
			activityID = examID
			allowsLessons = true
			comment = "Экзаменационная неделя"
		case 45:
			activityID = practiceEduID
			allowsLessons = false
			comment = "Учебная практика"
		case 48:
			activityID = practicePregradID
			allowsLessons = false
			comment = "Преддипломная практика"
		case 50:
			activityID = giaID
			allowsLessons = false
			comment = "ГИА"
		case 52:
			activityID = vacationID
			allowsLessons = false
			comment = "Каникулы"
		}
		row := schedule.StudyCalendarWeek{
			WeekNumber:    week,
			WeekStartDate: ptrTime(academicYearStart.AddDate(0, 0, int((week-1)*7))),
			ActivityID:    &activityID,
			AllowsLessons: allowsLessons,
		}
		if comment != "" {
			row.Comment = ptrString(comment)
		}
		weeks = append(weeks, row)
	}
	_, err := repo.UpsertStudyCalendarWeeks(groupID, weeks)
	return err
}

func seedCourseAssignments(db *gorm.DB, groupID int, semester int16, itemIDs map[int]int64, teacherBySubject map[int]*int, halfAssigned bool, campusID *int) error {
	i := 0
	for subjectID, itemID := range itemIDs {
		teacherID := teacherBySubject[subjectID]
		if halfAssigned && i%2 == 1 {
			teacherID = nil
		}
		if _, err := getOrCreateCourseAssignment(db, schedule.CourseAssignment{
			GroupID:          groupID,
			Semester:         semester,
			SubjectID:        subjectID,
			Status:           schedule.StatusPublished,
			TeacherID:        teacherID,
			CampusID:         campusID,
			CurriculumItemID: ptrInt64(itemID),
		}); err != nil {
			return err
		}
		i++
	}
	return nil
}

func seedLessonDate(parity schedule.WeekParity, dayOfWeek int16) (time.Time, bool) {
	var weekStart time.Time
	switch parity {
	case schedule.WeekParityDenominator, schedule.WeekParityBoth:
		weekStart = time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)
	case schedule.WeekParityNumerator:
		weekStart = time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)
	default:
		return time.Time{}, false
	}
	if dayOfWeek < 0 || dayOfWeek > 5 {
		return time.Time{}, false
	}
	return weekStart.AddDate(0, 0, int(dayOfWeek)), true
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
