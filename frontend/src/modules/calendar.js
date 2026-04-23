const MONTH_NAMES = [
  'Январь', 'Февраль', 'Март', 'Апрель', 'Май', 'Июнь',
  'Июль', 'Август', 'Сентябрь', 'Октябрь', 'Ноябрь', 'Декабрь',
];

// Only these months are available for rental (0-indexed: May=4, Jun=5, Jul=6, Sep=8)
const ALLOWED_MONTHS = [4, 5, 6, 8];

function nextAllowed(year, month) {
  const next = ALLOWED_MONTHS.find(m => m > month);
  if (next !== undefined) return { year, month: next };
  return { year: year + 1, month: ALLOWED_MONTHS[0] };
}

function prevAllowed(year, month) {
  const prev = [...ALLOWED_MONTHS].reverse().find(m => m < month);
  if (prev !== undefined) return { year, month: prev };
  return { year: year - 1, month: ALLOWED_MONTHS[ALLOWED_MONTHS.length - 1] };
}

function nearestAllowed(year, month) {
  if (ALLOWED_MONTHS.includes(month)) return { year, month };
  const next = ALLOWED_MONTHS.find(m => m > month);
  if (next !== undefined) return { year, month: next };
  return { year: year + 1, month: ALLOWED_MONTHS[0] };
}

/** @type {{from: Date, to: Date}[]} */
let _bookedRanges = [];

async function loadBookedRanges() {
  try {
    const res = await fetch('/api/bookings');
    if (!res.ok) return;
    const data = await res.json();
    _bookedRanges = (data ?? []).map(({ date_from, date_to }) => ({
      from: new Date(date_from + 'T00:00:00'),
      to:   new Date(date_to   + 'T00:00:00'),
    }));
  } catch {
    // network error — calendar shows all days as available
  }
}

// ─── Module state ────────────────────────────────────────────────────────────

let _gridEl   = null;
let _titleEl  = null;
let _year     = 0;
let _month    = 0;
let _minYear  = 0;
let _minMonth = 0;

/** @type {Date|null} */
let _fromDate = null;
/** @type {Date|null} */
let _toDate   = null;

// ─── Helpers ─────────────────────────────────────────────────────────────────

function toIso(date) {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, '0');
  const d = String(date.getDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
}

/** Compare two Date objects by date only (ignore time). */
function sameDay(a, b) {
  return a && b &&
    a.getFullYear() === b.getFullYear() &&
    a.getMonth()    === b.getMonth()    &&
    a.getDate()     === b.getDate();
}

function todayMidnight() {
  const d = new Date();
  d.setHours(0, 0, 0, 0);
  return d;
}

function isBooked(year, month, day) {
  const d = new Date(year, month, day);
  return _bookedRanges.some(({ from, to }) => d >= from && d <= to);
}

/** Returns true if any booked range overlaps strictly between dateA and dateB. */
function hasBookedDayBetween(dateA, dateB) {
  const [start, end] = dateA < dateB ? [dateA, dateB] : [dateB, dateA];
  return _bookedRanges.some(({ from, to }) => from < end && to > start);
}

function pluralDays(n) {
  const mod10  = n % 10;
  const mod100 = n % 100;
  if (mod10 === 1 && mod100 !== 11)                            return 'день';
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20)) return 'дня';
  return 'дней';
}

// ─── Render ───────────────────────────────────────────────────────────────────

function getDayClass(date, isPast, booked) {
  const isFrom    = _fromDate && sameDay(date, _fromDate);
  const isTo      = _toDate   && sameDay(date, _toDate);
  const isToday   = sameDay(date, todayMidnight());
  const inRange   = _fromDate && _toDate && date > _fromDate && date < _toDate;

  if (isPast)   return 'cd cd-past';
  if (booked)   return 'cd cd-bk';
  if (isFrom)   return 'cd cd-from';
  if (isTo)     return 'cd cd-to';
  if (inRange)  return 'cd cd-range';
  if (isToday)  return 'cd cd-today';
  return 'cd cd-free';
}

function renderGrid() {
  const today        = todayMidnight();
  const firstDayOfWeek = new Date(_year, _month, 1).getDay();
  const daysInMonth    = new Date(_year, _month + 1, 0).getDate();
  const startOffset    = (firstDayOfWeek + 6) % 7; // Monday-first

  const cells = [];

  for (let i = 0; i < startOffset; i++) {
    cells.push('<div class="aspect-square"></div>');
  }

  for (let day = 1; day <= daysInMonth; day++) {
    const date    = new Date(_year, _month, day);
    const isPast  = date < today;
    const booked  = isBooked(_year, _month, day);
    const cls     = getDayClass(date, isPast, booked);
    const clickable = !isPast && !booked;

    cells.push(
      `<div class="${cls}"${clickable ? ` data-day="${day}"` : ''}>${day}</div>`,
    );
  }

  _gridEl.innerHTML = cells.join('');
}

