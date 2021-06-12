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
func (f *JobSnapshotFailOver) handleJobClientDeleteEvent(event *JobClientDeleteEvent) error {

	var (
		keys    [][]byte
		values  [][]byte
		err     error
		client  *Client
		success bool
	)

RETRY:
	prefixKey := fmt.Sprintf(JobClientSnapshotPath, event.Group.name, event.Client.name)
	if keys, values, err = f.node.etcd.GetWithPrefixKeyLimit(prefixKey, 1000); err != nil {
		log.Errorf("the fail client: %v for path: %s, error must retry", event.Client.name, prefixKey)
		time.Sleep(time.Second * 2)
		goto RETRY
	}

	if len(keys) == 0 || len(values) == 0 {
		log.Warnf("the fail client: %v for path: %s is empty", event.Client.name, prefixKey)
		return err
	}

	for pos := 0; pos < len(keys); pos++ {
		if client, err = event.Group.selectClient(); err != nil {
			log.Error(err)
			return err
		}
		from := string(keys[pos])
		value := string(values[pos])

		// 新地址
		to := fmt.Sprintf(JobClientSnapshotPath, event.Group.name, client.name) + strings.TrimPrefix(from, prefixKey)

		//  transfer the kv
		success, err = f.node.etcd.Transfer(from, to, value)
		if !success {
			err = fmt.Errorf("transfer from %s to %s failed: %v", from, to, err)
			log.Error(err)
			return err
		}
		log.Infof("successfully transferred from %s to %s", from, to)
	}
	goto RETRY
}
