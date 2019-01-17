package tcpgossip

import (
	"github.com/NBSChain/go-gossip-tcp/pbs"
	"github.com/golang/protobuf/proto"
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

func (node *GspCtrlNode) SubMsg() []byte {

	data, _ := proto.Marshal(&gsp_tcp.CtrlMsg{
		Type: gsp_tcp.MsgType_SubInit,
		SubAck: &gsp_tcp.ID{
			NodeId: node.nodeId,
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
