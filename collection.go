package forest

import (
	"sync"
	"time"

	"github.com/admpub/log"
	"github.com/andistributed/etcd/etcdevent"
	"github.com/webx-top/db"
)

// collection job execute status

const (
	JobExecuteStatusCollectionPath = "/forest/client/execute/snapshot/"
)

type JobCollection struct {
	node *JobNode
	lk   *sync.RWMutex
}

func NewJobCollection(node *JobNode) (c *JobCollection) {

	c = &JobCollection{
		node: node,
		lk:   &sync.RWMutex{},
	}

	c.watch()
	go c.loop()
	return
}

// watch
func (c *JobCollection) watch() {
	keyChangeEventResponse := c.node.etcd.WatchWithPrefixKey(JobExecuteStatusCollectionPath)
	log.Infof("the job collection success watch for path: %s", JobExecuteStatusCollectionPath)
	go func() {
		for event := range keyChangeEventResponse.Event {
			c.handleJobExecuteStatusCollectionEvent(event)
		}
	}()
}

// handle the job execute status
func (c *JobCollection) handleJobExecuteStatusCollectionEvent(event *etcdevent.KeyChangeEvent) {

	if c.node.state == NodeFollowerState {
		return
	}

	switch event.Type {

	case etcdevent.KeyCreateChangeEvent:

		if len(event.Value) == 0 {
			return
		}

		executeSnapshot, err := UnpackJobExecuteSnapshot(event.Value)

		if err != nil {
			log.Warnf("UnpackJobExecuteSnapshot: %s fail, err: %#v", event.Value, err)
			_ = c.node.etcd.Delete(event.Key)
			return
		}
		c.handleJobExecuteSnapshot(event.Key, executeSnapshot)

	case etcdevent.KeyUpdateChangeEvent:

		if len(event.Value) == 0 {
			return
		}

		executeSnapshot, err := UnpackJobExecuteSnapshot(event.Value)

		if err != nil {
			log.Warnf("UnpackJobExecuteSnapshot: %s fail, err: %#v", event.Value, err)
			return
		}

		c.handleJobExecuteSnapshot(event.Key, executeSnapshot)

	case etcdevent.KeyDeleteChangeEvent:

	}
}

// handle job execute snapshot
func (c *JobCollection) handleJobExecuteSnapshot(path string, snapshot *JobExecuteSnapshot) {

	var (
		exist bool
		err   error
	)

	c.lk.Lock()
	defer c.lk.Unlock()
	if exist, err = c.checkExist(snapshot.Id); err != nil {
		log.Errorf("check snapshot exist error: %v", err)
		return
	}

	if exist {
		c.handleUpdateJobExecuteSnapshot(path, snapshot)
	} else {
		c.handleCreateJobExecuteSnapshot(path, snapshot)
	}
}

// handle create job execute snapshot
func (c *JobCollection) handleCreateJobExecuteSnapshot(path string, snapshot *JobExecuteSnapshot) {

	if snapshot.Status == JobExecuteSnapshotUnknownStatus ||
		snapshot.Status == JobExecuteSnapshotErrorStatus ||
		snapshot.Status == JobExecuteSnapshotSuccessStatus {
		err := c.node.etcd.Delete(path)
		if err != nil {
			log.Error(err)
		}
	}

	var days int
	dateTime, err := ParseInLocation(snapshot.CreateTime)
	if err == nil {
		days = TimeSubDays(time.Now(), dateTime)
	}
	if snapshot.Status == JobExecuteSnapshotDoingStatus && days >= 3 {
		err := c.node.etcd.Delete(path)
		if err != nil {
			log.Error(err)
		}
	}
	_, err = c.node.UseTable(TableJobExecuteSnapshot).Insert(snapshot)
	if err != nil {
		log.Errorf("err: %#v", err)
	}
}

// handle update job execute snapshot
func (c *JobCollection) handleUpdateJobExecuteSnapshot(path string, snapshot *JobExecuteSnapshot) {

	if snapshot.Status == JobExecuteSnapshotUnknownStatus ||
		snapshot.Status == JobExecuteSnapshotErrorStatus ||
		snapshot.Status == JobExecuteSnapshotSuccessStatus {
		err := c.node.etcd.Delete(path)
		if err != nil {
			log.Error(err)
		}
	}

	var days int
	dateTime, err := ParseInLocation(snapshot.CreateTime)
	if err == nil {
		days = TimeSubDays(time.Now(), dateTime)
	}
	if snapshot.Status == JobExecuteSnapshotDoingStatus && days >= 3 {
		err := c.node.etcd.Delete(path)
		if err != nil {
			log.Error(err)
		}
	}

	err = c.node.UseTable(TableJobExecuteSnapshot).
		Find(db.Cond{`id`: snapshot.Id}).Update(snapshot)
	if err != nil {
		log.Error(err)
	}
}

// check the exist
func (c *JobCollection) checkExist(id string) (exist bool, err error) {
	exist, err = c.node.UseTable(TableJobExecuteSnapshot).Find(db.Cond{`id`: id}).Exists()
	if err != nil {
		if err == db.ErrNoMoreRows {
			err = nil
		}
		return
	}
	return
}

func (c *JobCollection) loop() {

	timer := time.NewTimer(10 * time.Minute)
	defer timer.Stop()
	for {
		key := JobExecuteStatusCollectionPath
		select {
		case <-timer.C:

			timer.Reset(10 * time.Second)
			keys, values, err := c.node.etcd.GetWithPrefixKeyLimit(key, 1000)
			if err != nil {
				log.Warnf("collection loop err: %v", err)
				continue
			}

			if len(keys) == 0 {
				continue
			}

			for index, key := range keys {
				executeSnapshot, err := UnpackJobExecuteSnapshot(values[index])
				if err != nil {
					log.Warnf("UnpackJobExecuteSnapshot: %s fail, err: %#v", values[index], err)
					err := c.node.etcd.Delete(string(key))
					if err != nil {
						log.Error(err)
					}
					continue
				}

				if executeSnapshot.Status == JobExecuteSnapshotSuccessStatus ||
					executeSnapshot.Status == JobExecuteSnapshotErrorStatus ||
					executeSnapshot.Status == JobExecuteSnapshotUnknownStatus {
					path := string(key)
					c.handleJobExecuteSnapshot(path, executeSnapshot)
				}
			}
		}
	}
}
