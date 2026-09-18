(() => {
  try {
    const theme = localStorage.getItem("tally-theme");
    if (theme === "light" || theme === "dark" || theme === "system") {
      document.documentElement.dataset.theme = theme;
    }
  } catch {
    // Browser storage can be unavailable; the document's system default remains safe.
  }
})();
