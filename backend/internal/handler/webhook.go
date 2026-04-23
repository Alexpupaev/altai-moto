package handler

import (
	"log"
	"strconv"
	"strings"

	"github.com/altai-moto/moto-rental/internal/notify"
)

var statusLabels = map[string]string{
	"confirmed": "✅ Подтверждено",
	"rejected":  "❌ Отклонено",
	"pending":   "🔄 В ожидание",
}

var callbackToStatus = map[string]string{
	"confirm": "confirmed",
	"reject":  "rejected",
	"pending": "pending",
}

func (h *Handler) processCallback(cq *notify.Callback) {
	log.Printf("poll: callback id=%s data=%q", cq.ID, cq.Data)

	parts := strings.SplitN(cq.Data, ":", 2)
	if len(parts) != 2 {
		log.Printf("poll: bad callback data: %q", cq.Data)
		return
	}

	status, ok := callbackToStatus[parts[0]]
	if !ok {
		log.Printf("poll: unknown action: %q", parts[0])
		return
	}

	bookingID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		log.Printf("poll: bad booking id: %q", parts[1])
		return
	}

	log.Printf("poll: booking #%d → %s", bookingID, status)

	if err := h.db.UpdateBookingStatus(bookingID, status); err != nil {
		log.Printf("poll: update status error: %v", err)
		_ = h.notifier.AnswerCallback(cq.ID, "Ошибка обновления")
		return
	}

	booking, err := h.db.GetBooking(bookingID)
	if err != nil {
		log.Printf("poll: get booking error: %v", err)
		_ = h.notifier.AnswerCallback(cq.ID, "Ошибка получения заявки")
		return
	}

	log.Printf("poll: editing tg_message_id=%d", booking.TgMessageID)
	if err := h.notifier.EditMessage(booking.TgMessageID, booking); err != nil {
		log.Printf("poll: edit message error: %v", err)
	}
	if err := h.notifier.AnswerCallback(cq.ID, statusLabels[status]); err != nil {
		log.Printf("poll: answer callback error: %v", err)
	}
}
