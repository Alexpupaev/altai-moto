export function initNav() {
  const menuBtn  = document.getElementById('nav-menu-btn');
  const closeBtn = document.getElementById('nav-close-btn');
  const overlay  = document.getElementById('nav-overlay');
  const drawer   = document.getElementById('nav-drawer');
  const navLinks = drawer?.querySelectorAll('a') ?? [];

  function openDrawer() {
    overlay.classList.remove('hidden');
    // Allow display:block to paint before adding opacity for transition
    requestAnimationFrame(() => {
      overlay.classList.add('opacity-100');
      drawer.classList.remove('translate-x-full');
    });
    document.body.style.overflow = 'hidden';
  }

  function closeDrawer() {
    overlay.classList.remove('opacity-100');
    drawer.classList.add('translate-x-full');
    setTimeout(() => overlay.classList.add('hidden'), 300);
    document.body.style.overflow = '';
  }

  menuBtn?.addEventListener('click', openDrawer);
  closeBtn?.addEventListener('click', closeDrawer);
  overlay?.addEventListener('click', closeDrawer);
  navLinks.forEach(link => link.addEventListener('click', closeDrawer));
}
