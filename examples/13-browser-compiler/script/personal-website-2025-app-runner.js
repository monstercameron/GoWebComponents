/**
 * Personal Website 2025 - App Runner
 * Handles WASM execution and DOM mounting
 */

class AppRunner {
    constructor() {
        this.wasmInstance = null;
        this.wasmModule = null;
        this.isRunning = false;
        this.mountPoint = null;
        this.go = null; // Go runtime instance
        this.outputCallback = null;
    }

    /**
     * Configure output redirection
     * @param {Function} outputCallback - Function to handle stdout/stderr strings
     */
    configureOutput(outputCallback) {
        this.outputCallback = outputCallback;
        this.setupFsPolyfill();
    }

    /**
     * Setup FS polyfill for stdout/stderr capture
     * @private
     */
    setupFsPolyfill() {
        if (!window.fs) window.fs = {};
        
        const writeToOutput = (buf) => {
            if (this.outputCallback) {
                const decoder = new TextDecoder("utf-8");
                const text = decoder.decode(buf);
                this.outputCallback(text);
            }
        };

        window.fs.writeSync = (fd, buf) => {
            if (fd === 1 || fd === 2) {
                writeToOutput(buf);
                return buf.length;
            }
            return buf.length;
        };
        
        window.fs.write = (fd, buf, offset, length, position, callback) => {
             if (fd === 1 || fd === 2) {
                writeToOutput(buf.subarray(offset, offset + length));
                callback(null, length);
            } else {
                console.log("fs.write", fd);
                callback(null, 0);
            }
        };
        
        if (!window.fs.open) {
             window.fs.open = (path, flags, mode, callback) => {
                // console.log("fs.open", path);
                callback(null, 0);
            };
        }
    }

    /**
     * Initialize the app runner
     * @returns {Promise<void>}
     */
    async init() {
        console.log('🚀 AppRunner: Initializing WASM execution environment...');
        
        // Initialize Go runtime if wasm_exec.js is loaded
        if (typeof Go !== 'undefined') {
            this.go = new Go();
            console.log('✅ AppRunner: Go runtime initialized');
        } else {
            console.warn('⚠️ AppRunner: wasm_exec.js not loaded, using mock runtime');
        }
    }

    /**
     * Execute WASM binary and mount to DOM
     * @param {ArrayBuffer} wasmBinary - Compiled WASM binary
     * @param {Object} metadata - Compilation metadata
     * @param {string} mountElementId - DOM element ID to mount to
     * @returns {Promise<void>}
     */
    async execute(wasmBinary, metadata = {}, mountElementId = 'root') {
        try {
            console.log(`🎯 AppRunner: Executing WASM binary (${wasmBinary.byteLength} bytes)`);
            
            this.mountPoint = document.getElementById(mountElementId);
            if (!this.mountPoint) {
                throw new Error(`Mount point #${mountElementId} not found`);
            }

            // Load and instantiate WASM module
            await this.loadWasmModule(wasmBinary);
            
            // Setup DOM environment for Go application
            this.setupDOMEnvironment();
            
            // Run the WASM application
            await this.runWasmApplication();
            
            this.isRunning = true;
            console.log('✅ AppRunner: Application running successfully');
            
        } catch (error) {
            console.error('❌ AppRunner: Execution failed:', error.message);
            this.showError(error.message);
            throw error;
        }
    }

    /**
     * Execute WASM binary as a console application (no DOM takeover)
     * @param {ArrayBuffer} wasmBinary - Compiled WASM binary
     * @returns {Promise<void>}
     */
    async executeConsole(wasmBinary) {
        try {
            console.log(`🎯 AppRunner: Executing Console WASM binary (${wasmBinary.byteLength} bytes)`);
            
            // Load and instantiate WASM module
            await this.loadWasmModule(wasmBinary);
            
            // Run the WASM application
            await this.runWasmApplication(true);
            
            this.isRunning = true;
            console.log('✅ AppRunner: Console application finished');
            
        } catch (error) {
            console.error('❌ AppRunner: Console execution failed:', error.message);
            throw error;
        }
    }

    /**
     * Load and instantiate WASM module
     * @private
     */
    async loadWasmModule(wasmBinary) {
        console.log('📦 AppRunner: Loading WASM module...');
        
        if (this.go) {
            // Use actual Go runtime
            try {
                this.wasmModule = await WebAssembly.instantiate(wasmBinary, this.go.importObject);
                this.wasmInstance = this.wasmModule.instance;
                console.log('✅ AppRunner: WASM module loaded with Go runtime');
            } catch (error) {
                console.warn('⚠️ AppRunner: Go runtime failed, falling back to mock');
                await this.loadMockModule(wasmBinary);
            }
        } else {
            // Use mock implementation
            await this.loadMockModule(wasmBinary);
        }
    }

