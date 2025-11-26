/**
 * Personal Website 2025 - Virtual Filesystem
 * In-memory filesystem for Go compiler integration
 */

class VirtualFileSystem {
    constructor() {
        this.files = new Map();
        this.directories = new Set();
        this.workingDirectory = '/';
    }

    /**
     * Write file to virtual filesystem
     * @param {string} path - File path
     * @param {string|Uint8Array} content - File content
     */
    writeFile(path, content) {
        const normalizedPath = this.normalizePath(path);
        this.files.set(normalizedPath, content);
        this.ensureDirectoryExists(this.getDirectory(normalizedPath));
        console.log(`📝 VFS: Written ${normalizedPath} (${content.length} bytes)`);
    }

    /**
     * Read file from virtual filesystem
     * @param {string} path - File path
     * @returns {string|Uint8Array} File content
     */
    readFile(path) {
        const normalizedPath = this.normalizePath(path);
        if (!this.files.has(normalizedPath)) {
            throw new Error(`File not found: ${normalizedPath}`);
        }
        return this.files.get(normalizedPath);
    }

    /**
     * Check if file exists
     * @param {string} path - File path
     * @returns {boolean}
     */
    exists(path) {
        const normalizedPath = this.normalizePath(path);
        return this.files.has(normalizedPath);
    }

    /**
     * List directory contents
     * @param {string} path - Directory path
     * @returns {Array<string>} File and directory names
     */
    listDir(path = '/') {
        let normalizedPath = this.normalizePath(path);
        if (!normalizedPath.endsWith('/')) normalizedPath += '/';
        
        const contents = new Set();
        
        // Find files in this directory
        for (const filePath of this.files.keys()) {
            if (filePath.startsWith(normalizedPath)) {
                const relativePath = filePath.substring(normalizedPath.length);
                const pathParts = relativePath.split('/').filter(p => p);
                if (pathParts.length > 0) {
                    contents.add(pathParts[0]);
                }
            }
        }

        // Find directories in this directory
        for (const dirPath of this.directories) {
            let dirPathStr = dirPath;
            if (!dirPathStr.startsWith('/')) dirPathStr = '/' + dirPathStr;
            
            // Check if directory is inside the requested path
            // We need to handle the case where dirPath equals normalizedPath (minus slash)
            if (dirPathStr.startsWith(normalizedPath) || (normalizedPath === '/' && dirPathStr.startsWith('/'))) {
                 // If normalizedPath is /, dirPathStr is /src. startsWith works.
                 // If normalizedPath is /src/, dirPathStr is /src/foo. startsWith works.
                 if (dirPathStr.startsWith(normalizedPath)) {
                    const relativePath = dirPathStr.substring(normalizedPath.length);
                    const pathParts = relativePath.split('/').filter(p => p);
                    if (pathParts.length > 0) {
                        contents.add(pathParts[0]);
                    }
                 }
            }
        }
        
        return [...contents].sort();
    }

    /**
     * Create directory
     * @param {string} path - Directory path
     */
    mkdir(path) {
        const normalizedPath = this.normalizePath(path);
        this.directories.add(normalizedPath);
        console.log(`📁 VFS: Created directory ${normalizedPath}`);
    }

    /**
     * Load Go source files from fetched data
     * @param {Object} sourceFiles - Files from GitHub fetcher
     */
    loadGoSources(sourceFiles) {
        console.log('📦 VFS: Loading Go source files...');
        
        for (const [filePath, content] of Object.entries(sourceFiles)) {
            this.writeFile(filePath, content);
        }
        
        // Create standard Go directories
        this.mkdir('/src');
        this.mkdir('/pkg');
        this.mkdir('/bin');
        
        console.log(`✅ VFS: Loaded ${Object.keys(sourceFiles).length} source files`);
    }

    /**
     * Get all Go files
     * @returns {Array<string>} List of .go file paths
     */
    getGoFiles() {
        return Array.from(this.files.keys()).filter(path => path.endsWith('.go'));
    }

    /**
     * Get main package files
     * @returns {Array<string>} List of main package .go files
     */
    getMainPackageFiles() {
        const mainFiles = [];
        
        for (const filePath of this.getGoFiles()) {
            const content = this.readFile(filePath);
            if (content.includes('package main')) {
                mainFiles.push(filePath);
            }
        }
        
        return mainFiles;
    }

    /**
     * Get module information from go.mod
     * @returns {Object} Module info
     */
    getModuleInfo() {
        try {
            const goModContent = this.readFile('/go.mod');
            const moduleMatch = goModContent.match(/module\s+([^\s\n]+)/);
            const goVersionMatch = goModContent.match(/go\s+([0-9.]+)/);
            
            return {
                name: moduleMatch ? moduleMatch[1] : 'unknown',
                goVersion: goVersionMatch ? goVersionMatch[1] : '1.21',
                dependencies: this.parseDependencies(goModContent)
            };
        } catch (e) {
            return {
                name: 'personal-website-2025',
                goVersion: '1.21',
                dependencies: []
            };
        }
    }

    /**
     * Parse dependencies from go.mod content
     * @private
     */
    parseDependencies(goModContent) {
        const deps = [];
        const requireMatch = goModContent.match(/require\s*\(([\s\S]*?)\)/);
        
        if (requireMatch) {
            const requireBlock = requireMatch[1];
            const depMatches = requireBlock.matchAll(/([^\s]+)\s+([^\s\n]+)/g);
            
            for (const match of depMatches) {
                deps.push({ name: match[1], version: match[2] });
            }
        }
        
        return deps;
    }

    /**
     * Generate file tree for debugging
     * @returns {string} ASCII file tree
     */
    getFileTree() {
        const paths = Array.from(this.files.keys()).sort();
        let tree = 'Virtual Filesystem:\n';
        
        for (const path of paths) {
            const depth = (path.match(/\//g) || []).length - 1;
            const indent = '  '.repeat(depth);
            const fileName = path.split('/').pop();
            tree += `${indent}├── ${fileName}\n`;
        }
        
        return tree;
    }

    // Utility methods
    normalizePath(path) {
        if (!path.startsWith('/')) {
            path = '/' + path;
        }
        return path.replace(/\/+/g, '/');
    }

    getDirectory(filePath) {
        const parts = filePath.split('/');
        parts.pop(); // Remove filename
        return parts.join('/') || '/';
    }

    ensureDirectoryExists(dirPath) {
        if (dirPath !== '/') {
            this.directories.add(dirPath);
        }
    }

    /**
     * Clear all files and directories
     */
    clear() {
        this.files.clear();
        this.directories.clear();
        console.log('🗑️ VFS: Cleared all files');
    }

    /**
     * Get filesystem stats
     * @returns {Object} Statistics
     */
    getStats() {
        return {
            totalFiles: this.files.size,
            totalDirectories: this.directories.size,
            goFiles: this.getGoFiles().length,
            totalSize: Array.from(this.files.values()).reduce((sum, content) => sum + content.length, 0)
        };
    }
}

// Export for use in other modules
window.VirtualFileSystem = VirtualFileSystem; 