function updateStatus() {
  const el = document.getElementById('cal-status');
  if (!el) return;

  if (!_fromDate) {
    el.textContent = 'Нажмите на дату начала аренды';
    el.className = 'text-xs text-on-surface-variant/60 text-center mt-4 min-h-[1.25rem]';
    return;
  }
  if (!_toDate) {
    const from = _fromDate.toLocaleDateString('ru-RU', { day: 'numeric', month: 'long' });
    el.textContent = `Начало: ${from} — выберите дату окончания`;
    el.className = 'text-xs text-primary text-center mt-4 min-h-[1.25rem]';
    return;
  }
  const from = _fromDate.toLocaleDateString('ru-RU', { day: 'numeric', month: 'long' });
  const to   = _toDate.toLocaleDateString('ru-RU',   { day: 'numeric', month: 'long' });
  const days = Math.round((_toDate - _fromDate) / 86_400_000);
  el.textContent = `${from} — ${to} · ${days} ${pluralDays(days)}`;
  el.className = 'text-xs text-primary font-semibold text-center mt-4 min-h-[1.25rem]';
}

function dispatchSelection() {
  document.dispatchEvent(new CustomEvent('booking:dates-selected', {
    detail: {
      from: _fromDate ? toIso(_fromDate) : null,
      to:   _toDate   ? toIso(_toDate)   : null,
    },
  }));
}

// ─── Public API ──────────────────────────────────────────────────────────────

/**
 * Called by form.js when the user edits date inputs directly.
 * Does NOT fire 'booking:dates-selected' to avoid loops.
 */
export function setSelection(fromIso, toIso) {
  _fromDate = fromIso ? new Date(fromIso + 'T00:00:00') : null;
  _toDate   = toIso   ? new Date(toIso   + 'T00:00:00') : null;

  // Navigate to the fromDate month so the selection is visible
  if (_fromDate) {
    ({ year: _year, month: _month } = nearestAllowed(_fromDate.getFullYear(), _fromDate.getMonth()));
    _titleEl.textContent = `${MONTH_NAMES[_month]} ${_year}`;
  }

  renderGrid();
  updateStatus();
}

export async function initCalendar() {
  _titleEl       = document.getElementById('cal-title');
  _gridEl        = document.getElementById('cal-grid');
  const prevBtn  = document.getElementById('cal-prev');
  const nextBtn  = document.getElementById('cal-next');

  if (!_titleEl || !_gridEl || !prevBtn || !nextBtn) return;

  const now = new Date();
  ({ year: _year, month: _month } = nearestAllowed(now.getFullYear(), now.getMonth()));
  _minYear  = _year;
  _minMonth = _month;
  const _maxYear  = _minYear;
  const _maxMonth = ALLOWED_MONTHS[ALLOWED_MONTHS.length - 1];

  function isAtMin() { return _year === _minYear && _month === _minMonth; }
  function isAtMax() { return _year === _maxYear && _month === _maxMonth; }

  function updateNavBtns() {
    const atMin = isAtMin();
    prevBtn.disabled = atMin;
    prevBtn.style.opacity = atMin ? '0.3' : '';
    prevBtn.style.cursor  = atMin ? 'default' : '';

    const atMax = isAtMax();
    nextBtn.disabled = atMax;
    nextBtn.style.opacity = atMax ? '0.3' : '';
    nextBtn.style.cursor  = atMax ? 'default' : '';
  }

  function refreshTitle() {
    _titleEl.textContent = `${MONTH_NAMES[_month]} ${_year}`;
  }

  prevBtn.addEventListener('click', () => {
    if (isAtMin()) return;
    ({ year: _year, month: _month } = prevAllowed(_year, _month));
    refreshTitle();
    renderGrid();
    updateNavBtns();
  });

  nextBtn.addEventListener('click', () => {
    if (isAtMax()) return;
    ({ year: _year, month: _month } = nextAllowed(_year, _month));
    refreshTitle();
    renderGrid();
    updateNavBtns();
  });

  // Day click — datepicker logic
  _gridEl.addEventListener('click', (e) => {
    const dayEl = e.target.closest('[data-day]');
    if (!dayEl) return;

    const day     = parseInt(dayEl.dataset.day, 10);
    const clicked = new Date(_year, _month, day);

    if (!_fromDate || (_fromDate && _toDate)) {
      // Start a fresh selection
      _fromDate = clicked;
      _toDate   = null;
    } else if (sameDay(clicked, _fromDate)) {
      // Tap the same day again → deselect
      _fromDate = null;
      _toDate   = null;
    } else if (clicked > _fromDate) {
      if (hasBookedDayBetween(_fromDate, clicked)) {
        // Range crosses a booking — restart selection from this day
        _fromDate = clicked;
        _toDate   = null;
      } else {
        _toDate = clicked;
      }
    } else {
      // Clicked before fromDate → swap
      if (hasBookedDayBetween(clicked, _fromDate)) {
        _fromDate = clicked;
        _toDate   = null;
      } else {
        _toDate   = _fromDate;
        _fromDate = clicked;
      }
    }

    renderGrid();
    updateStatus();
    dispatchSelection();
  });

  await loadBookedRanges();
  refreshTitle();
  renderGrid();
  updateStatus();
  updateNavBtns();
}
