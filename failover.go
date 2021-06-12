package forest

import (
	"fmt"
	"strings"
	"time"

	"github.com/admpub/log"
)

// fail over the job snapshot when the task client

type JobSnapshotFailOver struct {
	node                   *JobNode
	deleteClientEventChans chan *JobClientDeleteEvent
}

// new job snapshot fail over
func NewJobSnapshotFailOver(node *JobNode) (f *JobSnapshotFailOver) {
	f = &JobSnapshotFailOver{
		node:                   node,
		deleteClientEventChans: make(chan *JobClientDeleteEvent, 50),
	}
	f.loop()
	return
}

// loop
func (f *JobSnapshotFailOver) loop() {
	go func() {
		for ch := range f.deleteClientEventChans {
			f.handleJobClientDeleteEvent(ch)
		}
	}()
}

// handle job client delete event
func (f *JobSnapshotFailOver) handleJobClientDeleteEvent(event *JobClientDeleteEvent) {
	prefixKey := fmt.Sprintf(JobClientSnapshotPath, event.Group.name, event.Client.name)
	handler := f.generateJobSnapshotDeleteHandler(event, prefixKey)
	for i := 1; i <= 10; i++ {
		err := f.node.etcd.GetWithPrefixKeyChunk(prefixKey, 501, handler)
		if err == nil {
			return
		}
		log.Errorf("the fail client: %v for path: %s, error must retry", event.Client.name, prefixKey)
		time.Sleep(time.Second * time.Duration(i))
	}
}

func (f *JobSnapshotFailOver) generateJobSnapshotDeleteHandler(event *JobClientDeleteEvent, prefixKey string) func(key, val []byte) error {
	return func(key, val []byte) error {
		client, err := event.Group.selectClient()
		if err != nil {
			log.Warnf("%v", err)
			return nil
		}

		from := string(key)
		value := string(val)

		// 新地址
		to := fmt.Sprintf(JobClientSnapshotPath, event.Group.name, client.name) + strings.TrimPrefix(from, prefixKey)

		//  transfer the kv
		if success, err := f.node.etcd.Transfer(from, to, value); success {
			log.Infof("successfully transferred from %s to %s", from, to)
		} else {
			log.Errorf("transfer from %s to %s failed: %v", from, to, err)
		}
		return nil
	}
}

func (f *JobSnapshotFailOver) handleJobClientDeleteEventLite(event *JobClientDeleteEvent) {

	var (
		keys   [][]byte
		values [][]byte
		err    error
	)

	prefixKey := fmt.Sprintf(JobClientSnapshotPath, event.Group.name, event.Client.name)
	handler := f.generateJobSnapshotDeleteHandler(event, prefixKey)

RETRY:
	keys, values, err = f.node.etcd.GetWithPrefixKey(prefixKey)
	if err != nil {
		log.Errorf("the fail client: %v for path: %s, error must retry", event.Client.name, prefixKey)
		time.Sleep(time.Second * 2)
		goto RETRY
	}

	if len(keys) == 0 || len(values) == 0 {
		log.Warnf("the fail client: %v for path: %s is empty", event.Client.name, prefixKey)
		return
	}

	for pos := 0; pos < len(keys); pos++ {
		if err = handler(keys[pos], values[pos]); err != nil {
			break
		}
	}
}