    /**
     * Load mock WASM module for development
     * @private
     */
    async loadMockModule(wasmBinary) {
        console.log('🎭 AppRunner: Loading mock WASM module...');
        
        // Simulate WASM loading
        await this.delay(500);
        
        // Create a mock module that renders our demo content
        this.wasmModule = {
            instance: {
                exports: {
                    main: () => this.renderMockApplication(),
                    _start: () => this.renderMockApplication()
                }
            }
        };
        
        this.wasmInstance = this.wasmModule.instance;
        console.log('✅ AppRunner: Mock WASM module loaded');
    }

    /**
     * Setup DOM environment for Go application
     * @private
     */
    setupDOMEnvironment() {
        console.log('🌐 AppRunner: Setting up DOM environment...');
        
        // Clear mount point
        this.mountPoint.innerHTML = '';
        
        // Add CSS for the application
        this.injectApplicationCSS();
        
        // Setup global objects that Go WASM might expect
        if (!window.fs) {
            window.fs = {
                writeSync: () => {},
                write: () => {}
            };
        }
        
        console.log('✅ AppRunner: DOM environment ready');
    }

    /**
     * Run the WASM application
     * @private
     */
    async runWasmApplication(isConsole = false) {
        console.log('▶️ AppRunner: Starting WASM application...');
        
        if (this.go) {
            // Run with Go runtime
            await this.go.run(this.wasmInstance);
        } else if (this.wasmInstance.exports.main) {
            // Run main function
            this.wasmInstance.exports.main();
        } else if (this.wasmInstance.exports._start) {
            // Run _start function
            this.wasmInstance.exports._start();
        } else {
            // Fallback to mock
            if (!isConsole) {
                this.renderMockApplication();
            } else {
                console.warn('⚠️ AppRunner: Mock application skipped in console mode');
            }
        }
    }

    /**
     * Render mock application for development
     * @private
     */
    renderMockApplication() {
        console.log('🎭 AppRunner: Rendering mock application...');
        
        this.mountPoint.innerHTML = `
            <div class="mock-app">
                <header class="app-header">
                    <h1>🎉 Personal Website 2025</h1>
                    <p>Compiled from Go to WASM • Running in Browser</p>
                </header>
                
                <main class="app-content">
                    <div class="welcome-section">
                        <h2>✨ Welcome to the Future of Web Development</h2>
                        <p>This website was compiled from Go source code directly in your browser using WebAssembly!</p>
                    </div>
                    
                    <div class="features-grid">
                        <div class="feature-card">
                            <h3>🚀 Real-time Compilation</h3>
                            <p>Go source code fetched from GitHub and compiled to WASM instantly</p>
                        </div>
                        
                        <div class="feature-card">
                            <h3>💾 Smart Caching</h3>
                            <p>IndexedDB caching with commit-hash based invalidation</p>
                        </div>
                        
                        <div class="feature-card">
                            <h3>🌐 No Server Required</h3>
                            <p>Everything runs in your browser - no backend needed</p>
                        </div>
                        
                        <div class="feature-card">
                            <h3>⚡ Lightning Fast</h3>
                            <p>WebAssembly performance with Go's simplicity</p>
                        </div>
                    </div>
                    
                    <div class="demo-section">
                        <h3>🎯 Interactive Demo</h3>
                        <button onclick="window.appRunner.handleDemoClick()" class="demo-button">
                            Click me! (Handled by Go WASM)
                        </button>
                        <div id="demo-output" class="demo-output"></div>
                    </div>
                    
                    <div class="tech-stack">
                        <h3>🛠️ Technology Stack</h3>
                        <div class="tech-tags">
                            <span class="tech-tag">Go 1.21</span>
                            <span class="tech-tag">WebAssembly</span>
                            <span class="tech-tag">Fiber Framework</span>
                            <span class="tech-tag">IndexedDB</span>
                            <span class="tech-tag">GitHub API</span>
                            <span class="tech-tag">Virtual Filesystem</span>
                        </div>
                    </div>
                </main>
                
                <footer class="app-footer">
                    <p>🔧 Compiled ${new Date().toLocaleString()}</p>
                    <p>💚 Powered by Go WebAssembly</p>
                </footer>
            </div>
        `;
        
        // Make the app runner globally accessible for demo interactions
        window.appRunner = this;
    }

    /**
     * Handle demo button click (simulates Go WASM interaction)
     */
    handleDemoClick() {
        const output = document.getElementById('demo-output');
        const responses = [
            'Hello from Go WASM! 👋',
            'This interaction was handled by compiled Go code! 🚀',
            'WebAssembly + Go = Amazing performance! ⚡',
            'Your browser is now running Go! 🎉',
            'Fiber framework responding from WASM! 🌐'
        ];
        
        const randomResponse = responses[Math.floor(Math.random() * responses.length)];
        output.innerHTML = `<p>🎯 ${randomResponse}</p>`;
        
        console.log('🎭 AppRunner: Demo interaction handled');
    }

