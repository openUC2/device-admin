export function loadThemeSetting() {
  const theme = window.localStorage.getItem('theme');
  switch (theme) {
    case 'dark':
      return 'dark';
    default:
      return 'light';
  }
}
export function storeThemeSetting(theme) {
  window.localStorage.setItem('theme', theme);
}
export function invert(theme) {
  switch (theme) {
    case 'dark':
      return 'light';
    default:
      return 'dark';
  }
}
function setThemeProperties(stylesheetID, theme) {
  const stylesheet = document.querySelector(`link#${stylesheetID}`);
  if (stylesheet === null || !(stylesheet instanceof HTMLLinkElement)) {
    return;
  }

  const active = stylesheet.id === `${theme}-theme`;
  stylesheet.rel = active ? 'stylesheet' : 'stylesheet alternate';
}
export function setTheme(theme) {
  setThemeProperties('light-theme', theme);
  setThemeProperties('dark-theme', theme);
}

export function onLoad(themeOverride) {
  const theme = themeOverride || loadThemeSetting();
  if (theme === 'dark') {
    // Set an initial color to prevent a white flash when dark mode has been set
    document.documentElement.classList.add('initial-load-dark');
    const darkStylesheet = document.querySelector('link#dark-theme');
    if (darkStylesheet !== null && darkStylesheet instanceof HTMLLinkElement) {
      darkStylesheet.addEventListener('load', () => {
        document.documentElement.classList.remove('initial-load-dark');
      });
    }
  }
  setTheme(theme);
  if (!themeOverride) {
    storeThemeSetting(theme);
  }
}
