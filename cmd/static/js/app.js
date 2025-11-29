// Salam Monitoring Platform - Client-side JavaScript

// Global utilities
window.SalamMonitoring = {
    // Format duration in milliseconds to human readable format
    formatDuration: function(ms) {
        if (!ms) return 'N/A';
        
        const seconds = Math.floor(ms / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);
        const days = Math.floor(hours / 24);
        
        if (days > 0) {
            return `${days}d ${hours % 24}h`;
        } else if (hours > 0) {
            return `${hours}h ${minutes % 60}m`;
        } else if (minutes > 0) {
            return `${minutes}m ${seconds % 60}s`;
        } else {
            return `${seconds}s`;
        }
    },
    
    // Format bytes to human readable format
    formatBytes: function(bytes) {
        if (!bytes) return '0 B';
        
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    },
    
    // Format memory in MB to human readable format
    formatMemory: function(mb) {
        if (!mb) return '0 MB';
        
        if (mb < 1024) {
            return `${mb} MB`;
        } else if (mb < 1024 * 1024) {
            return `${(mb / 1024).toFixed(1)} GB`;
        } else {
            return `${(mb / (1024 * 1024)).toFixed(1)} TB`;
        }
    },
    
    // Show notification
    showNotification: function(message, type = 'info') {
        const notification = document.createElement('div');
        notification.className = `fixed top-4 right-4 p-4 rounded-md shadow-lg z-50 max-w-sm transition-all duration-300 ${
            type === 'success' ? 'bg-green-500 text-white' :
            type === 'error' ? 'bg-red-500 text-white' :
            type === 'warning' ? 'bg-yellow-500 text-white' :
            'bg-blue-500 text-white'
        }`;
        notification.textContent = message;
        
        document.body.appendChild(notification);
        
        // Auto remove after 5 seconds
        setTimeout(() => {
            notification.style.opacity = '0';
            notification.style.transform = 'translateX(100%)';
            setTimeout(() => {
                if (notification.parentNode) {
                    notification.parentNode.removeChild(notification);
                }
            }, 300);
        }, 5000);
    },
    
    // Copy text to clipboard
    copyToClipboard: function(text) {
        navigator.clipboard.writeText(text).then(() => {
            this.showNotification('Copied to clipboard', 'success');
        }).catch(() => {
            // Fallback for older browsers
            const textArea = document.createElement('textarea');
            textArea.value = text;
            document.body.appendChild(textArea);
            textArea.select();
            document.execCommand('copy');
            document.body.removeChild(textArea);
            this.showNotification('Copied to clipboard', 'success');
        });
    },
    
    // Confirm dialog with custom styling
    confirm: function(message, onConfirm, onCancel) {
        const overlay = document.createElement('div');
        overlay.className = 'fixed inset-0 bg-gray-600 bg-opacity-50 z-50 flex items-center justify-center';
        
        const dialog = document.createElement('div');
        dialog.className = 'bg-white rounded-lg shadow-xl max-w-md w-full mx-4 p-6';
        dialog.innerHTML = `
            <h3 class="text-lg font-medium text-gray-900 mb-4">Confirm Action</h3>
            <p class="text-gray-600 mb-6">${message}</p>
            <div class="flex justify-end space-x-4">
                <button id="cancel-btn" class="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50">
                    Cancel
                </button>
                <button id="confirm-btn" class="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700">
                    Confirm
                </button>
            </div>
        `;
        
        overlay.appendChild(dialog);
        document.body.appendChild(overlay);
        
        const confirmBtn = dialog.querySelector('#confirm-btn');
        const cancelBtn = dialog.querySelector('#cancel-btn');
        
        const cleanup = () => {
            document.body.removeChild(overlay);
        };
        
        confirmBtn.addEventListener('click', () => {
            cleanup();
            if (onConfirm) onConfirm();
        });
        
        cancelBtn.addEventListener('click', () => {
            cleanup();
            if (onCancel) onCancel();
        });
        
        overlay.addEventListener('click', (e) => {
            if (e.target === overlay) {
                cleanup();
                if (onCancel) onCancel();
            }
        });
    }
};

