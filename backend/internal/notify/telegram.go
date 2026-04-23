package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/altai-moto/moto-rental/internal/storage"
)

type Notifier struct {
	token      string
	chatID     string
	client     *http.Client
	pollClient *http.Client
}

func New(token, chatID string) *Notifier {
	return &Notifier{
		token:      token,
		chatID:     chatID,
		client:     &http.Client{Timeout: 10 * time.Second},
		pollClient: &http.Client{Timeout: 35 * time.Second},
	}
}

type Update struct {
	UpdateID      int64     `json:"update_id"`
	CallbackQuery *Callback `json:"callback_query"`
}

type Callback struct {
	ID      string  `json:"id"`
	Data    string  `json:"data"`
	Message Message `json:"message"`
}

type Message struct {
	MessageID int64 `json:"message_id"`
}

func (n *Notifier) DeleteWebhook() error {
	return n.call("deleteWebhook", map[string]any{"drop_pending_updates": false}, nil)
}

func (n *Notifier) GetUpdates(ctx context.Context, offset int64) ([]Update, error) {
	body, err := json.Marshal(map[string]any{"timeout": 30, "offset": offset})
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", n.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.pollClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getUpdates: %w", err)
	}
	defer resp.Body.Close()

	var base struct {
		OK          bool            `json:"ok"`
		Description string          `json:"description"`
		Result      json.RawMessage `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&base); err != nil {
		return nil, fmt.Errorf("getUpdates decode: %w", err)
	}
	if !base.OK {
		return nil, fmt.Errorf("getUpdates: %s", base.Description)
	}

	var updates []Update
	if err := json.Unmarshal(base.Result, &updates); err != nil {
		return nil, fmt.Errorf("getUpdates unmarshal: %w", err)
	}
	return updates, nil
}

type inlineButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

func buildKeyboard(bookingID int64, status string) [][]inlineButton {
	confirm := inlineButton{"✅ Подтвердить", fmt.Sprintf("confirm:%d", bookingID)}
	reject := inlineButton{"❌ Отклонить", fmt.Sprintf("reject:%d", bookingID)}
	pending := inlineButton{"🔄 В ожидание", fmt.Sprintf("pending:%d", bookingID)}

	switch status {
	case "confirmed":
		return [][]inlineButton{{reject, pending}}
	case "rejected":
		return [][]inlineButton{{confirm, pending}}
	default:
		return [][]inlineButton{{confirm, reject}}
	}
}

func buildText(b storage.Booking) string {
	label := map[string]string{
		"pending":   "🕐 Новая заявка",
		"confirmed": "✅ Подтверждена",
		"rejected":  "❌ Отклонена",
	}[b.Status]

	days := daysBetween(b.DateFrom, b.DateTo)
	return fmt.Sprintf(
		"<b>%s #%d</b>\n\n👤 %s\n📞 %s\n📅 %s — %s\n⏱ %d %s",
		label, b.ID, b.Name, b.Phone, b.DateFrom, b.DateTo, days, pluralDays(days),
	)
}

func daysBetween(from, to string) int {
	f, err1 := time.Parse("2006-01-02", from)
	t, err2 := time.Parse("2006-01-02", to)
	if err1 != nil || err2 != nil {
		return 0
	}
	return int(t.Sub(f).Hours() / 24)
}

func pluralDays(n int) string {
	m10, m100 := n%10, n%100
	if m10 == 1 && m100 != 11 {
		return "день"
	}
	if m10 >= 2 && m10 <= 4 && (m100 < 10 || m100 >= 20) {
		return "дня"
	}
	return "дней"
}

// Send отправляет новую заявку и возвращает ID сообщения в Telegram.
func (n *Notifier) Send(b storage.Booking) (int64, error) {
	if n.token == "" {
		return 0, nil
	}

	var result struct {
		MessageID int64 `json:"message_id"`
	}

	err := n.call("sendMessage", map[string]any{
		"chat_id":    n.chatID,
		"text":       buildText(b),
		"parse_mode": "HTML",
		"reply_markup": map[string]any{
			"inline_keyboard": buildKeyboard(b.ID, b.Status),
		},
	}, &result)
	if err != nil {
		return 0, err
	}
	return result.MessageID, nil
}

// EditMessage обновляет текст и кнопки существующего сообщения.
func (n *Notifier) EditMessage(msgID int64, b storage.Booking) error {
	if n.token == "" || msgID == 0 {
		return nil
	}
	return n.call("editMessageText", map[string]any{
		"chat_id":    n.chatID,
		"message_id": msgID,
		"text":       buildText(b),
		"parse_mode": "HTML",
		"reply_markup": map[string]any{
			"inline_keyboard": buildKeyboard(b.ID, b.Status),
		},
	}, nil)
}

// AnswerCallback убирает индикатор загрузки на кнопке.
func (n *Notifier) AnswerCallback(callbackID, text string) error {
	if n.token == "" {
		return nil
	}
	return n.call("answerCallbackQuery", map[string]any{
		"callback_query_id": callbackID,
		"text":              text,
	}, nil)
}

func (n *Notifier) call(method string, payload any, result any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/%s", n.token, method)
	resp, err := n.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram %s: %w", method, err)
	}
	defer resp.Body.Close()

	var base struct {
		OK          bool            `json:"ok"`
		Description string          `json:"description"`
		Result      json.RawMessage `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&base); err != nil {
		return fmt.Errorf("telegram %s decode: %w", method, err)
	}
	if !base.OK {
		return fmt.Errorf("telegram %s: %s", method, base.Description)
	}
	if result != nil {
		return json.Unmarshal(base.Result, result)
	}
	return nil
}
