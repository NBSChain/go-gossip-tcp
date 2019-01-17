package tcpgossip

import (
	"github.com/NBSChain/go-gossip-tcp/pbs"
	"github.com/NBSChain/go-nbs/utils"
	"github.com/gogo/protobuf/proto"
	"net"
	"time"
)

var (
	conf   *GspConf = nil
	logger          = utils.GetLogInstance()
)

type GspConf struct {
	GenesisIP                string //
	TCPServicePort           int    //= 13001
	GossipControlMessageSize int
	SubTimeOut               time.Duration
}

type GspCtrlNode struct {
	nodeId      string
	serviceConn *net.TCPListener
	outView     map[string]*ViewEntity
	inView      map[string]*ViewEntity
}

func NewGspNode(nodeId string) *GspCtrlNode {

	node := &GspCtrlNode{
		nodeId:  nodeId,
		outView: make(map[string]*ViewEntity),
		inView:  make(map[string]*ViewEntity),
	}
	return node
}

func (node *GspCtrlNode) Init(c *GspConf) error {
	conf = c

	conn, err := net.ListenTCP("tcp4", &net.TCPAddr{
		Port: conf.TCPServicePort,
	})

	if err != nil {
		logger.Warning("start tcp service failed:->", err)
		return err
	}
	node.serviceConn = conn

	if err := node.Subscribe(); err != nil {
		logger.Warning("failed to subscribe any contact:->", err)
		return err
	}

	return nil
}

func (node *GspCtrlNode) Run() {

	logger.Info("tcp gossip node start to run.....")

	for {
		conn, err := node.serviceConn.Accept()
		if err != nil {
			logger.Error("service is done:->", err)
			return
		}
		go node.process(conn)
	}
}

func (node *GspCtrlNode) process(conn net.Conn) {
	defer conn.Close()

	logger.Debug("one process create:->", conn.RemoteAddr().String())

	buffer := make([]byte, conf.GossipControlMessageSize)
	n, err := conn.Read(buffer)

	if err != nil {
		logger.Error("get a weak tcp conn:->", err)
		return
	}

	msg := &gsp_tcp.CtrlMsg{}
	if err := proto.Unmarshal(buffer[:n], msg); err != nil {
		logger.Warning("unknown gossip msgManager:->", err)
		return
	}

	switch msg.Type {
	case gsp_tcp.MsgType_SubInit:
		err = node.asGenesisNode(msg, conn)
	}

	logger.Debug("one process exit:->", conn.RemoteAddr().String(), err)
}