// Global event handlers
document.addEventListener('DOMContentLoaded', function() {
    // Auto-refresh functionality
    const autoRefreshInterval = setInterval(() => {
        const elements = document.querySelectorAll('[data-auto-refresh="true"]');
        elements.forEach(element => {
            if (element && htmx) {
                htmx.trigger(element, 'refresh');
            }
        });
    }, 30000); // Refresh every 30 seconds
    
    // Keyboard shortcuts
    document.addEventListener('keydown', function(e) {
        // Escape key to close modals
        if (e.key === 'Escape') {
            const modals = document.querySelectorAll('.fixed.inset-0:not(.hidden)');
            modals.forEach(modal => {
                if (modal.classList.contains('z-50')) {
                    modal.classList.add('hidden');
                }
            });
        }
        
        // Ctrl+R to refresh current view
        if (e.ctrlKey && e.key === 'r') {
            e.preventDefault();
            const refreshButtons = document.querySelectorAll('button[hx-get]');
            if (refreshButtons.length > 0) {
                refreshButtons[0].click();
            }
        }
    });
    
    // Click outside to close dropdowns
    document.addEventListener('click', function(e) {
        const dropdowns = document.querySelectorAll('.dropdown-menu');
        dropdowns.forEach(dropdown => {
            if (!dropdown.contains(e.target) && !dropdown.previousElementSibling.contains(e.target)) {
                dropdown.classList.add('hidden');
            }
        });
    });
});

// HTMX event handlers
if (typeof htmx !== 'undefined') {
    // Show loading indicator
    document.body.addEventListener('htmx:beforeRequest', function(evt) {
        const indicator = document.getElementById('refresh-indicator');
        if (indicator) {
            indicator.classList.remove('hidden');
        }
    });
    
    // Hide loading indicator
    document.body.addEventListener('htmx:afterRequest', function(evt) {
        const indicator = document.getElementById('refresh-indicator');
        if (indicator) {
            indicator.classList.add('hidden');
        }
    });
    
    // Handle errors
    document.body.addEventListener('htmx:responseError', function(evt) {
        SalamMonitoring.showNotification(
            'Failed to load data. Please check your connection.',
            'error'
        );
    });
    
    // Handle successful requests
    document.body.addEventListener('htmx:afterSwap', function(evt) {
        // Re-initialize any JavaScript components in the new content
        const newContent = evt.detail.target;
        
        // Initialize tooltips, charts, etc. here if needed
        initializeComponents(newContent);
    });
}

// Component initialization
function initializeComponents(container = document) {
    // Initialize copy buttons
    const copyButtons = container.querySelectorAll('[data-copy]');
    copyButtons.forEach(button => {
        button.addEventListener('click', function() {
            const text = this.getAttribute('data-copy');
            SalamMonitoring.copyToClipboard(text);
        });
    });
    
    // Initialize confirm buttons
    const confirmButtons = container.querySelectorAll('[data-confirm]');
    confirmButtons.forEach(button => {
        button.addEventListener('click', function(e) {
            e.preventDefault();
            const message = this.getAttribute('data-confirm');
            SalamMonitoring.confirm(message, () => {
                // Proceed with the original action
                if (this.getAttribute('hx-post')) {
                    htmx.trigger(this, 'click');
                } else if (this.href) {
                    window.location.href = this.href;
                }
            });
        });
    });
}

// Utility functions for specific components
window.Utils = {
    // Toggle visibility of an element
    toggle: function(elementId) {
        const element = document.getElementById(elementId);
        if (element) {
            element.classList.toggle('hidden');
        }
    },
    
    // Show element
    show: function(elementId) {
        const element = document.getElementById(elementId);
        if (element) {
            element.classList.remove('hidden');
        }
    },
    
    // Hide element
    hide: function(elementId) {
        const element = document.getElementById(elementId);
        if (element) {
            element.classList.add('hidden');
        }
    },
    
    // Update progress bar
    updateProgress: function(elementId, percentage) {
        const element = document.getElementById(elementId);
        if (element) {
            element.style.width = percentage + '%';
            element.setAttribute('aria-valuenow', percentage);
        }
    },
    
    // Highlight text in search results
    highlightText: function(text, searchTerm) {
        if (!searchTerm) return text;
        
        const regex = new RegExp(`(${searchTerm})`, 'gi');
        return text.replace(regex, '<mark class="bg-yellow-200">$1</mark>');
    }
};

// Initialize components when DOM is ready
document.addEventListener('DOMContentLoaded', function() {
    initializeComponents();
});