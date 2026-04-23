package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/altai-moto/moto-rental/internal/handler"
	"github.com/altai-moto/moto-rental/internal/middleware"
	"github.com/altai-moto/moto-rental/internal/notify"
	"github.com/altai-moto/moto-rental/internal/storage"
)

func main() {
	// os.Getenv — читаем переменные окружения (из .env через docker-compose).
	// В PHP это $_ENV['KEY'] или getenv('KEY').
	dbPath := getenv("DB_PATH", "/data/bookings.db")
	allowedOrigin := getenv("ALLOWED_ORIGIN", "http://localhost")
	tgToken := getenv("TELEGRAM_BOT_TOKEN", "")
	tgChatID := getenv("TELEGRAM_CHAT_ID", "")
	tgWebhookSecret := getenv("TELEGRAM_WEBHOOK_SECRET", "")
	addr := getenv("LISTEN_ADDR", ":8080")

	// Инициализация зависимостей
	db, err := storage.New(dbPath)
	if err != nil {
		log.Fatalf("storage: %v", err) // Fatalf = Printf + os.Exit(1)
	}

	notifier := notify.New(tgToken, tgChatID)
	h := handler.New(db, notifier, tgWebhookSecret)

	// Rate limiter: 3 запроса за 15 минут с одного IP
	rateLimiter := middleware.NewRateLimiter(3, 15*time.Minute)

	// ── Роутер ───────────────────────────────────────────────────────────────
	// chi — тонкая обёртка над net/http.
	// Если знаешь Slim (PHP), chi выглядит похоже: $app->get('/path', handler).
	r := chi.NewRouter()

	// Глобальные middleware (применяются ко всем маршрутам)
	r.Use(chiMiddleware.Logger)    // логирование запросов
	r.Use(chiMiddleware.Recoverer) // перехватывает panic, отвечает 500
	r.Use(middleware.CORS(allowedOrigin))

	r.Get("/api/bookings", h.GetBookings)
	r.With(rateLimiter.Middleware).Post("/api/booking", h.CreateBooking)
	r.Post("/telegram/webhook", h.TelegramWebhook)

	// ── Старт сервера ─────────────────────────────────────────────────────────
	log.Printf("starting server on %s", addr)

	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ListenAndServe блокирует выполнение (как Apache/nginx в foreground).
	// Возвращает ошибку только при аварийном завершении.
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// getenv возвращает переменную окружения или значение по умолчанию.
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
