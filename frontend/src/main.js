import { initNav }      from './modules/nav.js';
import { initCalendar } from './modules/calendar.js';
import { initForm }     from './modules/form.js';

document.getElementById('footer-year').textContent = new Date().getFullYear();

initNav();
initCalendar();
initForm();
