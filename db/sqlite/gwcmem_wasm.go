//go:build js && wasm

package sqlite

// gwcmem is a snapshot-able in-memory SQLite VFS. It is a direct adaptation of
// github.com/ncruces/go-sqlite3/vfs/memdb (the public VFS template), extended
// with gwcmemSnapshot/gwcmemRestore so the whole database image can be moved in
// and out of a durable backend (IndexedDB). Databases are shared by name and
// selected via the DSN "file:/<name>?vfs=gwcmem".
//
// Like memdb, it never persists a separate journal/WAL file (OPEN_MEMORY), so a
// single contiguous byte image fully represents the database.

import (
	"io"
	"strings"
	"sync"
	"time"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/util/vfsutil"
	"github.com/ncruces/go-sqlite3/vfs"
)

const gwcSectorSize = 65536

func init() {
	vfs.Register("gwcmem", gwcVFS{})
}

var (
	gwcMemoryMtx sync.Mutex
	gwcMemoryDBs = map[string]*gwcDB{}
)

// gwcmemRestore seeds the shared database "name" with an existing image. It must
// be called before opening the connection. The image is copied.
func gwcmemRestore(parseName string, parseData []byte) {
	parseDB := &gwcDB{name: parseName, size: int64(len(parseData))}

	// Convert WAL/2 headers to a rollback journal, as memdb does.
	if len(parseData) >= 20 && (parseData[18] == 2 && parseData[19] == 2 ||
		parseData[18] == 3 && parseData[19] == 3) {
		parseData[18] = 1
		parseData[19] = 1
	}

	parseSectors := gwcDivRoundUp(parseDB.size, gwcSectorSize)
	parseDB.data = make([]*[gwcSectorSize]byte, parseSectors)
	for parseI := range parseDB.data {
		parseDB.data[parseI] = new([gwcSectorSize]byte)
		parseChunk := parseData[parseI*gwcSectorSize:]
		copy((*parseDB.data[parseI])[:], parseChunk)
	}

	gwcMemoryMtx.Lock()
	gwcMemoryDBs[parseName] = parseDB
	gwcMemoryMtx.Unlock()
}

// gwcmemSnapshot returns a copy of the current image of database "name".
func gwcmemSnapshot(parseName string) ([]byte, bool) {
	gwcMemoryMtx.Lock()
	parseDB := gwcMemoryDBs[parseName]
	gwcMemoryMtx.Unlock()
	if parseDB == nil {
		return nil, false
	}

	parseDB.dataMtx.RLock()
	defer parseDB.dataMtx.RUnlock()
	parseImage := make([]byte, parseDB.size)
	for parseI, parseSector := range parseDB.data {
		parseStart := int64(parseI) * gwcSectorSize
		if parseStart >= parseDB.size {
			break
		}
		copy(parseImage[parseStart:], (*parseSector)[:])
	}
	return parseImage, true
}

type gwcVFS struct{}

func (gwcVFS) Open(parseName string, parseFlags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
	const parseDatabases = vfs.OPEN_MAIN_DB | vfs.OPEN_TEMP_DB | vfs.OPEN_TRANSIENT_DB

	if parseFlags&vfs.OPEN_TEMP_JOURNAL != 0 {
		return &vfsutil.SliceFile{}, parseFlags | vfs.OPEN_MEMORY, nil
	}
	if parseFlags&parseDatabases == 0 {
		return nil, parseFlags, sqlite3.CANTOPEN
	}

	parseShared := strings.HasPrefix(parseName, "/")
	var parseDB *gwcDB
	if parseShared {
		parseName = parseName[1:]
		gwcMemoryMtx.Lock()
		defer gwcMemoryMtx.Unlock()
		parseDB = gwcMemoryDBs[parseName]
	}
	if parseDB == nil {
		if parseFlags&vfs.OPEN_CREATE == 0 {
			return nil, parseFlags, sqlite3.CANTOPEN
		}
		parseDB = &gwcDB{name: parseName}
	}
	if parseShared {
		parseDB.refs++
		gwcMemoryDBs[parseName] = parseDB
	}
	return &gwcFile{
		gwcDB:    parseDB,
		readOnly: parseFlags&vfs.OPEN_READONLY != 0,
	}, parseFlags | vfs.OPEN_MEMORY, nil
}

