package persistent

import (
	"esdeath_go/miku/base"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	"github.com/linxGnu/grocksdb"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
)

const (
	rocksdbPath      = "rocks_db"
	backupSubPath    = "backup"
	backZip          = "backZip.zip"
	compactThreshold = 10000
)

var (
	absStoragePath string
	absBackupPath  string
	opts           *grocksdb.Options
)

func init() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Error("get pwd err: ", err)
		os.Exit(0)
	}
	absStoragePath = filepath.Join(cwd, rocksdbPath)
	log.Info("storage path", "path", absStoragePath)
	absBackupPath = filepath.Join(absStoragePath, backupSubPath)
	log.Info("backup path", "path", absBackupPath)
	opts = initOps()

	PersistInstance = rocksdbPersistInit()
}

func initOps() *grocksdb.Options {
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	return opts
}

func rocksdbPersistInit() *rocksdbPersist {
	db, err := grocksdb.OpenDb(opts, rocksdbPath)
	if err != nil {
		log.Error("rocksdb init", "err", err)
		os.Exit(1)
	}

	db.EnableManualCompaction()
	log.Info("rocksdb init success", "path", rocksdbPath)
	return &rocksdbPersist{
		rocks:            db,
		ro:               grocksdb.NewDefaultReadOptions(),
		wo:               grocksdb.NewDefaultWriteOptions(),
		compactThreshold: compactThreshold,
		Logger:           log,
	}
}

type rocksdbPersist struct {
	rocks            *grocksdb.DB
	ro               *grocksdb.ReadOptions
	wo               *grocksdb.WriteOptions
	delCount         int32
	compactThreshold int32
	hclog.Logger
}

func (d *rocksdbPersist) GetOne(key []byte) (val []byte, err error) {
	value, err := d.rocks.Get(d.ro, key)
	if err != nil {
		d.Error("Error getting value:", err)
		return nil, err
	}
	defer value.Free()
	return append([]byte{}, value.Data()...), nil
}

// PrefixIterate 遍历指定前缀的数据
func (d *rocksdbPersist) PrefixIterate(prefix []byte, consumeCallback func(k, val []byte) bool) {
	iter := d.rocks.NewIterator(d.ro)
	defer iter.Close()
	for iter.Seek(prefix); iter.Valid(); iter.Next() {
		key := iter.Key()
		val := iter.Value()
		k := append([]byte{}, key.Data()...)
		v := append([]byte{}, val.Data()...)
		key.Free()
		val.Free()
		if consumeCallback(k, v) {
			break
		}
	}
}

// Insert 插入数据
func (d *rocksdbPersist) Insert(kvPairs []*KVPair) error {
	wb := grocksdb.NewWriteBatch()
	defer wb.Destroy()
	for _, kv := range kvPairs {
		wb.Put([]byte(kv.Key), []byte(kv.Value))
	}
	if err := d.rocks.Write(d.wo, wb); err != nil {
		d.Error("Error putting value:", err)
		return err
	}
	return nil
}

// Delete 删除数据
func (d *rocksdbPersist) Delete(kvPairs []*KVPair) error {
	wb := grocksdb.NewWriteBatch()
	defer wb.Destroy()
	for _, kv := range kvPairs {
		wb.Delete([]byte(kv.Key))
	}
	if err := d.rocks.Write(d.wo, wb); err != nil {
		d.Error("Error deleting value:", err)
		return err
	}

	atomic.AddInt32(&d.delCount, int32(len(kvPairs)))
	if atomic.LoadInt32(&d.delCount) >= d.compactThreshold {
		go d.tryCompact()
	}
	return nil
}

func (d *rocksdbPersist) tryCompact() {
	if atomic.CompareAndSwapInt32(&d.delCount, d.compactThreshold, 0) {
		log.Info("try compact")
		start := []byte("")
		end := []byte("")
		d.rocks.CompactRange(grocksdb.Range{Start: start, Limit: end})
		log.Info("compact success")
	}
}

func (d *rocksdbPersist) Shutdown() error {
	d.rocks.CancelAllBackgroundWork(true)
	d.rocks.Close()
	d.wo.Destroy()
	d.ro.Destroy()
	opts.Destroy()
	return nil
}

func (d *rocksdbPersist) Backup(sink raft.SnapshotSink) error {
	d.Info("start Backup")
	defer d.Info("end Backup")
	if err := d.rocksdbBackup(); err != nil {
		return err
	}
	if err := base.ZipBackup(absBackupPath, backZip); err != nil {
		return err
	}

	file, err := os.Open(backZip)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			d.Error("close file error", err)
		}
		if err := os.Remove(backZip); err != nil {
			d.Error("remove file error", err)
		}
	}()

	if _, err := io.Copy(sink, file); err != nil {
		return err
	}
	return nil
}

func (d *rocksdbPersist) rocksdbBackup() error {
	be, err := grocksdb.CreateBackupEngineWithPath(d.rocks, absBackupPath)
	if err != nil {
		log.Error("Error creating BackupEngine", "err", err)
		return nil
	}
	defer be.Close()

	if err = be.CreateNewBackup(); err != nil {
		log.Error("Error creating initial backup", "err", err)
		return err
	}
	if err := be.PurgeOldBackups(1); err != nil {
		log.Error("Error purging old backups", "err", err)
		return err
	}
	return nil
}

func (d *rocksdbPersist) ReLoad(rc io.ReadCloser) error {
	d.Info("start Restore")
	defer d.Info("end Restore")
	// 先关闭rocksdb
	d.rocks.CancelAllBackgroundWork(true)
	d.rocks.Close()

	tmpDir := filepath.Join(absStoragePath, "raft_reload_tmp")
	if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			d.Error("remove tmp dir error", err)
		}
	}()

	if err := base.UnzipBackup(rc, tmpDir); err != nil {
		return err
	}
	be, err := grocksdb.OpenBackupEngine(opts, tmpDir)
	if err != nil {
		log.Error("Error creating BackupEngine", "err", err)
		return err
	}
	defer be.Close()

	restoreOpts := grocksdb.NewRestoreOptions()
	defer restoreOpts.Destroy()
	err = be.RestoreDBFromLatestBackup(absStoragePath, absStoragePath, restoreOpts)
	if err != nil {
		log.Error("Error restoring from latest backup", "err", err)
		return err
	}
	d.rocks, _ = grocksdb.OpenDb(opts, rocksdbPath)
	d.rocks.EnableManualCompaction()
	return nil
}
