package tcpgossip

import (
	"fmt"
	"github.com/NBSChain/go-nbs/utils"
)

func (e *ViewEntity) String() string {
	format := utils.GetConfig().SysTimeFormat

	e.RLock()
	defer e.RUnlock()

	return fmt.Sprintf("------------%s------------\n"+
		"|%-15s:%20.2f|\n"+
		"|%-15s:%20s|\n"+
		"|%-15s:%20s|\n"+
		"|%-15s:%20s|\n"+
		"-----------------------------------------------------------------------\n",
		e.peerID,
		"probability",
		e.probability,
		"peerIP",
		e.peerIP,
		"heartBeatTime",
		e.heartBeatTime.Format(format),
		"expiredTime",
		e.expiredTime.Format(format),
	)
}

func (node *GspCtrlNode) ShowViews() {

	fmt.Println("------------out view------------")
	for _, item := range node.outView {
		fmt.Println(item.String())
	}

	fmt.Println("------------in view------------")
	for _, item := range node.inView {
		fmt.Println(item.String())
	}
}
