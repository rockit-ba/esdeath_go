package test

import (
	"github.com/linxGnu/grocksdb"
	"testing"
)

func TestIncrementalBackup(t *testing.T) {
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	db, err := grocksdb.OpenDb(opts, "rocksdb")
	if err != nil {
		t.Fatalf("Error opening RocksDB: %v", err)
	}
	defer db.Close()

	wo := grocksdb.NewDefaultWriteOptions()
	defer wo.Destroy()

	if err := db.Put(wo, []byte("key1"), []byte("value1")); err != nil {
		t.Fatalf("Error putting value: %v", err)
	}

	be, err := grocksdb.CreateBackupEngineWithPath(db, "rocksdb_backup")
	if err != nil {
		t.Fatalf("Error creating BackupEngine: %v", err)
	}
	defer be.Close()

	if err := be.CreateNewBackup(); err != nil {
		t.Fatalf("Error creating initial backup: %v", err)
	}
	err = be.VerifyBackup(1)
	if err != nil {
		t.Error("Error verifying backup:", err)
	}
	t.Log("first backup created")

	if err := db.Put(wo, []byte("key2"), []byte("value2")); err != nil {
		t.Fatalf("Error putting value: %v", err)
	}

	if err := be.CreateNewBackup(); err != nil {
		t.Fatalf("Error creating incremental backup: %v", err)
	}
	err = be.VerifyBackup(2)
	if err != nil {
		t.Error("Error verifying backup:", err)
	}
	t.Log("incremental backup created")

	if err := db.Put(wo, []byte("key3"), []byte("value3")); err != nil {
		t.Fatalf("Error putting value: %v", err)
	}

	if err := be.CreateNewBackup(); err != nil {
		t.Fatalf("Error creating incremental backup: %v", err)
	}
	err = be.VerifyBackup(3)
	if err != nil {
		t.Error("Error verifying backup:", err)
	}
	t.Log("incremental backup created")

	//if err := be.PurgeOldBackups(1); err != nil {
	//	t.Error("Error purging old backups:", err)
	//}
	//t.Log("old backups purged")
}

func TestDel(t *testing.T) {
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	db, err := grocksdb.OpenDb(opts, "rocksdb")
	if err != nil {
		t.Fatalf("Error opening RocksDB: %v", err)
	}
	defer db.Close()

	wo := grocksdb.NewDefaultWriteOptions()
	defer wo.Destroy()
	// Insert new data
	err = db.Delete(wo, []byte("key2"))
	if err != nil {
		t.Fatalf("Error putting value: %v", err)
	}
	t.Logf("delete key1 success")

}

func TestRestore(t *testing.T) {
	instance := "rocksdb1"
	opts := grocksdb.NewDefaultOptions()
	defer opts.Destroy()
	opts.SetCreateIfMissing(true)
	defer opts.Destroy()
	db, err := grocksdb.OpenDb(opts, instance)
	if err != nil {
		t.Fatalf("Error opening RocksDB: %v", err)
	}
	db.Close()

	// open BackupEngine
	be, err := grocksdb.OpenBackupEngine(opts, "rocksdb_backup")
	if err != nil {
		t.Fatalf("Error creating BackupEngine: %v", err)
	}
	defer be.Close()

	// Restore database from latest backup
	restoreOpts := grocksdb.NewRestoreOptions()
	defer restoreOpts.Destroy()
	err = be.RestoreDBFromLatestBackup(instance, instance, restoreOpts)
	if err != nil {
		t.Fatalf("Error restoring from latest backup: %v", err)
	}

	// 必须重置db，否则重置无效
	db, err = grocksdb.OpenDb(opts, instance)
	if err != nil {
		t.Fatalf("Error opening RocksDB: %v", err)
	}
	defer db.Close()
	ro := grocksdb.NewDefaultReadOptions()
	defer ro.Destroy()
	value1, err := db.Get(ro, []byte("key1"))
	if err != nil {
		t.Error("Error getting value1 from restored DB: ", err)
	} else {
		t.Log("Retrieved value1 from restored DB:", string(value1.Data()))
		value1.Free()
	}

	value2, err := db.Get(ro, []byte("key2"))
	if err != nil {
		t.Error("Error getting value2 from restored DB: ", err)
	} else {
		t.Log("Retrieved value2 from restored DB:", string(value2.Data()))
		value2.Free()
	}

	value3, err := db.Get(ro, []byte("key3"))
	if err != nil {
		t.Error("Error getting value3 from restored DB: ", err)
	} else {
		t.Log("Retrieved value3 from restored DB:", string(value3.Data()))
		value3.Free()
	}

}
