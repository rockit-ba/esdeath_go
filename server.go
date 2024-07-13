package main

import (
	"context"
	"errors"
	pb "esdeath_go/esdeath_proto"
	"esdeath_go/miku"
	"fmt"
	"github.com/hashicorp/raft"
	cmap "github.com/orcaman/concurrent-map/v2"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"math/rand"
	rand2 "math/rand/v2"
	"net"
	"sync"
	"time"
)

var (
	rkv *miku.RMikuKv
	// key 是topic，value是consumerGroups。用于处理一个topic下有多个consumerGroups的情况
	topicToManyConsumerGroupsMap = cmap.New[*cmap.ConcurrentMap[string, struct{}]]()
)

func Start() error {
	rkv = miku.Start()
	hostAddr := Conf.ServerAddr
	lis, err := net.Listen("tcp", hostAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", hostAddr, err)
	}

	serv := grpc.NewServer()
	esdeathSer := &esdeathServer{
		pullHoldNotifyContainer: cmap.New[chan struct{}](),
		pullLock:                &PullServiceLock{groups: cmap.New[*ServiceLock]()},
	}
	pb.RegisterEsdeathServer(serv, esdeathSer)
	if err := serv.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve on %s: %v", hostAddr, err)
	}
	log.Infoln("esdeathServer listening at", lis.Addr())
	return nil
}

func GracefulStop() {
	if err := rkv.Shutdown(); err != nil {
		log.Errorln("Error shutting down kv: ", err)
	}
	log.Infoln("kv shutdown success")

	if err := rkv.RaftShutdown(); err != nil {
		log.Errorln("Error shutting down raft: ", err)
	}
	log.Infoln("raft shutdown success")
}

// 提供对client的交互grpc服务
type esdeathServer struct {
	pb.UnimplementedEsdeathServer
	// 确保一个消息不会被同时消费，不同topic消费是并发的
	pullLock *PullServiceLock
	// 长轮询通知容器：key 为服务名，value 为通知结束hold chan
	// 在消息未达到延迟时间的时候，长轮询的请求会被hold住，直到消息到达延迟时间或者有新消息到达
	pullHoldNotifyContainer cmap.ConcurrentMap[string, chan struct{}]
}

// AddDelayMsg 添加延迟消息
// 添加延迟消息首先会新增一条延迟消息，然后会尝试唤醒长轮询的请求
func (es *esdeathServer) AddDelayMsg(c context.Context, r *pb.DelayMsgAdd) (*pb.AddDelayMsgResult, error) {
	log.Infoln("AddDelayMsg: ", r)
	result, err := produceReqHandle(r)
	if err != nil {
		log.Errorln("Error handling AddDelayMsg: ", err)
		return &pb.AddDelayMsgResult{BaseResult: errorRespHandle(err)}, nil
	}
	tryNotifyPullHold(es, r.Topic)
	return result, nil
}

func tryNotifyPullHold(es *esdeathServer, topic string) {
	if notifyCh, ok := es.pullHoldNotifyContainer.Pop(topic); ok && notifyCh != nil {
		select {
		case notifyCh <- struct{}{}:
		default:
		}
	}
}

func (es *esdeathServer) PullDelayMsg(c context.Context, r *pb.DelayMsgPull) (*pb.PullDelayMsgResult, error) {
	doTopicToManyConsumerGroup(r)
	lock := es.pullLock.getOrAddLock(newServiceLock(r.Topic, r.ConsumerGroup))
	lock.Lock()
	defer lock.Unlock()
	log.Debugln("PullDelayMsg into: ", r)

	var (
		delayTimestamp int64
		msg            *PulledMsg
		err            error
	)
	isHold := false
	for {
		if !isHold {
			msg, err = pullReqHandle(r)
			isHold = true
		} else {
			msg, err = es.holdProcess(r, delayTimestamp)
		}
		if err != nil {
			return &pb.PullDelayMsgResult{BaseResult: errorRespHandle(err)}, nil
		}
		if msg == nil {
			// 没有任何消息则设置默认的延迟时间
			delayTimestamp = time.Now().UnixMilli() + defHoldTime.Milliseconds() + rand2.Int64N(5000)
		} else {
			delayTimestamp = msg.DelayTimestamp
		}
		if msg != nil && msg.MsgId != "" {
			return &pb.PullDelayMsgResult{
				BaseResult: successResult(),
				Topic:      msg.Topic,
				Tag:        msg.Tag,
				MsgId:      msg.MsgId,
				Payload:    string(msg.Payload),
			}, nil
		}
	}
}