func (gwcVFS) Delete(parseName string, parseDirSync bool) error {
	return sqlite3.IOERR_DELETE_NOENT
}

func (gwcVFS) Access(parseName string, parseFlag vfs.AccessFlag) (bool, error) {
	return false, nil
}

func (gwcVFS) FullPathname(parseName string) (string, error) {
	return parseName, nil
}

type gwcDB struct {
	name string

	waiter *sync.Cond
	data   []*[gwcSectorSize]byte

	size     int64
	refs     int32
	shared   int32
	pending  bool
	reserved bool

	lockMtx sync.Mutex
	dataMtx sync.RWMutex
}

func (parseM *gwcDB) release() {
	gwcMemoryMtx.Lock()
	// Keep the image around for snapshotting even after the last connection
	// closes; only drop it if it was never the registered shared DB.
	if parseM.refs--; parseM.refs < 0 {
		parseM.refs = 0
	}
	gwcMemoryMtx.Unlock()
}

type gwcFile struct {
	*gwcDB
	lock     vfs.LockLevel
	readOnly bool
}

var (
	_ vfs.FileLockState = &gwcFile{}
	_ vfs.FileSizeHint  = &gwcFile{}
)

func (parseM *gwcFile) Close() error {
	parseM.release()
	return parseM.Unlock(vfs.LOCK_NONE)
}

func (parseM *gwcFile) ReadAt(parseB []byte, parseOff int64) (parseN int, parseErr error) {
	parseM.dataMtx.RLock()
	defer parseM.dataMtx.RUnlock()

	if parseOff >= parseM.size {
		return 0, io.EOF
	}
	parseBase := parseOff / gwcSectorSize
	parseRest := parseOff % gwcSectorSize
	parseHave := int64(gwcSectorSize)
	if parseM.size < parseOff+int64(len(parseB)) {
		parseHave = gwcModRoundUp(parseM.size, gwcSectorSize)
	}
	parseN = copy(parseB, (*parseM.data[parseBase])[parseRest:parseHave])
	if parseN < len(parseB) {
		return 0, io.ErrNoProgress
	}
	return parseN, nil
}

func (parseM *gwcFile) WriteAt(parseB []byte, parseOff int64) (parseN int, parseErr error) {
	parseM.dataMtx.Lock()
	defer parseM.dataMtx.Unlock()

	parseBase := parseOff / gwcSectorSize
	parseRest := parseOff % gwcSectorSize
	for parseBase >= int64(len(parseM.data)) {
		parseM.data = append(parseM.data, new([gwcSectorSize]byte))
	}
	parseN = copy((*parseM.data[parseBase])[parseRest:], parseB)
	if parseSize := parseOff + int64(parseN); parseSize > parseM.size {
		parseM.size = parseSize
	}
	if parseN < len(parseB) {
		return parseN, io.ErrShortWrite
	}
	return parseN, nil
}

func (parseM *gwcFile) Size() (int64, error) {
	parseM.dataMtx.RLock()
	defer parseM.dataMtx.RUnlock()
	return parseM.size, nil
}

func (parseM *gwcFile) Truncate(parseSize int64) error {
	parseM.dataMtx.Lock()
	defer parseM.dataMtx.Unlock()
	return parseM.truncate(parseSize)
}

func (parseM *gwcFile) SizeHint(parseSize int64) error {
	parseM.dataMtx.Lock()
	defer parseM.dataMtx.Unlock()
	if parseSize > parseM.size {
		return parseM.truncate(parseSize)
	}
	return nil
}

