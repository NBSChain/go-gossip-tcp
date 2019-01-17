package tcpgossip

import (
	"fmt"
	"github.com/NBSChain/go-gossip-tcp/pbs"
	"github.com/gogo/protobuf/proto"
	"net"
	"time"
)

func (node *GspCtrlNode) Subscribe() error {

	conn, err := net.DialTCP("tcp4", nil, &net.TCPAddr{
		Port: conf.TCPServicePort,
		IP:   net.ParseIP(conf.GenesisIP),
	})
	if err != nil {
		logger.Warning("connect to genesis node err:->", err)
		return err
	}
	defer conn.Close()

	data := node.SubMsg()

	if _, err := conn.Write(data); err != nil {
		logger.Warning("write subscribe data err:->", err)
		return err
	}

	conn.SetReadDeadline(time.Now().Add(conf.SubTimeOut))
	buffer := make([]byte, conf.GossipControlMessageSize)
	n, err := conn.Read(buffer)
	if err != nil {
		logger.Warning("subscribe err:->", err)
		return err
	}

	msg := &gsp_tcp.CtrlMsg{}
	proto.Unmarshal(buffer[:n], msg)

	if msg.Type != gsp_tcp.MsgType_SubAck {
		return fmt.Errorf("no available genesis node(%s):->", msg)
	}

	logger.Debug("he will proxy our subscribe request:->", msg.SubAck.NodeId)
	return nil
}
