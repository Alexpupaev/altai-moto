# ALTAI — Прокат мотоциклов

Одностраничный лендинг для сервиса проката мотоциклов на Алтае.

**Стек:** Vite 5 · Tailwind CSS 3 · Vanilla JS (ES Modules)

---

## Быстрый старт

```bash
make up        # запустить dev-сервер → http://localhost:3000
make down       # остановить
```

### Все команды

```
make up     Запустить dev-сервер в фоне
make down       Остановить контейнеры
make logs       Показать логи контейнера
make build      Собрать продакшен-сборку в ./dist
make preview    Предпросмотр продакшен-сборки
make install    Пересобрать образ и установить зависимости заново
make shell      Открыть shell внутри контейнера
make clean      Удалить dist, node_modules и Docker-образ
```

Или просто `make` — выведет список с описаниями.

---

### Локально без Docker (Node.js ≥ 18)

```bash
npm install
npm run dev
```

---

## Структура проекта

```
.
├── index.html              # Разметка страницы
├── src/
│   ├── main.js             # Точка входа — импорт CSS и инициализация модулей
│   ├── style.css           # Tailwind + кастомные компоненты
│   └── modules/
│       ├── nav.js          # Мобильный drawer-навигация
│       ├── calendar.js     # Интерактивный календарь доступности
│       └── form.js         # Маска телефона, валидация, отправка заявки
├── tailwind.config.js      # Цветовая палитра и типографика
├── postcss.config.js
├── vite.config.js
├── Dockerfile
└── docker-compose.yml
```

---

## Сборка для продакшена

```bash
# В Docker
docker compose run --rm app npm run build

# Локально
npm run build
```

Результат — папка `dist/`. Содержимое можно деплоить на любой статический хостинг (Nginx, Caddy, GitHub Pages, Netlify и т.д.).

Предпросмотр собранной версии:

```bash
npm run preview
```

---

## Разработка

### Занятые даты в календаре

Отредактировать массив `BOOKED_RANGES` в `src/modules/calendar.js`:

```js
// [год, месяц (0 = январь), день начала, день конца]
const BOOKED_RANGES = [
  [2026, 3, 5, 7],   // апрель 5–7
  ...
];
```

### Подключение формы к бэкенду

В `src/modules/form.js` найти `setTimeout(() => showSuccess(form), 1200)` и заменить на `fetch`:

```js
const res = await fetch('/api/booking', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ name, phone, dateFrom, dateTo }),
});
if (res.ok) showSuccess(form);
```
