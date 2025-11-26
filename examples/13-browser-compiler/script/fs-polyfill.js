/**
 * Filesystem Polyfill for Go WASM
 * Bridges Node.js fs API to VirtualFileSystem
 */

class FSPolyfill {
    constructor(vfs) {
        this.vfs = vfs;
        this.fds = new Map();
        this.nextFd = 100;
    }

    patch() {
        const self = this;
        const enosys = () => {
            const err = new Error("not implemented");
            err.code = "ENOSYS";
            return err;
        };

        globalThis.fs = {
            constants: { O_WRONLY: 1, O_RDWR: 2, O_CREAT: 64, O_TRUNC: 512, O_APPEND: 1024, O_EXCL: 128, O_DIRECTORY: 65536 },
            
            writeSync(fd, buf) {
                if (fd === 1 || fd === 2) {
                    const text = new TextDecoder().decode(buf);
                    if (window.addConsoleOutput) {
                        window.addConsoleOutput(text.trimEnd());
                    } else {
                        console.log(text);
                    }
                    return buf.length;
                }
                
                const file = self.fds.get(fd);
                if (!file) throw new Error("EBADF");
                
                // Append to file content
                const newContent = new Uint8Array(file.content.length + buf.length);
                newContent.set(file.content);
                newContent.set(buf, file.content.length);
                file.content = newContent;
                
                // Sync to VFS immediately
                self.vfs.writeFile(file.path, file.content);
                
                return buf.length;
            },

            write(fd, buf, offset, length, position, callback) {
                try {
                    if (fd === 1 || fd === 2) {
                        const text = new TextDecoder().decode(buf.subarray(offset, offset + length));
                        if (window.addConsoleOutput) {
                            window.addConsoleOutput(text.trimEnd());
                        } else {
                            console.log(text);
                        }
                        callback(null, length);
                        return;
                    }
                    
                    const file = self.fds.get(fd);
                    if (!file) { callback(new Error("EBADF")); return; }
                    
                    const data = buf.subarray(offset, offset + length);
                    let pos = position !== null ? position : file.position;
                    
                    // Expand file if needed
                    if (pos + length > file.content.length) {
                        const newContent = new Uint8Array(pos + length);
                        newContent.set(file.content);
                        file.content = newContent;
                    }
                    
                    file.content.set(data, pos);
                    if (position === null) {
                        file.position = pos + length;
                    }
                    
                    // Sync to VFS
                    self.vfs.writeFile(file.path, file.content);
                    
                    callback(null, length);
                } catch (e) {
                    callback(e);
                }
            },

            open(path, flags, mode, callback) {
                try {
                    // Handle relative paths
                    if (!path.startsWith('/')) {
                        path = self.vfs.workingDirectory + (self.vfs.workingDirectory.endsWith('/') ? '' : '/') + path;
                    }
                    path = self.vfs.normalizePath(path);

                    let content = new Uint8Array(0);
                    if (self.vfs.exists(path)) {
                        const vfsContent = self.vfs.readFile(path);
                        if (typeof vfsContent === 'string') {
                            content = new TextEncoder().encode(vfsContent);
                        } else {
                            content = vfsContent;
                        }
                    } else {
                        if (!(flags & 64)) { // O_CREAT
                            const err = new Error("ENOENT");
                            err.code = "ENOENT";
                            callback(err);
                            return;
                        }
                    }
                    
                    if (flags & 512) { // O_TRUNC
                        content = new Uint8Array(0);
                    }
                    
                    const fd = self.nextFd++;
                    self.fds.set(fd, {
                        path,
                        flags,
                        content,
                        position: 0
                    });
                    
                    callback(null, fd);
                } catch (e) {
                    callback(e);
                }
            },

            read(fd, buffer, offset, length, position, callback) {
                try {
                    const file = self.fds.get(fd);
                    if (!file) { callback(new Error("EBADF")); return; }
                    
                    let pos = position !== null ? position : file.position;
                    
                    if (pos >= file.content.length) {
                        callback(null, 0);
                        return;
                    }
                    
                    const end = Math.min(pos + length, file.content.length);
                    const bytesRead = end - pos;
                    
                    buffer.set(file.content.subarray(pos, end), offset);
                    
                    if (position === null) {
                        file.position += bytesRead;
                    }
                    
                    callback(null, bytesRead);
                } catch (e) {
                    callback(e);
                }
            },

            close(fd, callback) {
                const file = self.fds.get(fd);
                if (file) {
                    self.fds.delete(fd);
                }
                callback(null);
            },

            fstat(fd, callback) {
                const file = self.fds.get(fd);
                if (!file) { callback(new Error("EBADF")); return; }
                callback(null, {
                    isDirectory: () => false,
                    isFile: () => true,
                    size: file.content.length,
                    mode: 0o666,
                    dev: 0,
                    ino: 0,
                    nlink: 1,
                    uid: 0,
                    gid: 0,
                    rdev: 0,
                    blksize: 4096,
                    blocks: 0,
                    atimeMs: Date.now(),
                    mtimeMs: Date.now(),
                    ctimeMs: Date.now()
                });
            },

            stat(path, callback) {
                try {
                    if (!path.startsWith('/')) {
                        path = self.vfs.workingDirectory + (self.vfs.workingDirectory.endsWith('/') ? '' : '/') + path;
                    }
                    path = self.vfs.normalizePath(path);

                    if (self.vfs.exists(path)) {
                         const content = self.vfs.readFile(path);
                         const size = content.length;
                         callback(null, {
                             isDirectory: () => false,
                             isFile: () => true,
                             size: size,
                             mode: 0o666,
                             dev: 0,
                             ino: 0,
                             nlink: 1,
                             uid: 0,
                             gid: 0,
                             rdev: 0,
                             blksize: 4096,
                             blocks: 0,
                             atimeMs: Date.now(),
                             mtimeMs: Date.now(),
                             ctimeMs: Date.now()
                         });
                    } else if (self.vfs.directories.has(path) || path === '/') {
                        callback(null, {
                            isDirectory: () => true,
                            isFile: () => false,
                            size: 0,
                            mode: 0o777 | 0o40000, // Add directory bit
                            dev: 0,
                            ino: 0,
                            nlink: 1,
                            uid: 0,
                            gid: 0,
                            rdev: 0,
                            blksize: 4096,
                            blocks: 0,
                            atimeMs: Date.now(),
                            mtimeMs: Date.now(),
                            ctimeMs: Date.now()
                        });
                    } else {
                        const err = new Error("ENOENT");
                        err.code = "ENOENT";
                        callback(err);
                    }
                } catch (e) {
                    callback(e);
                }
            },

            lstat(path, callback) {
                this.stat(path, callback);
            },

            mkdir(path, perm, callback) {
                try {
                    self.vfs.mkdir(path);
                    callback(null);
                } catch (e) {
                    callback(e);
                }
            },

            readdir(path, callback) {
                try {
                    const files = self.vfs.listDir(path);
                    callback(null, files);
                } catch (e) {
                    callback(e);
                }
            },
            
            unlink(path, callback) {
                // Not implemented in VFS yet
                callback(null);
            },
            
            rename(from, to, callback) {
                // Not implemented in VFS yet
                callback(null);
            },
            
            rmdir(path, callback) {
                callback(null);
            }
        };

        // Patch process
        if (!globalThis.process) globalThis.process = {};
        globalThis.process.cwd = () => self.vfs.workingDirectory;
        globalThis.process.chdir = (path) => {
            self.vfs.workingDirectory = path;
        };
    }
}

window.FSPolyfill = FSPolyfill;
