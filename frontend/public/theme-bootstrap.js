(() => {
  try {
    // Production HTML is personalized by the server from the active profile.
    // Keep that choice; localStorage may belong to a different profile/browser view.
    const serverTheme = document.documentElement.dataset.theme;
    if (serverTheme === "light" || serverTheme === "dark" || serverTheme === "system") {
      return;
    }
    const theme = localStorage.getItem("tally-theme");
    if (theme === "light" || theme === "dark" || theme === "system") {
      document.documentElement.dataset.theme = theme;
    }
  } catch {
    // Browser storage can be unavailable; the document's system default remains safe.
  }
})();
