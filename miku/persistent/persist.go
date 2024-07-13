// Package persistent 持久化抽象层，对外暴露的接口，不引用本工程其他模块
package persistent

import (
	"esdeath_go/miku/base"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	"io"
	"os"
)

var log = base.BuildLogger("persist", hclog.Info, os.Stdout)
var PersistInstance DbPersist

// DbPersist 持久化抽象层
type DbPersist interface {
	GetOne(key []byte) (val []byte, err error)
	PrefixIterate(prefix []byte, consumeCallback func(k, val []byte) bool)
	Insert(kvPairs []*KVPair) error
	Delete(kvPairs []*KVPair) error
	Shutdown() error
	RaftFSM
}

type RaftFSM interface {
	Backup(sink raft.SnapshotSink) error
	ReLoad(rc io.ReadCloser) error
}

func (c *Command) Decode(l *raft.Log) {
	if err := proto.Unmarshal(l.Data, c); err != nil {
		log.Error(fmt.Sprintf("failed to unmarshal command: %s", err.Error()))
		os.Exit(1)
	}
}

func (c *Command) Encode() ([]byte, error) {
	return proto.Marshal(c)
}

func NewKPair(key string) *KVPair {
	return NewKVPair(key, "")
}

func NewKVPair(key, value string) *KVPair {
	return &KVPair{Key: key, Value: value}
}
