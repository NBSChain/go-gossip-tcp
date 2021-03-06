package tcpgossip

import (
	"fmt"
	"github.com/NBSChain/go-gossip-tcp/pbs"
	"net"
)

func (node *GspCtrlNode) Subscribe(data []byte) error {

	rAddr := &net.TCPAddr{
		Port: conf.TCPServicePort,
		IP:   net.ParseIP(conf.GenesisIP),
	}

	msg, err := node.pingPongMsg(nil, rAddr, conf.CtrlMsgTimeOut, data)
	if err != nil {
		return err
	}

	if msg.Type != gsp_tcp.MsgType_SubAck {
		return fmt.Errorf("no available genesis node(%s):->", msg)
	}

	if msg.SubAck.NodeId == node.nodeId {
		return ESelfReq
	}

	logger.Debug("he will proxy our subscribe request:->", msg.SubAck.NodeId)

	return nil
}

func (node *GspCtrlNode) subSuccess(msg *gsp_tcp.CtrlMsg, conn net.Conn) error {

	logger.Debug("oh, I find my contact:->", msg)

	contact := msg.GotContact
	nodeId := contact.NodeId

	if _, ok := node.inView.Value(nodeId); ok {
		logger.Warning("duplicate contact notification:->", nodeId)
		return fmt.Errorf("duplicate contact notification(%s):->", nodeId)
	}

	ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	e := node.newViewEntity(conn, ip, contact.NodeId)

	node.inView.Add(nodeId, e)
	node.outView.Add(nodeId, e)

	node.ShowViews()

	return nil
}

func (node *GspCtrlNode) beWelcomed(msg *gsp_tcp.CtrlMsg, conn net.Conn) error {

	welcome := msg.Welcome
	nodeId := welcome.NodeId

	ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	e := node.newViewEntity(conn, ip, welcome.NodeId)

	node.inView.Add(nodeId, e)

	logger.Debug("thanks for your welcome:->", nodeId)
	node.ShowViews()

	return nil
}
