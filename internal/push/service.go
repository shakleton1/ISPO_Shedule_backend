package push

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"ispo-schedule/internal/schedule"
)

type Service struct {
	repo    *schedule.Repository
	notify  Notifier
	timeout time.Duration
}

type ServiceDeps struct {
	Repo     *schedule.Repository
	Notifier Notifier
	Timeout  time.Duration
}

func NewService(deps ServiceDeps) *Service {
	n := deps.Notifier
	if n == nil {
		n = noopNotifier{}
	}
	tout := deps.Timeout
	if tout <= 0 {
		tout = 5 * time.Second
	}
	return &Service{repo: deps.Repo, notify: n, timeout: tout}
}

func (s *Service) Enabled() bool {
	_, ok := s.notify.(noopNotifier)
	return !ok
}

func (s *Service) NotifyScheduleUpdatedAsync(groupID int, scheduleVersion time.Time) {
	if !s.Enabled() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
		defer cancel()
		if err := s.NotifyScheduleUpdated(ctx, groupID, scheduleVersion); err != nil {
			log.Error().Err(err).Int("group_id", groupID).Msg("push notify failed")
		}
	}()
}

func (s *Service) NotifyScheduleUpdatedAllAsync(scheduleVersion time.Time) {
	if !s.Enabled() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
		defer cancel()
		ids, err := s.repo.ListSubscribedGroupIDs()
		if err != nil {
			log.Error().Err(err).Msg("push: list subscribed groups")
			return
		}
		for _, gid := range ids {
			if err := s.NotifyScheduleUpdated(ctx, gid, scheduleVersion); err != nil {
				log.Error().Err(err).Int("group_id", gid).Msg("push notify failed")
			}
		}
	}()
}

func (s *Service) NotifyScheduleUpdated(ctx context.Context, groupID int, scheduleVersion time.Time) error {
	if !s.Enabled() {
		return nil
	}
	tokens, err := s.repo.ListDeviceTokensByGroup(groupID)
	if err != nil {
		return fmt.Errorf("list tokens: %w", err)
	}
	data := map[string]string{
		"event":            "schedule_updated",
		"group_id":         fmt.Sprintf("%d", groupID),
		"schedule_version": scheduleVersion.UTC().Format(time.RFC3339Nano),
	}
	for _, t := range tokens {
		if err := s.notify.Send(ctx, t.Token, data); err != nil {
			log.Warn().Err(err).Int64("token_id", t.ID).Msg("push send failed")
		}
	}
	return nil
}

func BuildFCMNotifier(ctx context.Context, projectID, credentialsFile string, timeout time.Duration) (Notifier, error) {
	return newFCMNotifier(ctx, fcmConfig{ProjectID: projectID, CredentialsFile: credentialsFile, RequestTimeout: timeout})
}