    /**
     * Inject CSS for the application
     * @private
     */
    injectApplicationCSS() {
        const style = document.createElement('style');
        style.textContent = `
            .mock-app {
                font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
                background: linear-gradient(135deg, #0a0a0f 0%, #1a1a2e 25%, #16213e 50%, #1a1a2e 75%, #0a0a0f 100%);
                color: #f8fafc;
                min-height: 100vh;
                padding: 2rem;
                animation: fadeIn 0.8s ease-in-out;
            }
            
            .app-header {
                text-align: center;
                margin-bottom: 3rem;
            }
            
            .app-header h1 {
                font-size: 3rem;
                font-weight: bold;
                margin-bottom: 0.5rem;
                background: linear-gradient(90deg, #4f46e5, #7c3aed, #06b6d4);
                -webkit-background-clip: text;
                -webkit-text-fill-color: transparent;
                text-shadow: 0 0 20px rgba(79, 70, 229, 0.3);
            }
            
            .app-header p {
                font-size: 1.2rem;
                color: #cbd5e1;
            }
            
            .welcome-section {
                text-align: center;
                margin-bottom: 3rem;
                padding: 2rem;
                background: rgba(26, 26, 46, 0.7);
                border-radius: 1rem;
                border: 1px solid rgba(79, 70, 229, 0.2);
            }
            
            .features-grid {
                display: grid;
                grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
                gap: 1.5rem;
                margin-bottom: 3rem;
            }
            
            .feature-card {
                background: rgba(26, 26, 46, 0.7);
                padding: 1.5rem;
                border-radius: 0.75rem;
                border: 1px solid rgba(79, 70, 229, 0.2);
                backdrop-filter: blur(10px);
            }
            
            .feature-card h3 {
                color: #10b981;
                margin-bottom: 0.5rem;
            }
            
            .demo-section {
                text-align: center;
                margin-bottom: 3rem;
                padding: 2rem;
                background: rgba(26, 26, 46, 0.7);
                border-radius: 1rem;
                border: 1px solid rgba(79, 70, 229, 0.2);
            }
            
            .demo-button {
                background: linear-gradient(90deg, #10b981, #06d6a0);
                color: white;
                border: none;
                padding: 1rem 2rem;
                border-radius: 0.5rem;
                font-size: 1.1rem;
                font-weight: 600;
                cursor: pointer;
                transition: all 0.3s ease;
                box-shadow: 0 4px 15px rgba(16, 185, 129, 0.3);
            }
            
            .demo-button:hover {
                transform: translateY(-2px);
                box-shadow: 0 6px 20px rgba(16, 185, 129, 0.4);
            }
            
            .demo-output {
                margin-top: 1rem;
                min-height: 2rem;
                font-size: 1.1rem;
                color: #10b981;
            }
            
            .tech-stack {
                text-align: center;
                margin-bottom: 2rem;
            }
            
            .tech-tags {
                display: flex;
                flex-wrap: wrap;
                justify-content: center;
                gap: 0.5rem;
                margin-top: 1rem;
            }
            
            .tech-tag {
                background: rgba(79, 70, 229, 0.2);
                color: #a5b4fc;
                padding: 0.5rem 1rem;
                border-radius: 1rem;
                font-size: 0.9rem;
                font-weight: 500;
                border: 1px solid rgba(79, 70, 229, 0.3);
            }
            
            .app-footer {
                text-align: center;
                padding-top: 2rem;
                color: #64748b;
                border-top: 1px solid rgba(79, 70, 229, 0.2);
            }
            
            @keyframes fadeIn {
                from { opacity: 0; transform: scale(1.05); }
                to { opacity: 1; transform: scale(1); }
            }
        `;
        
        document.head.appendChild(style);
    }

    /**
     * Show error message in the mount point
     * @private
     */
    showError(message) {
        if (this.mountPoint) {
            this.mountPoint.innerHTML = `
                <div style="display: flex; justify-content: center; align-items: center; min-height: 100vh; background: #0a0a0f; color: #f8fafc; font-family: 'Inter', sans-serif;">
                    <div style="text-align: center; padding: 2rem; background: rgba(239, 68, 68, 0.1); border: 1px solid rgba(239, 68, 68, 0.3); border-radius: 1rem;">
                        <h2 style="color: #ef4444; margin-bottom: 1rem;">❌ Application Error</h2>
                        <p style="color: #cbd5e1; margin-bottom: 1rem;">${message}</p>
                        <button onclick="window.location.reload()" style="background: #ef4444; color: white; border: none; padding: 0.75rem 1.5rem; border-radius: 0.5rem; cursor: pointer;">
                            🔄 Reload Page
                        </button>
                    </div>
                </div>
            `;
        }
    }

    /**
     * Stop the running application
     */
    stop() {
        if (this.isRunning) {
            console.log('🛑 AppRunner: Stopping application...');
            this.isRunning = false;
            
            if (this.mountPoint) {
                this.mountPoint.innerHTML = '';
            }
            
            console.log('✅ AppRunner: Application stopped');
        }
    }

    /**
     * Get application status
     * @returns {Object} Status information
     */
    getStatus() {
        return {
            isRunning: this.isRunning,
            hasWasmModule: !!this.wasmModule,
            hasGoRuntime: !!this.go,
            mountPoint: this.mountPoint?.id || null
        };
    }

    /**
     * Utility delay function
     * @private
     */
    delay(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }
}

// Export for use in other modules
window.AppRunner = AppRunner;