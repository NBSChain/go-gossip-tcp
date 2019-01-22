package tcpgossip

import (
	"sync"
)

type TcpGossip interface {
	Init(c *GspConf) error
	Run()
	Destroy()
}

var instance *GspCtrlNode
var once sync.Once

func GetInstance() TcpGossip {

	once.Do(func() {
		instance = newGspNode()
	})

	return instance
}
