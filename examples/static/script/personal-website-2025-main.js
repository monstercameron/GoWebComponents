/**
 * Personal Website 2025 - Main Integration Script
 * Orchestrates the complete system initialization and execution
 */

class PersonalWebsite2025 {
    constructor() {
        this.modules = {
            gitHubFetcher: null,
            vfs: null,
            cacheManager: null,
            compilationManager: null,
            appRunner: null
        };
        
        this.state = {
            initialized: false,
            compiling: false,
            compiled: false,
            running: false,
            compiledWasmBinary: null
        };
        
        this.ui = {
            updateProgress: null,
            updateProgressText: null,
            setStageStatus: null,
            addConsoleOutput: null,
            showReadyState: null,
            showError: null
        };
    }

    /**
     * Initialize the complete system
     * @param {Object} uiCallbacks - UI callback functions
     */
    async init(uiCallbacks) {
        try {
            this.ui = uiCallbacks;
            
            this.ui.addConsoleOutput('🚀 PWS-2025: System initialization started');
            this.ui.updateProgressText('SYSTEM_INIT');
            
            // Initialize all modules
            await this.initializeModules();
            
            // Set up module interconnections
            this.wireModules();
            
            // Start the compilation pipeline
            await this.startCompilation();
            
            this.state.initialized = true;
            this.ui.addConsoleOutput('✅ PWS-2025: System ready for launch');
            
        } catch (error) {
            this.ui.addConsoleOutput(`❌ PWS-2025: Initialization failed - ${error.message}`);
            this.ui.showError(error.message);
            throw error;
        }
    }

    /**
     * Initialize all system modules
     * @private
     */
    async initializeModules() {
        this.ui.addConsoleOutput('🔧 PWS-2025: Creating module instances...');
        
        // Create all modules
        this.modules.gitHubFetcher = new GitHubFetcher();
        this.modules.vfs = new VirtualFileSystem();
        this.modules.cacheManager = new CacheManager();
        this.modules.appRunner = new AppRunner();
        
        // Initialize async modules
        await this.modules.cacheManager.init();
        await this.modules.appRunner.init();
        
        // Create compilation manager
        this.modules.compilationManager = new CompilationManager();
        this.modules.compilationManager.init(this.modules.vfs, this.modules.cacheManager);
        
        this.ui.addConsoleOutput('✅ PWS-2025: All modules created successfully');
    }

    /**
     * Wire modules together with callbacks
     * @private
     */
    wireModules() {
        this.ui.addConsoleOutput('🔗 PWS-2025: Wiring module callbacks...');
        
        // Set up compilation manager callbacks
        this.modules.compilationManager.setCallbacks({
            onProgress: (percentage, status) => {
                this.ui.updateProgress(percentage);
                this.ui.updateProgressText(status);
                this.ui.addConsoleOutput(`📊 PROGRESS: ${percentage}% - ${status}`);
            },
            
            onStageUpdate: (stage, status) => {
                this.ui.setStageStatus(stage, status);
                const stageNames = [
                    '', 'Compiler Load', 'Cache Check', 'Source Fetch', 
                    'VFS Setup', 'Go Compile', 'Binary Cache', 'Prep Exec'
                ];
                this.ui.addConsoleOutput(`🎯 ${stageNames[stage]}: ${status.toUpperCase()}`);
            },
            
            onError: (message) => {
                this.ui.addConsoleOutput(`❌ COMPILATION_ERROR: ${message}`);
                this.state.compiling = false;
                this.ui.showError(message);
            },
            
            onComplete: (wasmBinary, metadata) => {
                this.state.compiledWasmBinary = wasmBinary;
                this.state.compiled = true;
                this.state.compiling = false;
                
                this.ui.addConsoleOutput(`✅ COMPILATION_SUCCESS: Generated ${wasmBinary.byteLength} byte WASM binary`);
                this.ui.addConsoleOutput(`📊 METADATA: ${metadata.sourceFiles} files, ${metadata.optimizations.join(', ')} opts`);
                
                // Show ready state for launch
                this.ui.showReadyState();
            }
        });
        
        this.ui.addConsoleOutput('✅ PWS-2025: Module callbacks wired');
    }

    /**
     * Start the compilation pipeline
     * @private
     */
    async startCompilation() {
        if (this.state.compiling) {
            this.ui.addConsoleOutput('⚠️ PWS-2025: Compilation already in progress');
            return;
        }

        try {
            this.state.compiling = true;
            this.ui.addConsoleOutput('🔄 PWS-2025: Starting compilation pipeline...');
            
            // Fetch source files
            const sourceFiles = await this.modules.gitHubFetcher.fetchGoWebComponentsSources();
            this.ui.addConsoleOutput(`📥 PWS-2025: Fetched ${Object.keys(sourceFiles).length} source files`);
            
            // Start compilation
            const wasmBinary = await this.modules.compilationManager.compile(sourceFiles);
            
            // Compilation completion is handled by callbacks
            
        } catch (error) {
            this.state.compiling = false;
            this.ui.addConsoleOutput(`❌ PWS-2025: Compilation failed - ${error.message}`);
            throw error;
        }
    }

    /**
     * Launch the compiled application
     */
    async launchApplication() {
        if (!this.state.compiled || !this.state.compiledWasmBinary) {
            this.ui.addConsoleOutput('❌ PWS-2025: No compiled binary available for launch');
            return;
        }

        try {
            this.state.running = true;
            this.ui.addConsoleOutput('🚀 PWS-2025: Launching compiled application...');
            
            // Execute the WASM binary
            await this.modules.appRunner.execute(this.state.compiledWasmBinary);
            
            this.ui.addConsoleOutput('✅ PWS-2025: Application launched successfully');
            
        } catch (error) {
            this.state.running = false;
            this.ui.addConsoleOutput(`❌ PWS-2025: Launch failed - ${error.message}`);
            throw error;
        }
    }

    /**
     * Get system status
     * @returns {Object} Current system state
     */
    getStatus() {
        return {
            ...this.state,
            modules: Object.keys(this.modules).reduce((acc, key) => {
                acc[key] = !!this.modules[key];
                return acc;
            }, {}),
            appRunnerStatus: this.modules.appRunner?.getStatus() || null
        };
    }

    /**
     * Reset the system to initial state
     */
    reset() {
        this.ui.addConsoleOutput('🔄 PWS-2025: Resetting system...');
        
        // Stop running application
        if (this.modules.appRunner && this.state.running) {
            this.modules.appRunner.stop();
        }
        
        // Reset state
        this.state = {
            initialized: false,
            compiling: false,
            compiled: false,
            running: false,
            compiledWasmBinary: null
        };
        
        // Clear VFS
        if (this.modules.vfs) {
            this.modules.vfs.clear();
        }
        
        this.ui.addConsoleOutput('✅ PWS-2025: System reset complete');
    }

    /**
     * Get compilation and cache statistics
     * @returns {Promise<Object>} System statistics
     */
    async getStats() {
        const stats = {
            vfs: this.modules.vfs?.getStats() || null,
            cache: await this.modules.cacheManager?.getStats() || null,
            compilation: {
                status: this.modules.compilationManager?.getStatus() || 'unknown',
                hasBinary: !!this.state.compiledWasmBinary,
                binarySize: this.state.compiledWasmBinary?.byteLength || 0
            }
        };
        
        return stats;
    }
}

// Export globally for use in HTML
window.PersonalWebsite2025 = PersonalWebsite2025; 