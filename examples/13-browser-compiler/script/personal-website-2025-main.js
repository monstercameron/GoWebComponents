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
            
            // Initialize all modules
            await this.initializeModules();
            
            // Configure output redirection
            this.modules.appRunner.configureOutput((text) => {
                this.ui.addConsoleOutput(text, true);
            });
            
            // Set up module interconnections
            this.wireModules();
            
            this.setupTerminal();
            
            this.state.initialized = true;
            this.ui.addConsoleOutput('✅ PWS-2025: System ready. Type "go run main.go" to compile and run.');
            
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
                // this.ui.addConsoleOutput(`📊 PROGRESS: ${percentage}% - ${status}`);
            },
            
            onStageUpdate: (stage, status) => {
                const stageNames = [
                    '', 'Compiler Load', 'Cache Check', 'Source Fetch', 
                    'VFS Setup', 'Go Compile', 'Binary Cache', 'Prep Exec'
                ];
                if (status === 'active') {
                    this.ui.addConsoleOutput(`🎯 ${stageNames[stage]}...`);
                }
            },
            
            onError: (message) => {
                this.ui.addConsoleOutput(`❌ COMPILATION_ERROR: ${message}`);
                this.state.compiling = false;
                this.ui.showError(message);
            },
            
            onComplete: async (wasmBinary, metadata) => {
                this.state.compiledWasmBinary = wasmBinary;
                this.state.compiled = true;
                this.state.compiling = false;
                
                this.ui.addConsoleOutput(`✅ COMPILATION_SUCCESS: Generated ${wasmBinary.byteLength} byte WASM binary`);
                
                // Auto-run console app
                this.ui.addConsoleOutput('⏳ Running application...');
                await this.runConsoleApp();
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
            
            // Check editor content
            const editorContent = document.getElementById('code-editor').value;
            let sourceFiles;
            
            if (editorContent && editorContent.trim() !== '') {
                 this.ui.addConsoleOutput('📝 PWS-2025: Using source code from editor');
                 sourceFiles = { 'main.go': editorContent };
            } else {
                 // Fetch source files
                 sourceFiles = await this.modules.gitHubFetcher.fetchGoWebComponentsSources();
                 this.ui.addConsoleOutput(`📥 PWS-2025: Fetched ${Object.keys(sourceFiles).length} source files`);
            }
            
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
     * Start the compilation pipeline with editor support
     * @private
     */
    async startCompilationWithEditor() {
        if (this.state.compiling) {
            this.ui.addConsoleOutput('⚠️ PWS-2025: Compilation already in progress');
            return;
        }

        try {
            this.state.compiling = true;
            this.ui.addConsoleOutput('🔄 PWS-2025: Starting compilation pipeline...');
            
            // Check editor content
            const editorContent = document.getElementById('code-editor').value;
            let sourceFiles;
            
            if (editorContent && editorContent.trim() !== '') {
                 this.ui.addConsoleOutput('📝 PWS-2025: Using source code from editor');
                 sourceFiles = { 'main.go': editorContent };
            } else {
                 // Fetch source files
                 sourceFiles = await this.modules.gitHubFetcher.fetchGoWebComponentsSources();
                 this.ui.addConsoleOutput(`📥 PWS-2025: Fetched ${Object.keys(sourceFiles).length} source files`);
            }
            
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
     * Run the compiled application in console mode
     */
    async runConsoleApp() {
        if (!this.state.compiled || !this.state.compiledWasmBinary) {
            this.ui.addConsoleOutput('❌ PWS-2025: No compiled binary available');
            return;
        }
        
        try {
            await this.modules.appRunner.executeConsole(this.state.compiledWasmBinary);
        } catch (error) {
            this.ui.addConsoleOutput(`❌ PWS-2025: Execution failed - ${error.message}`);
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

    /**
     * Setup the interactive terminal
     * @private
     */
    setupTerminal() {
        const input = document.getElementById('terminal-input');
        const container = document.getElementById('terminal-container');
        
        if (!input || !container) return;

        this.currentDir = '/';
        this.commandHistory = [];
        this.historyIndex = -1;

        input.addEventListener('keydown', async (e) => {
            if (e.key === 'Enter') {
                const command = input.value;
                input.value = '';
                
                // Add to history
                if (command.trim()) {
                    this.commandHistory.push(command);
                    this.historyIndex = this.commandHistory.length;
                }

                // Echo command
                this.ui.addConsoleOutput(`$ ${command}`);
                
                await this.handleCommand(command.trim());
                
                // Scroll to bottom
                container.scrollTop = container.scrollHeight;
            } else if (e.key === 'ArrowUp') {
                e.preventDefault();
                if (this.historyIndex > 0) {
                    this.historyIndex--;
                    input.value = this.commandHistory[this.historyIndex];
                }
            } else if (e.key === 'ArrowDown') {
                e.preventDefault();
                if (this.historyIndex < this.commandHistory.length - 1) {
                    this.historyIndex++;
                    input.value = this.commandHistory[this.historyIndex];
                } else {
                    this.historyIndex = this.commandHistory.length;
                    input.value = '';
                }
            }
        });
        
        // Focus input on click
        container.addEventListener('click', () => input.focus());
        input.focus();
    }

    /**
     * Handle terminal commands
     * @param {string} cmd 
     */
    async handleCommand(cmd) {
        if (!cmd) return;
        
        // Simple argument parsing handling quotes
        const args = cmd.match(/(?:[^\s"]+|"[^"]*")+/g)?.map(arg => arg.replace(/"/g, '')) || [];
        const command = args[0];
        const vfs = this.modules.vfs;

        // Helper to resolve path
        const resolvePath = (path) => {
            if (path.startsWith('/')) return path;
            if (this.currentDir === '/') return '/' + path;
            return this.currentDir + '/' + path;
        };
        
        switch(command) {
            case 'help':
                this.ui.addConsoleOutput('Available commands: help, clear, ls, cd, pwd, cat, echo, mkdir, touch, rm, cp, mv, grep, head, tail, history, whoami, date, exit, go');
                break;
            case 'clear':
                document.getElementById('console-output').textContent = '';
                break;
            case 'date':
                this.ui.addConsoleOutput(new Date().toString());
                break;
            case 'whoami':
                this.ui.addConsoleOutput('visitor');
                break;
            case 'pwd':
                this.ui.addConsoleOutput(this.currentDir);
                break;
            case 'echo':
                this.ui.addConsoleOutput(args.slice(1).join(' '));
                break;
            case 'ls':
                const lsPath = args[1] ? resolvePath(args[1]) : this.currentDir;
                try {
                    const files = vfs.listDir(lsPath);
                    this.ui.addConsoleOutput(files.join('  '));
                } catch (e) {
                    this.ui.addConsoleOutput(`ls: ${e.message}`);
                }
                break;
            case 'cd':
                const target = args[1] || '/';
                if (target === '..') {
                    const parts = this.currentDir.split('/').filter(p => p);
                    parts.pop();
                    this.currentDir = '/' + parts.join('/');
                    if (this.currentDir === '') this.currentDir = '/';
                } else {
                    const newPath = resolvePath(target);
                    // Simple directory check: assume it exists if we are just navigating or if it's root
                    // In a real VFS we'd check if it's a directory.
                    // For now, let's just set it.
                    this.currentDir = newPath.replace(/\/+$/, '') || '/';
                }
                break;
            case 'mkdir':
                if (!args[1]) { this.ui.addConsoleOutput('mkdir: missing operand'); break; }
                vfs.mkdir(resolvePath(args[1]));
                break;
            case 'touch':
                if (!args[1]) { this.ui.addConsoleOutput('touch: missing operand'); break; }
                vfs.writeFile(resolvePath(args[1]), '');
                break;
            case 'rm':
                if (!args[1]) { this.ui.addConsoleOutput('rm: missing operand'); break; }
                const rmPath = resolvePath(args[1]);
                if (vfs.files.has(rmPath)) {
                    vfs.files.delete(rmPath);
                } else {
                    this.ui.addConsoleOutput(`rm: ${args[1]}: No such file`);
                }
                break;
            case 'cat':
                if (!args[1]) { this.ui.addConsoleOutput('cat: missing operand'); break; }
                try {
                    const content = vfs.readFile(resolvePath(args[1]));
                    if (content instanceof Uint8Array) {
                        this.ui.addConsoleOutput(new TextDecoder().decode(content));
                    } else {
                        this.ui.addConsoleOutput(content);
                    }
                } catch (e) {
                    this.ui.addConsoleOutput(`cat: ${e.message}`);
                }
                break;
            case 'cp':
                if (args.length < 3) { this.ui.addConsoleOutput('cp: missing file operand'); break; }
                try {
                    const src = vfs.readFile(resolvePath(args[1]));
                    vfs.writeFile(resolvePath(args[2]), src);
                } catch (e) {
                    this.ui.addConsoleOutput(`cp: ${e.message}`);
                }
                break;
            case 'mv':
                if (args.length < 3) { this.ui.addConsoleOutput('mv: missing file operand'); break; }
                try {
                    const srcPath = resolvePath(args[1]);
                    const destPath = resolvePath(args[2]);
                    const srcContent = vfs.readFile(srcPath);
                    vfs.writeFile(destPath, srcContent);
                    vfs.files.delete(srcPath);
                } catch (e) {
                    this.ui.addConsoleOutput(`mv: ${e.message}`);
                }
                break;
            case 'grep':
                if (args.length < 3) { this.ui.addConsoleOutput('grep: usage: grep PATTERN FILE'); break; }
                try {
                    const pattern = args[1];
                    const content = vfs.readFile(resolvePath(args[2]));
                    const text = content instanceof Uint8Array ? new TextDecoder().decode(content) : content;
                    const lines = text.split('\n');
                    lines.forEach(line => {
                        if (line.includes(pattern)) this.ui.addConsoleOutput(line);
                    });
                } catch (e) {
                    this.ui.addConsoleOutput(`grep: ${e.message}`);
                }
                break;
            case 'head':
                if (!args[1]) { this.ui.addConsoleOutput('head: missing operand'); break; }
                try {
                    const content = vfs.readFile(resolvePath(args[1]));
                    const text = content instanceof Uint8Array ? new TextDecoder().decode(content) : content;
                    const lines = text.split('\n').slice(0, 10);
                    this.ui.addConsoleOutput(lines.join('\n'));
                } catch (e) {
                    this.ui.addConsoleOutput(`head: ${e.message}`);
                }
                break;
            case 'tail':
                if (!args[1]) { this.ui.addConsoleOutput('tail: missing operand'); break; }
                try {
                    const content = vfs.readFile(resolvePath(args[1]));
                    const text = content instanceof Uint8Array ? new TextDecoder().decode(content) : content;
                    const lines = text.split('\n');
                    this.ui.addConsoleOutput(lines.slice(-10).join('\n'));
                } catch (e) {
                    this.ui.addConsoleOutput(`tail: ${e.message}`);
                }
                break;
            case 'history':
                this.commandHistory.forEach((cmd, i) => {
                    this.ui.addConsoleOutput(`${i + 1}  ${cmd}`);
                });
                break;
            case 'exit':
                this.ui.addConsoleOutput('logout');
                break;
            case 'go':
                if (args[1] === 'run') {
                    await this.startCompilationWithEditor();
                } else if (args[1] === 'version') {
                    this.ui.addConsoleOutput('go version go1.21.0 js/wasm');
                } else {
                    this.ui.addConsoleOutput('Usage: go run [file]');
                }
                break;
            case './main.wasm':
                if (this.state.compiled) {
                    await this.launchApplication();
                } else {
                    this.ui.addConsoleOutput('bash: ./main.wasm: No such file or directory (compile first)');
                }
                break;
            default:
                this.ui.addConsoleOutput(`bash: command not found: ${command}`);
        }
    }
}

// Export globally for use in HTML
window.PersonalWebsite2025 = PersonalWebsite2025;