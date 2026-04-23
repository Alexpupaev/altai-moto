package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// ── CORS ──────────────────────────────────────────────────────────────────────

// CORS добавляет заголовки и отвечает на preflight-запросы (OPTIONS).
// allowedOrigin — твой домен, например "https://altai-moto.ru".
func CORS(allowedOrigin string) func(http.Handler) http.Handler {
	// Возвращаем функцию-обёртку — это и есть middleware в Go.
	// В PHP Slim это выглядело бы как $app->add(function($req, $handler) {...})
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			// Preflight — браузер спрашивает "можно?", отвечаем 204 и выходим
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ── Rate Limiting ─────────────────────────────────────────────────────────────

// visitor хранит лимитер и время последнего запроса (для очистки старых записей)
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter хранит лимитеры по IP-адресам.
// sync.Mutex — мьютекс для безопасного доступа из горутин.
// В PHP это не нужно: каждый запрос — отдельный процесс.
// В Go сервер живёт постоянно и обрабатывает запросы параллельно.
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    rate.Limit // запросов в секунду
	burst    int        // максимальный всплеск
}

// NewRateLimiter создаёт лимитер: maxReq запросов за window.
// Пример: NewRateLimiter(3, 15*time.Minute) = 3 запроса за 15 минут.
func NewRateLimiter(maxReq int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    rate.Every(window / time.Duration(maxReq)),
		burst:    maxReq,
	}

	// Горутина — лёгкий поток, запускается через go.
	// Чистим старые IP каждые 5 минут чтобы карта не росла вечно.
	go rl.cleanup(5 * time.Minute)

	return rl
}

func (rl *RateLimiter) getVisitor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock() // defer гарантирует разблокировку даже при panic

	v, ok := rl.visitors[ip]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(rl.limit, rl.burst)}
		rl.visitors[ip] = v
	}
	v.lastSeen = time.Now()
	return v.limiter
}

func (rl *RateLimiter) cleanup(interval time.Duration) {
	for {
		time.Sleep(interval)
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > interval*2 {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware возвращает http.Handler с rate limiting.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		// Проверяем X-Forwarded-For — за nginx/OLS реальный IP будет там
		if forwarded := r.Header.Get("X-Real-IP"); forwarded != "" {
			ip = forwarded
		}

		if !rl.getVisitor(ip).Allow() {
			http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
