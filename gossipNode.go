package tcpgossip

import (
	"github.com/NBSChain/go-nbs/utils"
	"net"
)

var logger = utils.GetLogInstance()

const (
	TCPServicePort = 13001
)

type GspCtlNode struct {
	serviceConn *net.TCPListener
}

func NewGspNode() *GspCtlNode {
	node := &GspCtlNode{}

	return node
}

func (node *GspCtlNode) Init() error {
	conn, err := net.ListenTCP("tcp4", &net.TCPAddr{
		Port: TCPServicePort,
	})

	if err != nil {
		logger.Warning("start tcp service failed:->", err)
		return err
	}

	node.serviceConn = conn

	return nil
}

func (node *GspCtlNode) Run() {
	for {
		conn, err := node.serviceConn.Accept()
		if err != nil {
			logger.Error("service is done:->", err)
			return
		}

		NewConnNode(conn)
	}
}

func (node *GspCtlNode) runLoop() {

}
