package tcpgossip

import (
	"fmt"
	"github.com/NBSChain/go-gossip-tcp/pbs"
	"math/rand"
	"sync"
	"time"
)

type viewMap struct {
	sync.RWMutex
	views map[string]*viewEntity
}

func newViewMap() *viewMap {
	v := &viewMap{
		views: make(map[string]*viewEntity),
	}
	return v
}

func (m *viewMap) IsEmpty() bool {
	m.RLock()
	defer m.RUnlock()
	return len(m.views) == 0
}

func (m *viewMap) Keys() []string {
	ins := make([]string, 0)

	m.RLock()
	defer m.RUnlock()

	for id := range m.views {
		ins = append(ins, id)
	}
	return ins
}

func (m *viewMap) Value(key string) (*viewEntity, bool) {
	m.RLock()
	defer m.RUnlock()
	item, ok := m.views[key]
	return item, ok
}

func (m *viewMap) Clear() {
	m.Lock()
	defer m.Unlock()

	for id, item := range m.views {
		delete(m.views, id)
		item.Close()
	}
	m.views = nil
}

func (m *viewMap) AverageProb() float64 {
	m.RLock()
	defer m.RUnlock()

	if len(m.views) == 0 {
		return 1.0
	}

	var sum float64
	for _, item := range m.views {
		sum += item.probability
	}

	return sum / float64(len(m.views))
}

func (m *viewMap) NormalizeProb() {

	m.Lock()
	defer m.Unlock()

	if len(m.views) == 0 {
		return
	}

	var sum float64
	for _, item := range m.views {
		sum += item.probability
	}

	for _, item := range m.views {
		item.probability = item.probability / sum
	}
}

func (m *viewMap) Size() int {
	m.RLock()
	defer m.RUnlock()
	return len(m.views)
}

func (m *viewMap) GetRandomNodeByProb() *viewEntity {
	rand.Seed(time.Now().UnixNano())
	var (
		p           = rand.Float64()
		sum         = 0.0
		index       = 0
		defaultNode *viewEntity
	)
	logger.Debug("random mode prob:->", p)
	m.RLock()
	defer m.RUnlock()

	for _, item := range m.views {

		if index == 0 {
			defaultNode = item
		}
		index++

		sum += item.probability
		logger.Debug("total sum, prob:->", sum, item.probability, item.KeyString())

		if p < sum {
			return item
		}
	}

	return defaultNode
}

func (m *viewMap) ChoseRandom() *viewEntity {
	m.RLock()
	defer m.RUnlock()

	idx := rand.Intn(len(m.views))
	i := 0

	for _, item := range m.views {
		if i == idx {
			return item
		}
		i++
	}
	return nil
}

func (m *viewMap) Add(id string, node *viewEntity) {
	m.Lock()
	defer m.Unlock()

	m.views[id] = node
}

func (m *viewMap) Remove(id string) {
	m.Lock()
	defer m.Unlock()

	if item, ok := m.views[id]; ok {
		logger.Debug("remove from out put view :->", item.nodeID)
		delete(m.views, id)
		item.Close()
	}
}

func (m *viewMap) SendNewWeight(id string, t gsp_tcp.MsgType) {
	m.Lock()
	defer m.Unlock()

	var sum float64
	for _, item := range m.views {
		sum += item.probability
	}

	for _, item := range m.views {
		item.probability = item.probability / sum
		item.send(m.UpdateMsg(id, t, item.probability))
	}
}

func (m *viewMap) UpdateLocalWeight(msg *gsp_tcp.CtrlMsg) error {
	m.RLock()
	defer m.RUnlock()

	up := msg.UpdateWeight
	nodeId := up.NodeId

	item, ok := m.views[nodeId]
	if !ok {
		return fmt.Errorf("update in view err, no such item(%s):->", item.KeyString())
	}
	item.probability = up.Weight
	logger.Debug("item in in view get updated:->", item.KeyString())

	return nil
}

func (m *viewMap) AllViews() map[string]*viewEntity {
	return m.views
}
