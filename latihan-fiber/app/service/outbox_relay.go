package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
)

type TaskEnqueuer interface {
	EnqueueContext(ctx context.Context, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

type OutboxRelay struct {
	outbox   repository.OutboxRepository
	tx       *repository.TxManager
	queue    TaskEnqueuer
	logger   *slog.Logger
	interval time.Duration
	batch    int
}

func NewOutboxRelay(
	outbox repository.OutboxRepository,
	tx *repository.TxManager,
	queue TaskEnqueuer,
	logger *slog.Logger,
	interval time.Duration,
	batch int,
) *OutboxRelay {
	return &OutboxRelay{outbox: outbox, tx: tx, queue: queue, logger: logger,
		interval: interval, batch: batch}
}

func (r *OutboxRelay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		n, err := r.RelayOnce(ctx)
		if err != nil {
			r.logger.Warn("relay_delayed",
				slog.Int("published", n), slog.String("error", err.Error()))
		} else if n > 0 {
			r.logger.Info("relay_done", slog.Int("published", n))
		}

		if err == nil && n == r.batch && ctx.Err() == nil {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r *OutboxRelay) RelayOnce(ctx context.Context) (int, error) {
	var published int
	var enqueueErr error

	err := r.tx.WithTxOnce(ctx, func(tx repository.DBTX) error {
		repo := r.outbox.WithTx(tx)

		events, err := repo.FetchUnpublished(ctx, r.batch)
		if err != nil {
			return err
		}

		ids := make([]int64, 0, len(events))
		for _, e := range events {
			if err := r.publish(ctx, e); err != nil {
				enqueueErr = err
				break
			}	
			ids = append(ids, e.ID)
		}

		if err := repo.MarkPublished(ctx, ids); err != nil {
			return err
		}
		published = len(ids)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return published, enqueueErr
}

func (r *OutboxRelay) publish(ctx context.Context, e model.OutboxEvent) error {
	// Event kuota ditujukan ke banyak peserta: pecah menjadi satu task per peserta.
	if e.EventType == model.EventCourseQuotaChanged {
		return r.publishQuotaChanged(ctx, e)
	}

	job := model.NotificationJob{
		OutboxID: e.ID, EventType: e.EventType, Payload: e.Payload,
	}

	// bug 2: [] byte
	payloadBytes, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("gagal marshal job outbox %d: %w", e.ID, err)
	}

	// Masukkan payloadBytes ke param kedua, bukan job
	task := asynq.NewTask(model.TaskSendNotification, payloadBytes,
		asynq.TaskID(fmt.Sprintf("outbox-%d", e.ID)),
		asynq.Queue(model.QueueNotifications),
		asynq.MaxRetry(5),
		asynq.Timeout(30*time.Second),
		asynq.Retention(24*time.Hour),
	)

	_, err = r.queue.EnqueueContext(ctx, task)
	if errors.Is(err, asynq.ErrTaskIDConflict) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("enqueue outbox %d: %w", e.ID, err)
	}
	return nil
}
// publishQuotaChanged memecah satu event kuota menjadi satu task per peserta.
//
// TaskID "outbox-<id>-u<user>" membuat enqueue ulang untuk peserta yang sama
// ditolak Redis (ErrTaskIDConflict), dan itu dianggap berhasil. Jadi bila
// relay gagal di tengah jalan, putaran berikutnya aman mengulang dari awal.
func (r *OutboxRelay) publishQuotaChanged(ctx context.Context, e model.OutboxEvent) error {
	var batch model.QuotaChangedEvent
	if err := json.Unmarshal(e.Payload, &batch); err != nil {
		return fmt.Errorf("payload event kuota %d rusak: %w", e.ID, err)
	}

	for _, userID := range batch.UserIDs {
		one := batch
		one.UserIDs = nil
		one.UserID = userID

		payload, err := json.Marshal(one)
		if err != nil {
			return fmt.Errorf("gagal marshal event kuota %d: %w", e.ID, err)
		}
		body, err := json.Marshal(model.NotificationJob{
			OutboxID: e.ID, EventType: e.EventType, Payload: payload,
		})
		if err != nil {
			return fmt.Errorf("gagal marshal job outbox %d: %w", e.ID, err)
		}

		task := asynq.NewTask(model.TaskSendNotification, body,
			asynq.TaskID(fmt.Sprintf("outbox-%d-u%d", e.ID, userID)),
			asynq.Queue(model.QueueNotifications),
			asynq.MaxRetry(5),
			asynq.Timeout(30*time.Second),
			asynq.Retention(24*time.Hour),
		)

		_, err = r.queue.EnqueueContext(ctx, task)
		if errors.Is(err, asynq.ErrTaskIDConflict) {
			continue
		}
		if err != nil {
			return fmt.Errorf("enqueue outbox %d untuk user %d: %w", e.ID, userID, err)
		}
	}
	return nil
}