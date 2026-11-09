// Runs before first paint so the page never flashes the wrong theme.
(function () {
  var d = document.documentElement;
  var pref = null;
  try { pref = localStorage.getItem('ds-theme'); } catch (e) {}
  var t = pref || d.getAttribute('data-theme-default') || 'system';
  if (t !== 'dark' && t !== 'light') {
    t = window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
  }
  d.setAttribute('data-theme', t);
})();
