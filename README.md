# ALTAI MOTO — Прокат мотоциклов на Алтае

Одностраничный лендинг с формой заявки, интерактивным календарём доступности и уведомлениями в Telegram.

**Стек:** Vite 5 · Vanilla JS (ES Modules) · Go 1.24 · SQLite · Docker Compose

---

## Быстрый старт (dev)

```bash
cp .env.example .env   # заполнить токены Telegram
make dev-up            # запустить → http://localhost
make logs              # логи всех контейнеров
make down              # остановить
```

Или `make` — выведет список всех команд.

---

## Структура проекта

```
.
├── frontend/
│   ├── index.html
│   ├── src/
│   │   ├── main.js
│   │   └── modules/
│   │       ├── nav.js          # мобильная навигация
│   │       ├── calendar.js     # календарь доступности
│   │       └── form.js         # валидация и отправка заявки
│   └── public/
│       ├── favicon.svg
│       └── images/
├── backend/
│   ├── cmd/server/main.go      # точка входа, роутер
│   └── internal/
│       ├── handler/            # HTTP-обработчики
│       ├── notify/             # Telegram Bot API
│       ├── storage/            # SQLite
│       └── middleware/         # rate limiter
├── develop/                    # dev Docker Compose + nginx
├── production/                 # prod Docker Compose + nginx
├── dockerfiles/                # Dockerfile для frontend и backend
├── _db/                        # SQLite база (не коммитится)
├── .env.example
└── Makefile
```

---

## Переменные окружения

Скопировать `.env.example` → `.env` и заполнить:

```env
TELEGRAM_BOT_TOKEN=<токен от @BotFather>
TELEGRAM_CHAT_ID=<ID чата или группы>
TELEGRAM_WEBHOOK_SECRET=<произвольная строка, опционально>
DB_PATH=/data/bookings.db
```

---

## API

| Метод | Путь | Описание |
|-------|------|----------|
| `GET` | `/api/bookings` | Занятые диапазоны дат `[{date_from, date_to}]` |
| `POST` | `/api/booking` | Создать заявку |
| `POST` | `/telegram/webhook` | Webhook для Telegram (callback-кнопки) |

### POST /api/booking

```json
{
  "name": "Иван",
  "phone": "+7 999 123-45-67",
  "date_from": "2026-06-01",
  "date_to": "2026-06-05",
  "website": ""
}
```

Поле `website` — honeypot для ботов.

---

## Telegram

При новой заявке бот отправляет сообщение с inline-кнопками:
- **Подтвердить** / **Отклонить** (для новых заявок)
- Подтверждённую можно отклонить или вернуть в ожидание — и наоборот

При нажатии кнопки статус обновляется в БД и сообщение редактируется.

### Регистрация webhook

```bash
curl "https://api.telegram.org/bot<TOKEN>/setWebhook?url=https://твой-домен/telegram/webhook"
```

---

## Деплой (production)

```bash
git clone <repo> && cd moto-rental
cp .env.example .env && nano .env
make prod-build
make prod-up
```

Nginx слушает на `127.0.0.1:8888`. Настроить reverse proxy в CyberPanel:
`altaymoto.rockhockey.ru` → `http://127.0.0.1:8888`

SSL выдаётся через CyberPanel (Let's Encrypt).

---

## Автозапуск после перезагрузки сервера

Выполнить один раз от root на сервере:

```bash
# заменить /path/to/moto-rental на реальный путь к проекту
sed 's|{{PROJECT_DIR}}|/path/to/moto-rental|g' \
    /path/to/moto-rental/production/moto-rental.service \
    > /etc/systemd/system/moto-rental.service

systemctl daemon-reload
systemctl enable moto-rental
systemctl start moto-rental
```

После этого `docker compose up` запускается автоматически при старте ОС.

```bash
systemctl status moto-rental   # статус
systemctl disable moto-rental  # отключить автозапуск
```

---

## Доступ к базе данных

В production доступен веб-интерфейс [sqlite-web](https://github.com/coleifer/sqlite-web):

```
https://altaymoto.rockhockey.ru/database/
```

Защищён HTTP Basic Auth. Логин и пароль задаются в `.env`:

```env
DB_ADMIN_USER=admin
DB_ADMIN_PASSWORD=supersecret
```

При `make prod-up` nginx генерирует htpasswd-файл из этих переменных автоматически.
