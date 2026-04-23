// ─── Phone mask ──────────────────────────────────────────────────────────────

function applyPhoneMask(e) {
  const input  = e.target;
  let digits = input.value.replace(/\D/g, '');

  if (digits.length === 0) { input.value = ''; return; }
  if (digits[0] === '8') digits = '7' + digits.slice(1);
  if (digits[0] !== '7') digits = '7' + digits;
  digits = digits.slice(0, 11);

  let result = '+7';
  if (digits.length > 1) result += ' (' + digits.slice(1, 4);
  if (digits.length >= 4) result += ') ' + digits.slice(4, 7);
  if (digits.length >= 7) result += '-'  + digits.slice(7, 9);
  if (digits.length >= 9) result += '-'  + digits.slice(9, 11);

  input.value = result;
}

// ─── Field error helpers ─────────────────────────────────────────────────────

function showFieldError(fieldEl, message) {
  clearFieldError(fieldEl);
  fieldEl.classList.add('err');

  const err = document.createElement('p');
  err.className = 'err-msg';
  err.textContent = message;
  fieldEl.parentElement.appendChild(err);

  fieldEl.addEventListener('input', () => clearFieldError(fieldEl), { once: true });
}

function clearFieldError(fieldEl) {
  fieldEl.classList.remove('err');
  fieldEl.parentElement.querySelector('.err-msg')?.remove();
}

// ─── Validation ──────────────────────────────────────────────────────────────

function validate(fields) {
  const { name, phone, dateFrom, dateTo } = fields;

  if (!name.value.trim() || name.value.trim().length < 2) {
    showFieldError(name, 'Введите ваше имя');
    name.focus();
    return false;
  }
  if (phone.value.replace(/\D/g, '').length < 11) {
    showFieldError(phone, 'Введите корректный номер (+7 XXX XXX-XX-XX)');
    phone.focus();
    return false;
  }
  if (!dateFrom.value) {
    showFieldError(dateFrom, 'Выберите дату начала');
    dateFrom.focus();
    return false;
  }
  if (!dateTo.value) {
    showFieldError(dateTo, 'Выберите дату окончания');
    dateTo.focus();
    return false;
  }
  if (dateTo.value <= dateFrom.value) {
    showFieldError(dateTo, 'Дата окончания должна быть позже начала');
    dateTo.focus();
    return false;
  }
  return true;
}

// ─── Success state ───────────────────────────────────────────────────────────

function showSuccess(form) {
  form.innerHTML = `
    <div style="text-align:center;padding:4rem 0;">
      <div style="width:4rem;height:4rem;background:#E6F4F3;border-radius:50%;display:flex;align-items:center;justify-content:center;margin:0 auto 1.5rem;font-size:1.5rem;color:var(--teal);">✓</div>
      <h3 class="bc" style="font-size:2rem;font-weight:800;text-transform:uppercase;margin-bottom:.75rem;">Заявка отправлена</h3>
      <p style="color:var(--mid);font-size:.9rem;">Свяжемся в течение 30 минут.</p>
    </div>
  `;
}

// ─── Toast ───────────────────────────────────────────────────────────────────

function showToast(message) {
  document.getElementById('toast')?.remove();

  const toast = document.createElement('div');
  toast.id = 'toast';
  toast.style.cssText = 'position:fixed;bottom:1.5rem;left:50%;transform:translateX(-50%);z-index:50;background:var(--ink);color:var(--cream);font-size:.875rem;font-weight:500;padding:.75rem 1.5rem;border-radius:2rem;box-shadow:0 4px 16px rgba(0,0,0,.2);transition:opacity .3s;opacity:0;white-space:nowrap;';
  toast.textContent = message;
  document.body.appendChild(toast);

  requestAnimationFrame(() => requestAnimationFrame(() => toast.style.opacity = '1'));

  setTimeout(() => {
    toast.style.opacity = '0';
    toast.addEventListener('transitionend', () => toast.remove(), { once: true });
  }, 3000);
}

// ─── Init ────────────────────────────────────────────────────────────────────

export function initForm() {
  const form      = document.getElementById('booking-form');
  const nameEl    = document.getElementById('field-name');
  const phone     = document.getElementById('field-phone');
  const dateFrom  = document.getElementById('field-date-from');
  const dateTo    = document.getElementById('field-date-to');
  const submitBtn = document.getElementById('booking-submit');

  if (!form) return;

  // Date fields are readonly — values are set only via the calendar widget.

  // ── Sync: calendar → form inputs ──────────────────────────────────────────
  // Assigning .value programmatically does NOT fire the 'change' event,
  // so there is no feedback loop here.
  document.addEventListener('booking:dates-selected', (e) => {
    const { from, to } = e.detail;

    if (dateFrom) {
      dateFrom.value = from ?? '';
      clearFieldError(dateFrom);
    }
    if (dateTo) {
      dateTo.value = to ?? '';
      if (from) dateTo.setAttribute('min', from);
      clearFieldError(dateTo);
    }

    // Scroll form into view when both dates are selected
    if (from && to) {
      document.getElementById('booking')?.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
  });

  // ── Phone mask ─────────────────────────────────────────────────────────────
  phone?.addEventListener('input', applyPhoneMask);

  // ── Submit ─────────────────────────────────────────────────────────────────
  form.addEventListener('submit', async (e) => {
    e.preventDefault();

    if (!validate({ name: nameEl, phone, dateFrom, dateTo })) return;

    submitBtn.disabled = true;
    submitBtn.textContent = 'Отправляем…';

    try {
      const res = await fetch('/api/booking', {
        method:  'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name:      nameEl.value.trim(),
          phone:     phone.value,
          date_from: dateFrom.value,
          date_to:   dateTo.value,
          website:   '',
        }),
      });

      if (res.ok) {
        showSuccess(form);
        return;
      }

      if (res.status === 422) {
        const { errors } = await res.json();
        if (errors.name)      showFieldError(nameEl,   errors.name);
        if (errors.phone)     showFieldError(phone,    errors.phone);
        if (errors.date_from) showFieldError(dateFrom, errors.date_from);
        if (errors.date_to)   showFieldError(dateTo,   errors.date_to);
        return;
      }

      if (res.status === 429) {
        showToast('Слишком много попыток. Попробуйте через 15 минут.');
        return;
      }

      showToast('Ошибка сервера. Попробуйте позже.');
    } catch {
      showToast('Нет соединения. Проверьте интернет.');
    } finally {
      submitBtn.disabled = false;
      submitBtn.textContent = 'Отправить заявку';
    }
  });

  // ── Download rules placeholder ─────────────────────────────────────────────
  document.getElementById('btn-download-rules')?.addEventListener('click', () => {
    showToast('PDF с правилами будет доступен в ближайшее время');
  });
}
