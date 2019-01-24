package tcpgossip

import (
	"fmt"
	"github.com/NBSChain/go-nbs/utils"
)

func (e *viewEntity) String() string {
	format := utils.GetConfig().SysTimeFormat

	e.RLock()
	defer e.RUnlock()

	return fmt.Sprintf("------------%s------------\n"+
		"|%-15s:%20.2f|\n"+
		"|%-15s:%20s|\n"+
		"|%-15s:%20s|\n"+
		"|%-15s:%20s|\n"+
		"-----------------------------------------------------------------------\n",
		e.nodeID,
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
	for _, item := range node.outView.AllViews() {
		fmt.Println(item.String())
	}

	fmt.Println("------------in view------------")
	for _, item := range node.inView.AllViews() {
		fmt.Println(item.String())
	}
}

func (e *viewEntity) KeyString() string {
	return fmt.Sprintf("\nnodeId(%s)--->ip(%s)--->prob(%f)\n", e.nodeID, e.peerIP, e.probability)
}
