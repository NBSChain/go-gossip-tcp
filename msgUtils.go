package tcpgossip

import (
	"github.com/NBSChain/go-gossip-tcp/pbs"
	"github.com/golang/protobuf/proto"
	"net"
	"time"
)

func (node *GspCtrlNode) AckMSG() []byte {

	data, _ := proto.Marshal(&gsp_tcp.CtrlMsg{
		Type: gsp_tcp.MsgType_SubAck,
		SubAck: &gsp_tcp.ID{
			NodeId: node.nodeId,
		},
	})

	return data
}

func (node *GspCtrlNode) SubMsg(isReSub bool) []byte {

	data, _ := proto.Marshal(&gsp_tcp.CtrlMsg{
		Type: gsp_tcp.MsgType_SubInit,
		Subscribe: &gsp_tcp.Subscribe{
			NodeId:  node.nodeId,
			IsReSub: isReSub,
		},
	})
	return data
}

func (node *GspCtrlNode) VoteMSG(nodeId, ip string, ttl int32) []byte {
	data, _ := proto.Marshal(&gsp_tcp.CtrlMsg{
		Type: gsp_tcp.MsgType_Forward,
		Vote: &gsp_tcp.Vote{
			NodeId: nodeId,
			IP:     ip,
			TTL:    ttl,
		},
	})

	return data
}

func (node *GspCtrlNode) FwdSubMSG(nodeId, ip string) []byte {
	data, _ := proto.Marshal(&gsp_tcp.CtrlMsg{
		Type: gsp_tcp.MsgType_Forward,
		Forward: &gsp_tcp.IDWithIP{
			NodeId: nodeId,
			IP:     ip,
		},
	})

	return data
}

func (node *GspCtrlNode) ContactMsg(ip string) []byte {
	data, _ := proto.Marshal(&gsp_tcp.CtrlMsg{
		Type: gsp_tcp.MsgType_GotContact,
		GotContact: &gsp_tcp.IDWithIP{
			NodeId: node.nodeId,
			IP:     ip,
		},
	})
	return data
}

func (node *GspCtrlNode) HeartBeatMsg() []byte {
	data, _ := proto.Marshal(&gsp_tcp.CtrlMsg{
		Type: gsp_tcp.MsgType_HeartBeat,
		HeartBeat: &gsp_tcp.ID{
			NodeId: node.nodeId,
		},
	})
	return data
}

func (node *GspCtrlNode) pingPongMsg(lAddr, rAddr *net.TCPAddr, timeOut time.Duration, data []byte) (*gsp_tcp.CtrlMsg, error) {

	conn, err := net.DialTCP("tcp4", lAddr, rAddr)
	if err != nil {
		logger.Warning("connect to genesis node err:->", err)
		return nil, err
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(timeOut)); err != nil {
		return nil, err
	}

	if _, err := conn.Write(data); err != nil {
		logger.Warning("write subscribe data err:->", err)
		return nil, err
	}

	buffer := make([]byte, conf.GossipControlMessageSize)
	n, err := conn.Read(buffer)
	if err != nil {
		logger.Warning("subscribe err:->", err)
		return nil, err
	}

	msg := &gsp_tcp.CtrlMsg{}
	if err := proto.Unmarshal(buffer[:n], msg); err != nil {
		return nil, err
	}

	return msg, nil
}
