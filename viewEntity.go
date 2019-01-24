package tcpgossip

import (
	"github.com/NBSChain/go-gossip-tcp/pbs"
	"github.com/gogo/protobuf/proto"
	"net"
	"sync"
	"time"
)

type viewEntity struct {
	sync.RWMutex
	ok bool

	nodeID string
	peerIP string

	conn     net.Conn
	pareNode *GspCtrlNode

	probability   float64
	heartBeatTime time.Time
	expiredTime   time.Time
}

func (e *viewEntity) reading() {

	logger.Debug("start to read......")
	defer e.pareNode.removeViewEntity(e.nodeID)

	for {

		buffer := make([]byte, conf.GossipControlMessageSize)
		n, err := e.conn.Read(buffer)

		if err != nil {
			logger.Warning("connection node read err:->", e.nodeID, err)
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
func (node *GspCtrlNode) newViewEntity(c net.Conn, ip, id string) *viewEntity {

	logger.Debug("create a new item :->", ip, id)
	e := &viewEntity{
		pareNode:      node,
		probability:   node.outView.AverageProb(),
		conn:          c,
		ok:            true,
		nodeID:        id,
		peerIP:        ip,
		expiredTime:   time.Now().Add(conf.ExpireTime),
		heartBeatTime: time.Now(),
	}

	go e.reading()
	return e
}

func (e *viewEntity) send(msg []byte) error {

	if _, err := e.conn.Write(msg); err != nil {
		logger.Warning("send msg err :->", err)
		e.pareNode.removeViewEntity(e.nodeID)
		return err
	}
	return nil
}

func (e *viewEntity) Close() {
	e.Lock()
	defer e.Unlock()

	if !e.ok {
		logger.Debug("try to cancel a closed connection node:->", e.nodeID)
		return
	}
	logger.Info("the connection node closed:->", e.nodeID)

	e.ok = false
	if err := e.conn.Close(); err != nil {
		logger.Warning("failed to cancel connection node:->", e.nodeID)
	}
}

func (node *GspCtrlNode) removeViewEntity(id string) {

	node.outView.Remove(id)
	node.inView.Remove(id)

	node.ShowViews()

	if node.inView.IsEmpty() {
		if err := node.Subscribe(node.SubMsg(true)); err != nil {
			logger.Warning("resubscribe err:->", err)
		}
		logger.Debug("no input view entities and resubscribe now")
	}
}

func (node *GspCtrlNode) sendHeartBeat() {

	data := node.HeartBeatMsg()
	now := time.Now()

	for id, item := range node.outView.AllViews() {

		if now.After(item.expiredTime) {
			logger.Warning("subscribe expired:->", id)
			node.removeViewEntity(id)
			continue
		}
		item.send(data)
	}
}
