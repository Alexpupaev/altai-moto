package handler

import (
	"context"
	"log"
	"time"
)

func (h *Handler) StartPolling(ctx context.Context) {
	log.Println("poll: deleting webhook and starting long polling")
	if err := h.notifier.DeleteWebhook(); err != nil {
		log.Printf("poll: delete webhook error: %v", err)
	}

	var offset int64
	for {
		if ctx.Err() != nil {
			log.Println("poll: stopped")
			return
		}

		updates, err := h.notifier.GetUpdates(ctx, offset)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("poll: getUpdates error: %v", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}

		for _, u := range updates {
			offset = u.UpdateID + 1
			if u.CallbackQuery != nil {
				go h.processCallback(u.CallbackQuery)
			}
		}
	}
}
