package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type tgUpdate struct {
	CallbackQuery *tgCallbackQuery `json:"callback_query"`
}

type tgCallbackQuery struct {
	ID      string    `json:"id"`
	Data    string    `json:"data"`
	Message tgMessage `json:"message"`
}

type tgMessage struct {
	MessageID int64 `json:"message_id"`
}

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

// TelegramWebhook обрабатывает callback-запросы от Telegram (нажатие inline-кнопок).
// Telegram ждёт 200 OK в течение 60 секунд — отвечаем сразу, обработку делаем в горутине.
func (h *Handler) TelegramWebhook(w http.ResponseWriter, r *http.Request) {
	if h.webhookSecret != "" &&
		r.Header.Get("X-Telegram-Bot-Api-Secret-Token") != h.webhookSecret {
		w.WriteHeader(http.StatusOK)
		return
	}

	var update tgUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		log.Printf("webhook: decode error: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Отвечаем Telegram немедленно, иначе он будет ждать пока мы сами вызываем его API.
	w.WriteHeader(http.StatusOK)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	if update.CallbackQuery == nil {
		return
	}

	go h.processCallback(update.CallbackQuery)
}

func (h *Handler) processCallback(cq *tgCallbackQuery) {
	log.Printf("webhook: callback id=%s data=%q", cq.ID, cq.Data)

	parts := strings.SplitN(cq.Data, ":", 2)
	if len(parts) != 2 {
		log.Printf("webhook: bad callback data: %q", cq.Data)
		return
	}

	status, ok := callbackToStatus[parts[0]]
	if !ok {
		log.Printf("webhook: unknown action: %q", parts[0])
		return
	}

	bookingID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		log.Printf("webhook: bad booking id: %q", parts[1])
		return
	}

	log.Printf("webhook: booking #%d → %s", bookingID, status)

	if err := h.db.UpdateBookingStatus(bookingID, status); err != nil {
		log.Printf("webhook: update status error: %v", err)
		_ = h.notifier.AnswerCallback(cq.ID, "Ошибка обновления")
		return
	}

	booking, err := h.db.GetBooking(bookingID)
	if err != nil {
		log.Printf("webhook: get booking error: %v", err)
		_ = h.notifier.AnswerCallback(cq.ID, "Ошибка получения заявки")
		return
	}

	log.Printf("webhook: editing tg_message_id=%d", booking.TgMessageID)
	if err := h.notifier.EditMessage(booking.TgMessageID, booking); err != nil {
		log.Printf("webhook: edit message error: %v", err)
	}
	if err := h.notifier.AnswerCallback(cq.ID, statusLabels[status]); err != nil {
		log.Printf("webhook: answer callback error: %v", err)
	}
}
