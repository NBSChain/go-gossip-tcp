package tcpgossip

import (
	"fmt"
	"github.com/NBSChain/go-gossip-tcp/pbs"
	"github.com/NBSChain/go-nbs/utils/crypto"
	"github.com/golang/protobuf/proto"
	"net"
	"time"
)

func (node *GspCtrlNode) SubAckMSg() []byte {

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
		Type: gsp_tcp.MsgType_VoteContact,
		Vote: &gsp_tcp.Vote{
			NodeId: nodeId,
			IP:     ip,
			TTL:    ttl,
		},
	})

	return data
}

func (node *GspCtrlNode) FwdSubMSG(nodeId, ip string) []byte {

	msgID := fmt.Sprintf("%s%d", nodeId, time.Now().Unix())

	data, _ := proto.Marshal(&gsp_tcp.CtrlMsg{
		Type: gsp_tcp.MsgType_Forward,
		Forward: &gsp_tcp.ForwardMsg{
			NodeId: nodeId,
			IP:     ip,
			MsgId:  crypto.MD5SS(msgID),
		},
	})

	return data
}

func (node *GspCtrlNode) ContactMsg() []byte {

	data, _ := proto.Marshal(&gsp_tcp.CtrlMsg{
		Type: gsp_tcp.MsgType_GotContact,
		GotContact: &gsp_tcp.ID{
			NodeId: node.nodeId,
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

func (node *GspCtrlNode) WelcomeMsg() []byte {
	data, _ := proto.Marshal(&gsp_tcp.CtrlMsg{
		Type: gsp_tcp.MsgType_WelCome,
		Welcome: &gsp_tcp.ID{
			NodeId: node.nodeId,
		},
	})

	return data
}

func (node *GspCtrlNode) UpdateMsg(t gsp_tcp.MsgType, w float64) []byte {

	data, _ := proto.Marshal(&gsp_tcp.CtrlMsg{
		Type: t,
		UpdateWeight: &gsp_tcp.Weight{
			NodeId: node.nodeId,
			Weight: w,
		},
	})

	return data
}

func (node *GspCtrlNode) pingPongMsg(lAddr, rAddr *net.TCPAddr, timeOut time.Duration, data []byte) (*gsp_tcp.CtrlMsg, error) {

	conn, err := net.DialTCP("tcp4", lAddr, rAddr)
	if err != nil {
		logger.Warning("connect to remote err:->", err, rAddr.String())
		return nil, err
	}
	defer conn.Close()

	if timeOut > 0 {
		if err := conn.SetDeadline(time.Now().Add(timeOut)); err != nil {
			return nil, err
		}
	}

	if _, err := conn.Write(data); err != nil {
		logger.Warning("write data err:->", err)
		return nil, err
	}

	buffer := make([]byte, conf.GossipControlMessageSize)
	n, err := conn.Read(buffer)
	if err != nil {
		logger.Warning("read err:->", err)
		return nil, err
	}

	msg := &gsp_tcp.CtrlMsg{}
	if err := proto.Unmarshal(buffer[:n], msg); err != nil {
		return nil, err
	}

	return msg, nil
}

func (node *GspCtrlNode) pingMsg(lAddr, rAddr *net.TCPAddr, timeOut time.Duration, data []byte) (*net.TCPConn, error) {

	conn, err := net.DialTCP("tcp4", lAddr, rAddr)
	if err != nil {
		logger.Warning("connect to remote err:->", err, rAddr.String())
		return nil, err
	}

	if timeOut > 0 {
		conn.SetWriteDeadline(time.Now().Add(timeOut))
	}

	if _, err := conn.Write(data); err != nil {
		conn.Close()
		logger.Warning("write data err:->", err)
		return nil, err
	}

	if timeOut > 0 {
		conn.SetWriteDeadline(NoTimeOut)
	}

	return conn, nil
}
