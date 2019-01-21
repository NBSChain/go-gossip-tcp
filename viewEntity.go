package tcpgossip

import (
	"context"
	"github.com/NBSChain/go-gossip-tcp/pbs"
	"github.com/gogo/protobuf/proto"
	"net"
	"sync"
	"time"
)

type ViewEntity struct {
	sync.RWMutex
	ok bool

	peerID string
	peerIP string

	ctx      context.Context
	closer   context.CancelFunc
	conn     net.Conn
	pareNode *GspCtrlNode

	probability   float64
	heartBeatTime time.Time
	expiredTime   time.Time
}

func (e *ViewEntity) reading() {

	logger.Debug("start to read......")
	defer e.Close()

	for {

		buffer := make([]byte, conf.GossipControlMessageSize)
		n, err := e.conn.Read(buffer)

		if err != nil {
			logger.Warning("connection node read err:->", e.peerID)
			return
		}

		msg := &gsp_tcp.CtrlMsg{}
		if err := proto.Unmarshal(buffer[:n], msg); err != nil {
			return
		}

		logger.Debug("view entity node received :->", msg)

		if msg.Type == gsp_tcp.MsgType_HeartBeat {
			e.Lock()
			e.heartBeatTime = time.Now()
			e.Unlock()

		} else {
			e.pareNode.msgTask <- msg
		}
	}
}

func (node *GspCtrlNode) newViewEntity(c net.Conn, ip, id string) *ViewEntity {

	logger.Debug("create a new item :->", ip, id)
	ctx, cancel := context.WithCancel(context.Background())
	e := &ViewEntity{
		pareNode:      node,
		probability:   node.averageProbability(),
		ctx:           ctx,
		closer:        cancel,
		conn:          c,
		ok:            true,
		peerID:        id,
		peerIP:        ip,
		expiredTime:   time.Now().Add(conf.ExpireTime),
		heartBeatTime: time.Now(),
	}

	go e.reading()
	return e
}

func (e *ViewEntity) send(msg []byte) error {

	if _, err := e.conn.Write(msg); err != nil {
		logger.Warning("send msg err :->", err)
		return err
	}
	return nil
}

func (e *ViewEntity) Close() {
	e.Lock()
	defer e.Unlock()

	if !e.ok {
		logger.Debug("try to cancel a closed connection node:->", e.peerID)
		return
	}

	logger.Info("the connection node closed:->", e.peerID)

	e.ok = false
	e.closer()

	if err := e.conn.Close(); err != nil {
		logger.Warning("failed to cancel connection node:->", e.peerID)
	}
}

func (node *GspCtrlNode) removeViewEntity(id string) {

	if item, ok := node.outView[id]; ok {
		logger.Debug("remove from out put view :->", item.peerID)
		delete(node.outView, id)
		item.Close()
	}

	if item, ok := node.inView[id]; ok {
		logger.Debug("remove from in put view :->", item.peerID)
		delete(node.outView, id)
		item.Close()
	}

	node.ShowViews()

	if len(node.inView) == 0 {
		if err := node.Subscribe(node.SubMsg(true)); err != nil {
			logger.Warning("resubscribe err:->", err)
		}
		logger.Debug("no input view entities and resubscribe now")
	}
}

func (node *GspCtrlNode) sendHeartBeat() {

	data := node.HeartBeatMsg()
	now := time.Now()

	node.outLock.RLock()
	defer node.outLock.RUnlock()

	for id, item := range node.outView {

		if now.After(item.expiredTime) {
			logger.Warning("subscribe expired:->", id)
			node.removeViewEntity(id)
			continue
		}

		if _, err := item.conn.Write(data); err != nil {
			logger.Warning("sending heart beat err:->", id, err)
			node.removeViewEntity(id)
		}
	}
}
