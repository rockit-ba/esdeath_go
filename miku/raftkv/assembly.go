package raftkv

import (
	"esdeath_go/miku/base"
	"esdeath_go/miku/persistent"
	"fmt"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	"io"
	"os"
)

type fsm RaftKV

var fsmLog = base.BuildLogger("fsm", hclog.Debug, os.Stdout)

// FSM,一旦大多数节点提交了log entry，就会调用Apply。apply 属于应用层的行为
func (f *fsm) Apply(l *raft.Log) interface{} {
	fsmLog.Info(fmt.Sprintf("apply log entry: %d", l.Index))
	c := &persistent.Command{}
	c.Decode(l)

	switch c.Op {
	case Insert:
		return f.Insert(c.Kvps)
	case Delete:
		return f.Delete(c.Kvps)
	default:
		log.Error(fmt.Sprintf("unrecognized command op: %s", c.Op))
		os.Exit(1)
		return nil
	}
}

func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	log.Info("into Snapshot")
	defer func() {
		log.Info("out Snapshot")
	}()
	return f, nil
}

func (f *fsm) Restore(rc io.ReadCloser) error {
	log.Info("into Restore")
	defer func() {
		log.Info("out Restore")
	}()
	// 读取快照数据
	return f.RaftLoad(rc)
}

func (f *fsm) Persist(sink raft.SnapshotSink) error {
	err := func() error {
		if err := f.RaftBackup(sink); err != nil {
			return err
		}
		return sink.Close()
	}()

	if err != nil {
		log.Error("failed to RaftBackup", "error", err.Error())
		if err := sink.Cancel(); err != nil {
			return err
		}
	}
	return err
}

// Release 则是快照处理完成后的回调，不需要的话可以实现为空函数。
func (f *fsm) Release() {}
