package tcpgossip

import (
	"context"
	"github.com/NBSChain/go-gossip-tcp/pbs"
	"github.com/gogo/protobuf/proto"
	"net"
)

const (
	GossipControlMessageSize = 1 << 12
)

type connNode struct {
	ok     bool
	peerId string
	ctx    context.Context
	closer context.CancelFunc
	conn   net.Conn
}

func (node *connNode) reading() {

	logger.Debug("start to read......")
	defer node.Close()

	for {

		buffer := make([]byte, GossipControlMessageSize)
		n, err := node.conn.Read(buffer)

		if err != nil {
			logger.Warning("connection node read err:->", node.peerId)
			return
		}

		msg := &gsp_tcp.CtrlMsg{}
		if err := proto.Unmarshal(buffer[:n], msg); err != nil {
			return
		}

		logger.Debug("connection node received control message:->", msg)
	}
}

func NewConnNode(c net.Conn) *connNode {

	ctx, cancel := context.WithCancel(context.Background())
	node := &connNode{
		ctx:    ctx,
		closer: cancel,
		conn:   c,
		ok:     true,
	}

	go node.reading()
	return node
}

func (node *connNode) Close() {

	if !node.ok {
		logger.Debug("try to close a closed connection node:->", node.peerId)
		return
	}

	logger.Info("the connection node exit:->", node.peerId)

	node.ok = false

	node.closer()

	if err := node.conn.Close(); err != nil {
		logger.Warning("failed to close connection node:->", node.peerId)
	}
}
