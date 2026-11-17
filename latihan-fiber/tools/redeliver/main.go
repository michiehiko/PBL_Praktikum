package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/hibiken/asynq"
	"github.com/joho/godotenv"

	"latihan-fiber/app/model"
	"latihan-fiber/config"
	"latihan-fiber/database"
)

func main() {
	_ = godotenv.Load()

	outboxID := flag.Int64("outbox", 0, "id baris outbox yang akan dikirim ulang")
	copies := flag.Int("copies", 5, "jumlah salinan task (per peserta untuk event kuota)")
	onlyUser := flag.Int("user", 0, "khusus event kuota: kirim ulang hanya untuk peserta ini")
	flag.Parse()

	if *outboxID <= 0 || *copies <= 0 {
		fmt.Fprintln(os.Stderr, "pemakaian: go run ./tools/redeliver -outbox <id> -copies <jumlah> [-user <id>]")
		os.Exit(1)
	}

	ctx := context.Background()

	pool, err := database.NewPool(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gagal terhubung ke database:", err)
		os.Exit(1)
	}
	defer pool.Close()

	var eventType string
	var payload []byte
	err = pool.QueryRow(ctx,
		`SELECT event_type, payload FROM outbox WHERE id = $1`, *outboxID,
	).Scan(&eventType, &payload)
	if err != nil {
		fmt.Fprintln(os.Stderr, "baris outbox tidak ditemukan:", err)
		os.Exit(1)
	}

	client := asynq.NewClient(config.RedisOpt())
	defer client.Close()

	// Setiap elemen adalah (isi event untuk satu task, penanda untuk TaskID).
	type target struct {
		payload []byte
		label   string
	}
	var targets []target

	if eventType == model.EventCourseQuotaChanged {
		var batch model.QuotaChangedEvent
		if err := json.Unmarshal(payload, &batch); err != nil {
			fmt.Fprintln(os.Stderr, "payload event kuota rusak:", err)
			os.Exit(1)
		}
		for _, userID := range batch.UserIDs {
			if *onlyUser != 0 && userID != *onlyUser {
				continue
			}
			one := batch
			one.UserIDs = nil
			one.UserID = userID
			b, _ := json.Marshal(one)
			targets = append(targets, target{payload: b, label: fmt.Sprintf("u%d", userID)})
		}
		if len(targets) == 0 {
			fmt.Fprintln(os.Stderr, "tidak ada peserta yang cocok pada event ini")
			os.Exit(1)
		}
	} else {
		targets = append(targets, target{payload: payload, label: "all"})
	}

	total := 0
	for _, t := range targets {
		body, err := json.Marshal(model.NotificationJob{
			OutboxID:  *outboxID,
			EventType: eventType,
			Payload:   json.RawMessage(t.payload),
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "gagal menyusun task:", err)
			os.Exit(1)
		}

		for i := 1; i <= *copies; i++ {
			// TaskID sengaja dibuat BERBEDA dari "outbox-<id>" milik relay dan 
			// berbeda satu sama lain, supaya Redis menerima semuanya sebagai task terpisah.
			task := asynq.NewTask(model.TaskSendNotification, body,
				asynq.TaskID(fmt.Sprintf("redeliver-%d-%s-%d", *outboxID, t.label, i)),
				asynq.Queue(model.QueueNotifications),
				asynq.MaxRetry(5),
				asynq.Timeout(30*time.Second),
				asynq.Retention(24*time.Hour),
			)
			if _, err := client.EnqueueContext(ctx, task); err != nil {
				fmt.Fprintf(os.Stderr, "salinan %d (%s) gagal di-enqueue: %v\n", i, t.label, err)
				os.Exit(1)
			}
			total++
		}
		fmt.Printf("%d salinan di-enqueue (outbox_id=%d, event=%s, penerima=%s)\n",
			*copies, *outboxID, eventType, t.label)
	}
	fmt.Printf("total %d task di-enqueue\n", total)
}