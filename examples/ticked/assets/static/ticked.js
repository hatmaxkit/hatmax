document.addEventListener('DOMContentLoaded', () => {
    const themeToggle = document.getElementById('theme-toggle');
    const body = document.body;

    // Check for saved theme preference or default to light
    const currentTheme = localStorage.getItem('theme') || 'light';
    if (currentTheme === 'dark') {
        body.classList.add('dark-mode');
        if (themeToggle) themeToggle.textContent = '☀️';
    }

    // Toggle theme function
    function toggleTheme() {
        body.classList.toggle('dark-mode');
        const isDark = body.classList.contains('dark-mode');

        // Update button icon
        if (themeToggle) {
            themeToggle.textContent = isDark ? '☀️' : '🌙';
        }

        // Save preference
        localStorage.setItem('theme', isDark ? 'dark' : 'light');
    }

    // Attach click handler
    if (themeToggle) {
        themeToggle.addEventListener('click', toggleTheme);
    }

    // Keyboard shortcut: Alt+D
    document.addEventListener('keydown', (event) => {
        if (event.altKey && event.key.toLowerCase() === 'd') {
            event.preventDefault();
            toggleTheme();
        }
    });
});