func doTopicToManyConsumerGroup(r *pb.DelayMsgPull) {
	newConsumerGroupSet := cmap.New[struct{}]()
	if topicToManyConsumerGroupsMap.SetIfAbsent(r.Topic, &newConsumerGroupSet) {
		newConsumerGroupSet.Set(r.ConsumerGroup, struct{}{})
		log.Debugln("add new topic: ", r.Topic)
		return
	}
	oldConsumerGroupSet, _ := topicToManyConsumerGroupsMap.Get(r.Topic)
	oldConsumerGroupSet.Set(r.ConsumerGroup, struct{}{})
	log.Debugln("add group: ", r.ConsumerGroup)
}

var holdEndError = errors.New("hold end")

// 长轮询处理
func (es *esdeathServer) holdProcess(r *pb.DelayMsgPull, delayTimestamp int64) (*PulledMsg, error) {
	notify := make(chan struct{}, 1)
	es.pullHoldNotifyContainer.Set(r.Topic, notify)
	holdTime := time.Duration(delayTimestamp-time.Now().UnixMilli()) * time.Millisecond
	log.Infoln("hold time: ", holdTime)
	select {
	case <-time.After(holdTime):
		es.pullHoldNotifyContainer.Pop(r.Topic)
		return nil, holdEndError
	case <-notify:
		return pullReqHandle(r)
	}
}

func (*esdeathServer) AckDelayMsg(c context.Context, r *pb.DelayMsgAck) (*pb.AckDelayMsgResult, error) {
	log.Infoln("AckDelayMsg: ", r)
	if err := reqConsumeResultHandle(r); err != nil {
		return &pb.AckDelayMsgResult{BaseResult: errorRespHandle(err)}, nil
	}
	return &pb.AckDelayMsgResult{BaseResult: successResult()}, nil
}

func (*esdeathServer) CancelDelayMsg(c context.Context, r *pb.DelayMsgCancel) (*pb.CancelDelayMsgResult, error) {
	log.Infoln("CancelDelayMsg: ", r)
	result, err := cancelReqHandle(r)
	if err != nil {
		return &pb.CancelDelayMsgResult{BaseResult: errorRespHandle(err)}, nil
	}
	return result, nil
}

func (*esdeathServer) GetRole(c context.Context, r *emptypb.Empty) (*pb.RoleResult, error) {
	log.Infoln("GetRole: ", r)
	result := &pb.RoleResult{
		BaseResult: successResult(),
		Role:       pb.Role_FOLLOWER,
	}
	if rkv.IsLeader() {
		result.Role = pb.Role_LEADER
	} else {
		time.Sleep(time.Duration(1+rand.Intn(3)) * time.Second)
	}
	return result, nil
}

func errorRespHandle(err error) *pb.BaseResult {
	if errors.Is(err, holdEndError) {
		return &pb.BaseResult{
			Status: pb.ResultStatus_NO_MESSAGE,
			Desc:   "no message",
		}
	}

	if errors.Is(err, raft.ErrNotLeader) || errors.Is(err, raft.ErrLeadershipLost) {
		return &pb.BaseResult{
			Status: pb.ResultStatus_REFUSE_BY_FOLLOWER,
			Desc:   successDesc,
		}
	}

	return &pb.BaseResult{
		Status: pb.ResultStatus_FAIL,
		Desc:   err.Error(),
	}
}

const successDesc = "success"

func successResult() *pb.BaseResult {
	return &pb.BaseResult{
		Status: pb.ResultStatus_SUCCESS,
		Desc:   successDesc,
	}
}

type PullServiceLock struct {
	groups cmap.ConcurrentMap[string, *ServiceLock]
}

func (p *PullServiceLock) getOrAddLock(newLock *ServiceLock) *ServiceLock {
	if p.groups.SetIfAbsent(newLock.lockKey(), newLock) {
		return newLock
	}
	lock, _ := p.groups.Get(newLock.lockKey())
	return lock
}

// topic + consumerGroup 级别的锁,如果只有一个topic消费，则默认使用defConsumerGroup
type ServiceLock struct {
	sync.Mutex
	Topic         string
	ConsumerGroup string
}

func (s *ServiceLock) lockKey() string {
	return s.Topic + s.ConsumerGroup
}

func newServiceLock(topic, consumeGroup string) *ServiceLock {
	if consumeGroup == "" {
		consumeGroup = defConsumerGroup
	}
	return &ServiceLock{Topic: topic, ConsumerGroup: consumeGroup}
}
