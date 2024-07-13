package main

import (
	esdeathproto "esdeath_go/esdeath_proto"
	"esdeath_go/miku/persistent"
	log "github.com/sirupsen/logrus"
)

func reqConsumeResultHandle(ack *esdeathproto.DelayMsgAck) error {
	msgKey := ack.MsgId
	switch ack.Status {
	case esdeathproto.AckStatus_ACK:
		message := persistent.NewKPair(msgKey)
		if err := rkv.Delete([]*persistent.KVPair{message}); err != nil {
			log.Errorln("删除消息失败：", err)
			return err
		}
		log.Infoln("删除消息：", msgKey)
	case esdeathproto.AckStatus_RECONSUME:
		// 重新消费,将消息状态设置为待消费
		msg, err := rkv.Query(msgKey)
		if err != nil {
			log.Errorln("根据消息key查询消息失败：", err)
			return err
		}
		// 重新设置为 waitConsume
		msg = waitConsume.toString() + msg[1:]
		updateMsg := persistent.NewKVPair(msgKey, msg)
		if err = rkv.Insert([]*persistent.KVPair{updateMsg}); err != nil {
			log.Errorln("重置消息待消费失败：", err)
			return err
		}
		log.Debugln("重新消费：", ack.MsgId)
	}
	return nil
}
