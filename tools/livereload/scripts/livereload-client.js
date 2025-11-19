// Live Reload Client Script
(function() {
    'use strict';
    
    let ws;
    let reconnectTimer = null;
    const reconnectDelay = 5000; // 5 seconds
    
    // State storage in memory
    let storedState = null;
    
    // Live reload status tracking
    let wsStatus = 'disconnected';
    let lastBuildStatus = null;
    let buildErrors = [];
    let buildHistory = [];
    
    // State management
    window.GoLiveReload = {
        exportState: function() {
            // Try to export WASM app state if available
            if (window.exportAppState && typeof window.exportAppState === 'function') {
                try {
                    const wasmState = window.exportAppState();
                    // console.log('🔄 GoLiveReload: Exported WASM state:', wasmState);
                    return wasmState;
                } catch (e) {
                    console.warn('🚨 Failed to export WASM state:', e);
                }
            }
            
            return null;
        },
        
        importState: function(state) {
            if (!state) return;
            
            // Try to import WASM app state if available
            if (window.importAppState && typeof window.importAppState === 'function') {
                try {
                    // console.log('🔄 GoLiveReload: Importing WASM state:', state);
                    window.importAppState(state);
                } catch (e) {
                    console.warn('🚨 Failed to import WASM state:', e);
                }
            }
        },
        
        storeState: function(state) {
            storedState = state;
            // console.log('💾 GoLiveReload: State stored in memory:', state);
        },
        
        getStoredState: function() {
            // console.log('📥 GoLiveReload: Retrieved stored state:', storedState);
            return storedState;
        },
        
        clearStoredState: function() {
            // console.log('🧹 GoLiveReload: Cleared stored state');
            storedState = null;
        },
        
        onReload: function(callback) {
            document.addEventListener('beforeunload', callback);
        }
    };
    
    function connect() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = protocol + '//' + window.location.host + '/ws';
        
        ws = new WebSocket(wsUrl);
        
        ws.onopen = function() {
            console.log('🔄 Live reload connected');
            wsStatus = 'connected';
            updateGWCIcon();
            
            // Clear any existing reconnect timer
            if (reconnectTimer) {
                clearTimeout(reconnectTimer);
                reconnectTimer = null;
            }
        };
        
        ws.onmessage = function(event) {
            try {
                const message = JSON.parse(event.data);
                handleMessage(message);
            } catch (e) {
                console.error('Failed to parse WebSocket message:', e);
            }
        };
        
        ws.onclose = function() {
            // console.log('🔄 Live reload disconnected');
            wsStatus = 'disconnected';
            updateGWCIcon();
            
            // Always try to reconnect every 5 seconds
            if (!reconnectTimer) {
                reconnectTimer = setTimeout(function() {
                    reconnectTimer = null;
                    connect();
                }, reconnectDelay);
            }
        };
        
        ws.onerror = function(error) {
            console.error('WebSocket error:', error);
            wsStatus = 'error';
            updateGWCIcon();
        };
    }
    
    function handleMessage(message) {
        // console.log('📨 Live reload message:', message.type);
        
        switch (message.type) {
            case 'build_start':
                const classification = message.payload?.classification;
                if (classification) {
                    // console.log('🔍 Update classification:', classification.type, '(' + classification.reloadType + ') -', classification.reason);
                }
                // Remove build toast - status shown in GWC icon instead
                break;
                
            case 'build_complete':
                if (message.payload && message.payload.success) {
                    const reloadType = message.payload.reloadType || 'full';
                    // console.log('✅ Build successful, reload type:', reloadType);
                    
                    lastBuildStatus = {
                        success: true,
                        duration: message.payload.duration,
                        reloadType: reloadType,
                        timestamp: new Date()
                    };
                    buildHistory.unshift(lastBuildStatus);
                    if (buildHistory.length > 10) buildHistory.pop(); // Keep last 10 builds
                    buildErrors = []; // Clear errors on successful build
                    updateGWCIcon();
                    showBuildStatusPopup('Build successful', 'success');
                    
                    if (reloadType === 'hot') {
                        performHotReload();
                    } else {
                        performFullReload();
                    }
                } else {
                    lastBuildStatus = {
                        success: false,
                        error: message.payload.error || 'Unknown build error',
                        timestamp: new Date()
                    };
                    buildHistory.unshift(lastBuildStatus);
                    if (buildHistory.length > 10) buildHistory.pop();
                    updateGWCIcon();
                    showBuildStatusPopup('Build failed', 'error');
                }
                break;
                
            case 'build_error':
                const errorMsg = message.payload || 'Unknown error';
                buildErrors.push({
                    error: errorMsg,
                    timestamp: new Date()
                });
                if (buildErrors.length > 5) buildErrors.shift(); // Keep last 5 errors
                
                lastBuildStatus = {
                    success: false,
                    error: errorMsg,
                    timestamp: new Date()
                };
                buildHistory.unshift(lastBuildStatus);
                if (buildHistory.length > 10) buildHistory.pop();
                updateGWCIcon();
                showBuildStatusPopup('Build error', 'error');
                break;
                
            case 'reload':
                location.reload();
                break;
                
            case 'hot_reload':
                performHotReload();
                break;
                
            case 'debounce_status':
                handleDebounceStatus(message.payload);
                break;
                
            case 'current_status':
                // Handle current status (don't trigger reloads, just update UI)
                if (message.payload) {
                    lastBuildStatus = {
                        success: message.payload.success,
                        error: message.payload.error,
                        duration: message.payload.duration,
                        reloadType: message.payload.reloadType,
                        timestamp: new Date()
                    };
                    
                    if (!message.payload.success) {
                        buildErrors.push({
                            error: message.payload.error,
                            timestamp: new Date()
                        });
                        if (buildErrors.length > 5) buildErrors.shift();
                    } else {
                        buildErrors = []; // Clear errors on successful status
                    }
                    
                    buildHistory.unshift(lastBuildStatus);
                    if (buildHistory.length > 10) buildHistory.pop();
                    updateGWCIcon();
                    // console.log('📊 Current build status received:', message.payload.success ? 'SUCCESS' : 'FAILED');
                }
                break;
        }
    }
    
    function performHotReload() {
        // console.log('🔥 Attempting hot reload...');
        
        // Export and store current state before reload
        const currentState = window.GoLiveReload.exportState();
        if (currentState) {
            window.GoLiveReload.storeState(currentState);
            // console.log('💾 State saved for hot reload');
        }
        
        // Try hot reload with WASM module replacement
        setTimeout(() => {
            try {
                if (window.hotReloadWasm && typeof window.hotReloadWasm === 'function') {
                    // console.log('🔥 Calling WASM hot reload function');
                    window.hotReloadWasm();
                } else {
                    // console.log('⚠️ WASM hot reload not available, trying manual reload');
                    // Try to reload just the WASM module
                    reloadWasmModule();
                }
            } catch (e) {
                // console.warn('🚨 Hot reload failed, falling back to full page reload:', e);
                performFullReload();
            }
        }, 100);
    }
    
    function performFullReload() {
        // console.log('🔄 Performing full page reload...');
        
        // Export and store current state before reload
        const currentState = window.GoLiveReload.exportState();
        if (currentState) {
            window.GoLiveReload.storeState(currentState);
            // console.log('💾 State saved for full reload');
        }
        
        // Full page reload
        setTimeout(() => {
            location.reload();
        }, 100);
    }
    
    function reloadWasmModule() {
        // console.log('🔄 Attempting WASM module reload...');
        
        // Clean up DOM before reloading WASM
        cleanupDOMContainers();
        
        // Try to find and reload the WASM script
        const wasmScript = document.querySelector('script[src*="wasm_exec.js"]');
        if (wasmScript) {
            // Create a new script element
            const newScript = document.createElement('script');
            newScript.src = wasmScript.src + '?t=' + Date.now();
            newScript.onload = function() {
                // console.log('✅ WASM script reloaded');
                // Try to reinitialize the Go WASM
                if (window.Go) {
                    const go = new Go();
                    WebAssembly.instantiateStreaming(fetch('/bin/main.wasm?t=' + Date.now()), go.importObject)
                        .then((result) => {
                            // console.log('✅ WASM module reloaded');
                            go.run(result.instance);
                        })
                        .catch((e) => {
                            // console.warn('🚨 WASM module reload failed:', e);
                            performFullReload();
                        });
                } else {
                    // console.warn('⚠️ Go WASM runtime not available');
                    performFullReload();
                }
            };
            newScript.onerror = function() {
                // console.warn('🚨 WASM script reload failed');
                performFullReload();
            };
            
            // Replace the old script
            wasmScript.parentNode.replaceChild(newScript, wasmScript);
        } else {
            // console.warn('⚠️ WASM script not found, falling back to full reload');
            performFullReload();
        }
    }
    
    function cleanupDOMContainers() {
        // console.log('🧹 Cleaning up DOM containers before WASM reload...');
        
        // Clean up the main app container
        const appElement = document.getElementById('app');
        if (appElement) {
            appElement.innerHTML = '';
            // console.log('✅ Cleared #app container');
        }
        
        // Clean up other common containers
        const containers = ['root', 'main', 'content'];
        containers.forEach(containerId => {
            const element = document.getElementById(containerId);
            if (element) {
                element.innerHTML = '';
                // console.log(`✅ Cleared #${containerId} container`);
            }
        });
        
        // Try to call WASM cleanup function if available
        if (window.cleanupDOM && typeof window.cleanupDOM === 'function') {
            try {
                window.cleanupDOM();
                // console.log('✅ Called WASM cleanup function');
            } catch (e) {
                // console.warn('⚠️ WASM cleanup function failed:', e);
            }
        }
    }
    
    function handleDebounceStatus(payload) {
        if (!payload) return;
        
        const { changeCount, timeSinceFirstChange, waitTime, maxWaitTime } = payload;
        const remainingTime = Math.max(0, waitTime);
        const totalElapsed = timeSinceFirstChange;
        
        let message = 'Waiting for changes... (' + changeCount + ' change' + (changeCount > 1 ? 's' : '') + ')';
        if (remainingTime > 0) {
            message += ' ' + (remainingTime / 1000).toFixed(1) + 's';
        }
        
        // Show progress if we're approaching max wait time
        if (totalElapsed > maxWaitTime * 0.5) {
            const progress = Math.min(100, (totalElapsed / maxWaitTime) * 100);
            message += ' [' + progress.toFixed(0) + '%]';
        }
        
        // Don't show debounce status as toast anymore - only in GWC icon
    }
    
    function showBuildStatusPopup(message, type) {
        // Remove existing popup
        const existing = document.getElementById('gwc-build-popup');
        if (existing) {
            existing.remove();
        }
        
        // Create small popup near the GWC icon
        const popup = document.createElement('div');
        popup.id = 'gwc-build-popup';
        popup.textContent = message;
        
        const colors = {
            success: '#10B981',
            error: '#EF4444',
            info: '#3B82F6'
        };
        
        Object.assign(popup.style, {
            position: 'fixed',
            bottom: '80px',
            left: '20px',
            padding: '8px 12px',
            backgroundColor: colors[type] || colors.info,
            color: 'white',
            borderRadius: '6px',
            fontFamily: 'monospace',
            fontSize: '11px',
            zIndex: '9998',
            boxShadow: '0 2px 8px rgba(0,0,0,0.3)',
            transform: 'translateY(10px)',
            opacity: '0',
            transition: 'all 0.3s ease',
            maxWidth: '200px'
        });
        
        document.body.appendChild(popup);
        
        // Animate in
        setTimeout(function() {
            popup.style.transform = 'translateY(0)';
            popup.style.opacity = '1';
        }, 10);
        
        // Auto-remove after 3 seconds
        setTimeout(function() {
            if (popup.parentNode) {
                popup.style.transform = 'translateY(10px)';
                popup.style.opacity = '0';
                setTimeout(function() {
                    if (popup.parentNode) {
                        popup.remove();
                    }
                }, 300);
            }
        }, 3000);
    }
    
    function showStatus(message, type, autoRemove = true) {
        // Remove existing status
        const existing = document.getElementById('livereload-status');
        if (existing) {
            existing.remove();
        }
        
        // Create status element
        const status = document.createElement('div');
        status.id = 'livereload-status';
        status.textContent = message;
        
        const colors = {
            success: '#10B981',
            error: '#EF4444',
            building: '#F59E0B',
            info: '#3B82F6',
            waiting: '#8B5CF6'
        };
        
        Object.assign(status.style, {
            position: 'fixed',
            top: '10px',
            right: '10px',
            padding: '8px 16px',
            backgroundColor: colors[type] || colors.info,
            color: 'white',
            borderRadius: '4px',
            fontFamily: 'monospace',
            fontSize: '12px',
            zIndex: '10000',
            boxShadow: '0 2px 4px rgba(0,0,0,0.2)',
            transition: 'all 0.3s ease'
        });
        
        document.body.appendChild(status);
        
        // Auto-remove success messages or when specified
        if (autoRemove && (type === 'success' || type === 'waiting')) {
            setTimeout(() => {
                if (status.parentNode) {
                    status.remove();
                }
            }, type === 'waiting' ? 1000 : 3000);
        }
    }
    
    // Restore state on page load
    document.addEventListener('DOMContentLoaded', function() {
        // Delay state restoration to ensure WASM is loaded
        setTimeout(() => {
            const savedState = window.GoLiveReload.getStoredState();
            if (savedState) {
                try {
                    // console.log('🔄 Restoring state after page load...');
                    window.GoLiveReload.importState(savedState);
                    window.GoLiveReload.clearStoredState();
                } catch (e) {
                    // console.warn('🚨 Failed to restore state:', e);
                }
            }
        }, 1000);
    });
    
    // Add CSS animation for pulse effect
    if (!document.getElementById('gwc-pulse-style')) {
        const style = document.createElement('style');
        style.id = 'gwc-pulse-style';
        style.textContent = '@keyframes pulse { 0% { opacity: 1; } 50% { opacity: 0.5; } 100% { opacity: 1; } }';
        document.head.appendChild(style);
    }

    // Create and manage GWC status icon
    function createGWCIcon() {
        // Remove existing icon if present
        const existing = document.getElementById('gwc-status-icon');
        if (existing) {
            existing.remove();
        }
        
        // Create icon container
        const icon = document.createElement('div');
        icon.id = 'gwc-status-icon';
        icon.innerHTML = 'GWC';
        
        // Create error badge for when there are build errors
        const errorBadge = document.createElement('div');
        errorBadge.id = 'gwc-error-badge';
        errorBadge.style.display = 'none';
        Object.assign(errorBadge.style, {
            position: 'absolute',
            top: '-5px',
            right: '-5px',
            backgroundColor: '#EF4444',
            color: 'white',
            borderRadius: '50%',
            width: '18px',
            height: '18px',
            fontSize: '10px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontWeight: 'bold',
            border: '2px solid white'
        });
        icon.appendChild(errorBadge);
        
        Object.assign(icon.style, {
            position: 'fixed',
            bottom: '20px',
            left: '20px',
            width: '50px',
            height: '50px',
            borderRadius: '50%',
            backgroundColor: '#1F2937',
            color: '#F9FAFB',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontFamily: 'monospace',
            fontSize: '10px',
            fontWeight: 'bold',
            cursor: 'pointer',
            zIndex: '9999',
            boxShadow: '0 4px 12px rgba(0,0,0,0.5)',
            border: '2px solid #374151',
            transform: 'none',
            transition: 'background-color 0.3s ease, border-color 0.3s ease, box-shadow 0.3s ease'
        });
        
        // Add click handler
        icon.onclick = toggleGWCPanel;
        
        // Add hover effects (no transform to prevent movement)
        icon.onmouseenter = function() {
            icon.style.boxShadow = '0 6px 16px rgba(0,0,0,0.6)';
            icon.style.borderColor = '#4B5563';
        };
        
        icon.onmouseleave = function() {
            icon.style.boxShadow = '0 4px 12px rgba(0,0,0,0.5)';
            icon.style.borderColor = '#374151';
        };
        
        document.body.appendChild(icon);
        updateGWCIcon();
    }
    
    function updateGWCIcon() {
        const icon = document.getElementById('gwc-status-icon');
        if (!icon) return;
        
        // Update icon color based on status (dark mode colors)
        let backgroundColor = '#1F2937'; // Default dark gray
        let borderColor = '#374151';
        
        if (wsStatus === 'connected') {
            if (lastBuildStatus && lastBuildStatus.success) {
                backgroundColor = '#059669'; // Dark green for success
                borderColor = '#047857';
            } else if (lastBuildStatus && !lastBuildStatus.success) {
                backgroundColor = '#DC2626'; // Dark red for build error
                borderColor = '#B91C1C';
            } else {
                backgroundColor = '#2563EB'; // Dark blue for connected but no build yet
                borderColor = '#1D4ED8';
            }
        } else if (wsStatus === 'error') {
            backgroundColor = '#D97706'; // Dark orange for connection error
            borderColor = '#B45309';
        }
        // else keep dark gray for disconnected
        
        icon.style.backgroundColor = backgroundColor;
        icon.style.borderColor = borderColor;
        
        // Add pulse animation for errors
        if ((lastBuildStatus && !lastBuildStatus.success) || wsStatus === 'error') {
            icon.style.animation = 'pulse 2s infinite';
        } else {
            icon.style.animation = 'none';
        }
        
        // Update error badge
        const errorBadge = document.getElementById('gwc-error-badge');
        if (errorBadge) {
            if (buildErrors.length > 0) {
                errorBadge.textContent = buildErrors.length;
                errorBadge.style.display = 'flex';
            } else {
                errorBadge.style.display = 'none';
            }
        }
    }
    
    function toggleGWCPanel() {
        const existingPanel = document.getElementById('gwc-status-panel');
        if (existingPanel) {
            existingPanel.remove();
            return;
        }
        
        createGWCPanel();
    }
    
    function createGWCPanel() {
        const panel = document.createElement('div');
        panel.id = 'gwc-status-panel';
        
        Object.assign(panel.style, {
            position: 'fixed',
            bottom: '80px',
            left: '20px',
            width: '350px',
            maxHeight: '400px',
            backgroundColor: '#1F2937',
            border: '1px solid #374151',
            borderRadius: '8px',
            boxShadow: '0 10px 25px rgba(0,0,0,0.5)',
            zIndex: '10000',
            fontFamily: 'monospace',
            fontSize: '12px',
            overflow: 'hidden',
            color: '#F9FAFB'
        });
        
        // Create header
        const header = document.createElement('div');
        header.innerHTML = 'Go Web Components - Live Reload Status';
        Object.assign(header.style, {
            padding: '12px 16px',
            backgroundColor: '#374151',
            borderBottom: '1px solid #4B5563',
            fontWeight: 'bold',
            color: '#F9FAFB'
        });
        panel.appendChild(header);
        
        // Create content
        const content = document.createElement('div');
        content.style.padding = '16px';
        content.style.maxHeight = '320px';
        content.style.overflowY = 'auto';
        
        // WebSocket Status
        const wsStatusDiv = document.createElement('div');
        wsStatusDiv.style.marginBottom = '16px';
        const wsStatusColor = wsStatus === 'connected' ? '#10B981' : wsStatus === 'error' ? '#F59E0B' : '#EF4444';
        wsStatusDiv.innerHTML = '<strong>WebSocket:</strong> <span style="color: ' + wsStatusColor + '">' + wsStatus.toUpperCase() + '</span>';
        content.appendChild(wsStatusDiv);
        
        // Last Build Status
        if (lastBuildStatus) {
            const buildStatusDiv = document.createElement('div');
            buildStatusDiv.style.marginBottom = '16px';
            const buildStatusColor = lastBuildStatus.success ? '#10B981' : '#EF4444';
            const statusText = lastBuildStatus.success ? 'SUCCESS' : 'FAILED';
            const timeAgo = formatTimeAgo(lastBuildStatus.timestamp);
            
            buildStatusDiv.innerHTML = '<strong>Last Build:</strong> <span style="color: ' + buildStatusColor + '">' + statusText + '</span> (' + timeAgo + ')';
            
            if (lastBuildStatus.success && lastBuildStatus.duration) {
                buildStatusDiv.innerHTML += '<br><small>Duration: ' + lastBuildStatus.duration + '</small>';
                if (lastBuildStatus.reloadType) {
                    buildStatusDiv.innerHTML += '<br><small>Reload: ' + lastBuildStatus.reloadType + '</small>';
                }
            }
            
            if (!lastBuildStatus.success && lastBuildStatus.error) {
                const errorDiv = document.createElement('div');
                errorDiv.style.marginTop = '8px';
                errorDiv.style.padding = '8px';
                errorDiv.style.backgroundColor = '#372B2B';
                errorDiv.style.border = '1px solid #5B4545';
                errorDiv.style.borderRadius = '4px';
                errorDiv.style.color = '#F87171';
                errorDiv.style.fontSize = '11px';
                errorDiv.style.position = 'relative';
                
                // Create copy button
                const copyBtn = document.createElement('button');
                copyBtn.innerHTML = '📋';
                copyBtn.title = 'Copy error to clipboard';
                Object.assign(copyBtn.style, {
                    position: 'absolute',
                    top: '4px',
                    right: '4px',
                    background: 'none',
                    border: 'none',
                    color: '#9CA3AF',
                    cursor: 'pointer',
                    fontSize: '12px',
                    padding: '2px 4px',
                    borderRadius: '2px',
                    transition: 'all 0.2s ease'
                });
                
                copyBtn.onmouseenter = function() {
                    copyBtn.style.backgroundColor = '#4B5563';
                    copyBtn.style.color = '#F9FAFB';
                };
                
                copyBtn.onmouseleave = function() {
                    copyBtn.style.backgroundColor = 'transparent';
                    copyBtn.style.color = '#9CA3AF';
                };
                
                copyBtn.onclick = function(e) {
                    e.stopPropagation();
                    copyToClipboard(lastBuildStatus.error);
                    copyBtn.innerHTML = '✅';
                    copyBtn.style.color = '#10B981';
                    setTimeout(function() {
                        copyBtn.innerHTML = '📋';
                        copyBtn.style.color = '#9CA3AF';
                    }, 2000);
                };
                
                errorDiv.innerHTML = '<strong>Error:</strong><br>' + escapeHtml(lastBuildStatus.error);
                errorDiv.appendChild(copyBtn);
                buildStatusDiv.appendChild(errorDiv);
            }
            
            content.appendChild(buildStatusDiv);
        }
        
        // Build History
        if (buildHistory.length > 0) {
            const historyDiv = document.createElement('div');
            historyDiv.innerHTML = '<strong>Recent Builds:</strong>';
            historyDiv.style.marginBottom = '8px';
            content.appendChild(historyDiv);
            
            buildHistory.slice(0, 5).forEach(function(build, index) {
                const buildDiv = document.createElement('div');
                buildDiv.style.padding = '4px 0';
                buildDiv.style.borderBottom = index < 4 ? '1px solid #4B5563' : 'none';
                buildDiv.style.color = '#D1D5DB';
                
                const statusIcon = build.success ? '✅' : '❌';
                const timeAgo = formatTimeAgo(build.timestamp);
                buildDiv.innerHTML = statusIcon + ' ' + timeAgo;
                
                if (build.success && build.duration) {
                    buildDiv.innerHTML += ' (' + build.duration + ')';
                }
                
                content.appendChild(buildDiv);
            });
        }
        
        panel.appendChild(content);
        
        // Close on click outside
        setTimeout(function() {
            document.addEventListener('click', function closePanel(e) {
                if (!panel.contains(e.target) && e.target.id !== 'gwc-status-icon') {
                    panel.remove();
                    document.removeEventListener('click', closePanel);
                }
            });
        }, 100);
        
        document.body.appendChild(panel);
    }
    
    function formatTimeAgo(timestamp) {
        const now = new Date();
        const diff = now - timestamp;
        const seconds = Math.floor(diff / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);
        
        if (hours > 0) return hours + 'h ago';
        if (minutes > 0) return minutes + 'm ago';
        return seconds + 's ago';
    }
    
    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
    
    function copyToClipboard(text) {
        if (navigator.clipboard && navigator.clipboard.writeText) {
            // Modern clipboard API
            navigator.clipboard.writeText(text).then(function() {
                // console.log('✅ Error copied to clipboard');
            }).catch(function(err) {
                // console.error('❌ Failed to copy to clipboard:', err);
                fallbackCopyToClipboard(text);
            });
        } else {
            // Fallback for older browsers
            fallbackCopyToClipboard(text);
        }
    }
    
    function fallbackCopyToClipboard(text) {
        const textArea = document.createElement('textarea');
        textArea.value = text;
        textArea.style.position = 'fixed';
        textArea.style.left = '-999999px';
        textArea.style.top = '-999999px';
        document.body.appendChild(textArea);
        textArea.focus();
        textArea.select();
        
        try {
            const successful = document.execCommand('copy');
            if (successful) {
                // console.log('✅ Error copied to clipboard (fallback)');
            } else {
                // console.error('❌ Failed to copy to clipboard (fallback)');
            }
        } catch (err) {
            // console.error('❌ Fallback copy failed:', err);
        }
        
        document.body.removeChild(textArea);
    }
    
    // Add CSS animation for pulse effect
    const style = document.createElement('style');
    style.textContent = '@keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.5; } }';
    document.head.appendChild(style);
    
    // Create icon on load
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', createGWCIcon);
    } else {
        createGWCIcon();
    }
    
    // Connect on load
    connect();
})(); 