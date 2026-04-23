package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // side-effect import: регистрирует SQLite-драйвер
	// в PHP аналог: require 'driver.php', где файл просто вызывает register_driver()
)

// Booking — структура данных, аналог класса-модели в PHP.
// Теги `db:"..."` используются для маппинга колонок (как в Eloquent/Doctrine).
type Booking struct {
	ID          int64     `db:"id"`
	Name        string    `db:"name"`
	Phone       string    `db:"phone"`
	DateFrom    string    `db:"date_from"` // формат YYYY-MM-DD
	DateTo      string    `db:"date_to"`
	Status      string    `db:"status"` // pending | confirmed | rejected
	TgMessageID int64     `db:"tg_message_id"`
	CreatedAt   time.Time `db:"created_at"`
}

// DateRange — урезанная структура для публичного API (без личных данных).
// Теги `json:"..."` управляют сериализацией в JSON.
type DateRange struct {
	DateFrom string `json:"date_from"`
	DateTo   string `json:"date_to"`
}

// DB — обёртка над подключением к базе данных.
// В Go нет классов, но структура + методы дают то же самое.
type DB struct {
	conn *sql.DB
}

// New открывает БД и создаёт таблицу если её нет.
// Возвращает (*DB, error) — Go-идиома: всегда возвращаем ошибку явно,
// никаких исключений. В PHP это было бы try/catch PDOException.
func New(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
		// %w — "wrap error", сохраняет цепочку для errors.Is/errors.As
	}

	// Проверяем реальное подключение (sql.Open ленивый — ещё не подключается)
	if err = conn.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	if err = migrate(conn); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &DB{conn: conn}, nil
}

// migrate создаёт схему. Вынесено отдельно для читаемости.
func migrate(conn *sql.DB) error {
	_, err := conn.Exec(`
		CREATE TABLE IF NOT EXISTS bookings (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			name           TEXT    NOT NULL,
			phone          TEXT    NOT NULL,
			date_from      TEXT    NOT NULL,
			date_to        TEXT    NOT NULL,
			status         TEXT    NOT NULL DEFAULT 'pending',
			tg_message_id  INTEGER NOT NULL DEFAULT 0,
			created_at     TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		)
	`)
	if err != nil {
		return err
	}
	// для уже существующих баз без колонки
	conn.Exec(`ALTER TABLE bookings ADD COLUMN tg_message_id INTEGER NOT NULL DEFAULT 0`)
	return nil
}

// GetBooking возвращает заявку по ID.
func (db *DB) GetBooking(id int64) (Booking, error) {
	var b Booking
	err := db.conn.QueryRow(`
		SELECT id, name, phone, date_from, date_to, status, tg_message_id
		FROM bookings WHERE id = ?
	`, id).Scan(&b.ID, &b.Name, &b.Phone, &b.DateFrom, &b.DateTo, &b.Status, &b.TgMessageID)
	if err != nil {
		return Booking{}, fmt.Errorf("get booking: %w", err)
	}
	return b, nil
}

// UpdateBookingStatus меняет статус заявки.
func (db *DB) UpdateBookingStatus(id int64, status string) error {
	_, err := db.conn.Exec(`UPDATE bookings SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

// SetTgMessageID сохраняет ID сообщения в Telegram для последующего редактирования.
func (db *DB) SetTgMessageID(id, msgID int64) error {
	_, err := db.conn.Exec(`UPDATE bookings SET tg_message_id = ? WHERE id = ?`, msgID, id)
	if err != nil {
		return fmt.Errorf("set tg_message_id: %w", err)
	}
	return nil
}

// GetBookedRanges возвращает только занятые диапазоны дат — без личных данных.
// Публикуем во фронтенд, поэтому PersonalData не включаем.
func (db *DB) GetBookedRanges() ([]DateRange, error) {
	rows, err := db.conn.Query(`
		SELECT date_from, date_to
		FROM   bookings
		WHERE  status = 'confirmed'
		ORDER  BY date_from
	`)
	if err != nil {
		return nil, fmt.Errorf("query booked ranges: %w", err)
	}
	defer rows.Close() // defer — выполнится при выходе из функции, как __destruct

	var ranges []DateRange
	for rows.Next() {
		var r DateRange
		if err := rows.Scan(&r.DateFrom, &r.DateTo); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		ranges = append(ranges, r)
	}

	// ranges == nil если строк нет — вернём пустой slice для корректного JSON ([])
	if ranges == nil {
		ranges = []DateRange{}
	}

	return ranges, nil
}

// SaveBooking записывает новую заявку и возвращает её ID.
func (db *DB) SaveBooking(b Booking) (int64, error) {
	result, err := db.conn.Exec(`
		INSERT INTO bookings (name, phone, date_from, date_to)
		VALUES (?, ?, ?, ?)
	`, b.Name, b.Phone, b.DateFrom, b.DateTo)
	if err != nil {
		return 0, fmt.Errorf("insert booking: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}

	return id, nil
}
