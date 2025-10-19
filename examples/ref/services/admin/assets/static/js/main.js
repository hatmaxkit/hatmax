// Main JavaScript for HatMax Admin

document.addEventListener('DOMContentLoaded', function() {
    // Initialize theme toggle
    initializeThemeToggle();
    
    // Initialize HTMX
    if (typeof htmx !== 'undefined') {
        // Global HTMX configuration
        htmx.config.globalViewTransitions = true;
        htmx.config.useTemplateFragments = true;
        
        // Handle flash messages
        htmx.on('htmx:afterRequest', function(evt) {
            // Auto-hide flash messages after 5 seconds
            const flashMessages = document.querySelectorAll('.flash');
            flashMessages.forEach(function(flash) {
                setTimeout(function() {
                    flash.style.transition = 'opacity 0.3s ease';
                    flash.style.opacity = '0';
                    setTimeout(function() {
                        flash.remove();
                    }, 300);
                }, 5000);
            });
        });
        
        // Handle delete confirmations
        htmx.on('htmx:confirm', function(evt) {
            if (evt.detail.question && evt.detail.question.includes('delete')) {
                evt.preventDefault();
                if (confirm('Are you sure you want to delete this item? This action cannot be undone.')) {
                    evt.detail.issueRequest();
                }
            }
        });
        
        // Handle form validation errors
        htmx.on('htmx:responseError', function(evt) {
            showError('An error occurred while processing your request.');
        });
        
        htmx.on('htmx:sendError', function(evt) {
            showError('Unable to connect to the server. Please check your connection.');
        });
    }
    
    // Initialize other functionality
    initializeDeleteButtons();
    initializeSearchFilters();
    initializeClickableRows();
});

// Delete button confirmation
function initializeDeleteButtons() {
    document.addEventListener('click', function(e) {
        if (e.target.classList.contains('btn-danger') && !e.target.hasAttribute('hx-confirm')) {
            e.preventDefault();
            if (confirm('Are you sure you want to delete this item? This action cannot be undone.')) {
                // If using HTMX, trigger the request
                if (e.target.hasAttribute('hx-delete')) {
                    htmx.trigger(e.target, 'click');
                } else {
                    // Fallback to form submission or direct navigation
                    if (e.target.closest('form')) {
                        e.target.closest('form').submit();
                    }
                }
            }
        }
    });
}

// Make table rows clickable to view details
function initializeClickableRows() {
    document.addEventListener('click', function(e) {
        const row = e.target.closest('tbody tr');
        if (!row) return;
        
        // Don't trigger if clicking on buttons or links
        if (e.target.closest('button, a')) return;
        
        // Find the View button in the row
        const viewButton = row.querySelector('.btn-view');
        if (viewButton) {
            window.location.href = viewButton.getAttribute('href');
        }
    });
}

// Search and filter functionality
function initializeSearchFilters() {
    const searchInputs = document.querySelectorAll('input[type="search"]');
    
    searchInputs.forEach(function(input) {
        let timeout;
        input.addEventListener('input', function(e) {
            clearTimeout(timeout);
            timeout = setTimeout(function() {
                // Trigger HTMX search if configured
                if (input.hasAttribute('hx-get')) {
                    htmx.trigger(input, 'input');
                }
            }, 300); // Debounce search requests
        });
    });
}

// Utility functions
function showSuccess(message) {
    showFlash(message, 'success');
}

function showError(message) {
    showFlash(message, 'error');
}

function showWarning(message) {
    showFlash(message, 'warning');
}

function showFlash(message, type) {
    // Remove existing flash messages
    const existingFlash = document.querySelectorAll('.flash');
    existingFlash.forEach(function(flash) {
        flash.remove();
    });
    
    // Create new flash message
    const flashDiv = document.createElement('div');
    flashDiv.className = `flash flash-${type}`;
    flashDiv.textContent = message;
    
    // Insert at the beginning of main content
    const main = document.querySelector('main .container');
    if (main) {
        main.insertBefore(flashDiv, main.firstChild);
        
        // Auto-hide after 5 seconds
        setTimeout(function() {
            flashDiv.style.transition = 'opacity 0.3s ease';
            flashDiv.style.opacity = '0';
            setTimeout(function() {
                flashDiv.remove();
            }, 300);
        }, 5000);
    }
}

// Table row highlighting
function highlightTableRow(row) {
    // Remove existing highlights
    const rows = row.closest('table').querySelectorAll('tr.highlighted');
    rows.forEach(function(r) {
        r.classList.remove('highlighted');
    });
    
    // Add highlight to current row
    row.classList.add('highlighted');
}

// Modal functionality (for future use)
function openModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
        modal.style.display = 'block';
        document.body.classList.add('modal-open');
    }
}

function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
        modal.style.display = 'none';
        document.body.classList.remove('modal-open');
    }
}

// Close modal when clicking outside
document.addEventListener('click', function(e) {
    if (e.target.classList.contains('modal')) {
        closeModal(e.target.id);
    }
});

// Close modal with Escape key
document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape') {
        const openModals = document.querySelectorAll('.modal[style*="block"]');
        openModals.forEach(function(modal) {
            closeModal(modal.id);
        });
    }
});

// Theme toggle functionality
function initializeThemeToggle() {
    const themeToggle = document.getElementById('theme-toggle');
    if (!themeToggle) return;
    
    // Load saved theme or detect system preference
    const savedTheme = localStorage.getItem('theme');
    const systemDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    
    if (savedTheme) {
        document.documentElement.setAttribute('data-theme', savedTheme);
    } else if (systemDark) {
        document.documentElement.setAttribute('data-theme', 'dark');
    }
    
    // Toggle theme on button click
    themeToggle.addEventListener('click', function() {
        const currentTheme = document.documentElement.getAttribute('data-theme');
        const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
        
        document.documentElement.setAttribute('data-theme', newTheme);
        localStorage.setItem('theme', newTheme);
        
        // Optional: Add a subtle animation
        themeToggle.style.transform = 'scale(0.9)';
        setTimeout(() => {
            themeToggle.style.transform = '';
        }, 150);
    });
    
    // Listen for system theme changes
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function(e) {
        // Only auto-update if user hasn't set a preference
        if (!localStorage.getItem('theme')) {
            document.documentElement.setAttribute('data-theme', e.matches ? 'dark' : 'light');
        }
    });
}

function getCurrentTheme() {
    return document.documentElement.getAttribute('data-theme') || 
           (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
}

function setTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('theme', theme);
}

// Export functions for use in templates
window.hatmaxAdmin = {
    showSuccess: showSuccess,
    showError: showError,
    showWarning: showWarning,
    openModal: openModal,
    closeModal: closeModal,
    highlightTableRow: highlightTableRow,
    getCurrentTheme: getCurrentTheme,
    setTheme: setTheme
};