func (parseM *gwcFile) truncate(parseSize int64) error {
	if parseSize < parseM.size {
		parseBase := parseSize / gwcSectorSize
		parseRest := parseSize % gwcSectorSize
		if parseRest != 0 {
			clear((*parseM.data[parseBase])[parseRest:])
		}
	}
	parseSectors := gwcDivRoundUp(parseSize, gwcSectorSize)
	for parseSectors > int64(len(parseM.data)) {
		parseM.data = append(parseM.data, new([gwcSectorSize]byte))
	}
	clear(parseM.data[parseSectors:])
	parseM.data = parseM.data[:parseSectors]
	parseM.size = parseSize
	return nil
}

func (parseM *gwcFile) Lock(parseLock vfs.LockLevel) error {
	if parseM.lock >= parseLock {
		return nil
	}
	if parseM.readOnly && parseLock >= vfs.LOCK_RESERVED {
		return sqlite3.IOERR_LOCK
	}

	parseM.lockMtx.Lock()
	defer parseM.lockMtx.Unlock()

	switch parseLock {
	case vfs.LOCK_SHARED:
		if parseM.pending {
			return sqlite3.BUSY
		}
		parseM.shared++
	case vfs.LOCK_RESERVED:
		if parseM.reserved {
			return sqlite3.BUSY
		}
		parseM.reserved = true
	case vfs.LOCK_EXCLUSIVE:
		if parseM.lock == vfs.LOCK_RESERVED {
			parseM.lock = vfs.LOCK_PENDING
			parseM.pending = true
		}
		if parseM.shared > 1 {
			parseBefore := time.Now()
			if parseM.waiter == nil {
				parseM.waiter = sync.NewCond(&parseM.lockMtx)
			}
			defer time.AfterFunc(time.Millisecond, parseM.waiter.Broadcast).Stop()
			for parseM.shared > 1 {
				if time.Since(parseBefore) > time.Millisecond {
					return sqlite3.BUSY
				}
				parseM.waiter.Wait()
			}
		}
	}
	parseM.lock = parseLock
	return nil
}

func (parseM *gwcFile) Unlock(parseLock vfs.LockLevel) error {
	if parseM.lock <= parseLock {
		return nil
	}
	parseM.lockMtx.Lock()
	defer parseM.lockMtx.Unlock()

	if parseM.lock >= vfs.LOCK_RESERVED {
		parseM.reserved = false
	}
	if parseM.lock >= vfs.LOCK_PENDING {
		parseM.pending = false
	}
	if parseLock < vfs.LOCK_SHARED {
		if parseM.shared--; parseM.pending && parseM.shared <= 1 && parseM.waiter != nil {
			parseM.waiter.Broadcast()
		}
	}
	parseM.lock = parseLock
	return nil
}

func (parseM *gwcFile) CheckReservedLock() (bool, error) {
	if parseM.lock >= vfs.LOCK_RESERVED {
		return true, nil
	}
	parseM.lockMtx.Lock()
	defer parseM.lockMtx.Unlock()
	return parseM.reserved, nil
}

func (parseM *gwcFile) LockState() vfs.LockLevel { return parseM.lock }

func (*gwcFile) Sync(parseFlag vfs.SyncFlag) error { return nil }

func (*gwcFile) SectorSize() int { return gwcSectorSize }

func (*gwcFile) DeviceCharacteristics() vfs.DeviceCharacteristic {
	return vfs.IOCAP_ATOMIC |
		vfs.IOCAP_SEQUENTIAL |
		vfs.IOCAP_SAFE_APPEND |
		vfs.IOCAP_POWERSAFE_OVERWRITE
}

func gwcDivRoundUp(parseA, parseB int64) int64 { return (parseA + parseB - 1) / parseB }

func gwcModRoundUp(parseA, parseB int64) int64 { return parseB - (parseB-parseA%parseB)%parseB }
