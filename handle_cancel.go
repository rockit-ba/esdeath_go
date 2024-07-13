package main

import (
	pb "esdeath_go/esdeath_proto"
	"esdeath_go/miku/persistent"
	log "github.com/sirupsen/logrus"
)

func cancelReqHandle(cancel *pb.DelayMsgCancel) (*pb.CancelDelayMsgResult, error) {
	tempMsgId := cancel.MsgId + keyDelimiter + defConsumerGroup
	item, err := newKeyEntityFromStr(tempMsgId)
	if err != nil {
		return nil, err
	}
	var pairs []*persistent.KVPair
	// 如果一个topic有多个consumerGroup，那么消息会全部删除
	if cgs, ok := topicToManyConsumerGroupsMap.Get(item.topic); ok {
		for _, consumeGroup := range cgs.Keys() {
			message := persistent.NewKPair(cancel.MsgId + keyDelimiter + consumeGroup)
			log.Debugln("取消延迟消息：", message.Key)
			pairs = append(pairs, message)
		}
	} else {
		message := persistent.NewKPair(tempMsgId)
		log.Debugln("取消延迟消息：", message.Key)
		pairs = append(pairs, message)
	}
	if err := rkv.Delete(pairs); err != nil {
		return nil, err
	}
	log.Debugln("取消延迟消息：", cancel.MsgId)
	return &pb.CancelDelayMsgResult{BaseResult: successResult()}, nil
}
