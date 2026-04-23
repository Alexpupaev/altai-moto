package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/altai-moto/moto-rental/internal/notify"
	"github.com/altai-moto/moto-rental/internal/storage"
)

// rePhone — только цифры, нужно минимум 10 (российский номер без кода страны).
var rePhone = regexp.MustCompile(`\d`)

// Handler объединяет зависимости обработчиков.
// Это Go-аналог DI-контейнера: зависимости передаются явно через конструктор.
type Handler struct {
	db            *storage.DB
	notifier      *notify.Notifier
	webhookSecret string
}

func New(db *storage.DB, notifier *notify.Notifier, webhookSecret string) *Handler {
	return &Handler{db: db, notifier: notifier, webhookSecret: webhookSecret}
}

// ── GET /api/bookings ─────────────────────────────────────────────────────────

// GetBookings возвращает список занятых диапазонов дат.
// Только [{date_from, date_to}] — никаких личных данных.
func (h *Handler) GetBookings(w http.ResponseWriter, r *http.Request) {
	ranges, err := h.db.GetBookedRanges()
	if err != nil {
		jsonError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	jsonOK(w, ranges)
}

// ── POST /api/booking ─────────────────────────────────────────────────────────

// bookingRequest — структура входящего JSON.
// Website — honeypot: реальный пользователь его не заполнит, бот — заполнит.
type bookingRequest struct {
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	DateFrom string `json:"date_from"`
	DateTo   string `json:"date_to"`
	Website  string `json:"website"` // honeypot
}

type validationErrors map[string]string

func (h *Handler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	// Декодируем JSON из тела запроса.
	// json.NewDecoder(r.Body) — стриминговый декодер, не грузит всё в память.
	var req bookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid json", http.StatusBadRequest)
		return
	}

	// Honeypot: если заполнено — это бот, тихо отвечаем 200 (не раскрываем логику)
	if req.Website != "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Валидация
	if errs := validate(req); len(errs) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity) // 422
		json.NewEncoder(w).Encode(map[string]any{"errors": errs})
		return
	}

	booking := storage.Booking{
		Name:     strings.TrimSpace(req.Name),
		Phone:    req.Phone,
		DateFrom: req.DateFrom,
		DateTo:   req.DateTo,
	}

	id, err := h.db.SaveBooking(booking)
	if err != nil {
		jsonError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	booking.ID = id

	// Уведомление в Telegram в фоне — не блокируем ответ пользователю.
	// Горутина — как async в PHP (но настоящий параллельный поток, не эмуляция).
	go func() {
		log.Printf("tg: sending notification for booking #%d", booking.ID)
		msgID, err := h.notifier.Send(booking)
		if err != nil {
			log.Printf("tg: send error: %v", err)
			return
		}
		log.Printf("tg: sent ok, message_id=%d", msgID)
		if msgID > 0 {
			if err := h.db.SetTgMessageID(id, msgID); err != nil {
				log.Printf("tg: set message_id error: %v", err)
			}
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201
	json.NewEncoder(w).Encode(map[string]any{"id": id})
}

// ── Validation ────────────────────────────────────────────────────────────────

func validate(req bookingRequest) validationErrors {
	errs := validationErrors{}

	if name := strings.TrimSpace(req.Name); name == "" {
		errs["name"] = "Введите имя"
	} else if len([]rune(name)) < 2 {
		// len(name) считает байты, len([]rune(name)) — символы (важно для кириллицы)
		errs["name"] = "Имя слишком короткое"
	}

	digits := rePhone.FindAllString(req.Phone, -1)
	if len(digits) < 10 {
		errs["phone"] = "Введите корректный номер телефона"
	}

	dateFrom, errFrom := time.Parse("2006-01-02", req.DateFrom)
	dateTo, errTo := time.Parse("2006-01-02", req.DateTo)
	// Go использует конкретную дату 2006-01-02 15:04:05 как эталон формата.
	// Это выглядит странно, но это фиксированная дата: Mon Jan 2 15:04:05 MST 2006.

	if errFrom != nil {
		errs["date_from"] = "Выберите дату начала"
	}
	if errTo != nil {
		errs["date_to"] = "Выберите дату окончания"
	}
	if errFrom == nil && errTo == nil && !dateTo.After(dateFrom) {
		errs["date_to"] = "Дата окончания должна быть позже даты начала"
	}

	return errs
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func jsonOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
