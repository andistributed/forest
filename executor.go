package forest

import (
	"fmt"

	"github.com/admpub/log"
)

const (
	JobSnapshotPath       = "/forest/client/snapshot/"
	JobSnapshotGroupPath  = "/forest/client/snapshot/%s/"
	JobClientSnapshotPath = "/forest/client/snapshot/%s/%s/"
)

type JobExecutor struct {
	node      *JobNode
	snapshots chan *JobSnapshot
}

func NewJobExecutor(node *JobNode) (exec *JobExecutor) {
	exec = &JobExecutor{
		node:      node,
		snapshots: make(chan *JobSnapshot, 500),
	}
	go exec.lookup()
	return
}

func (exec *JobExecutor) lookup() {
	for snapshot := range exec.snapshots {
		exec.handleJobSnapshot(snapshot)
	}
}

// handle the job snapshot
func (exec *JobExecutor) handleJobSnapshot(snapshot *JobSnapshot) {
	var (
		err    error
		client *Client
	)
	group := snapshot.Group
	if client, err = exec.node.groupManager.selectClient(group); err != nil {
		log.Warnf("the group: %s, select a client error: %#v", group, err)
		return
	}

	clientName := client.name
	snapshot.Ip = clientName

	log.Debugf("clientName: %v", clientName)
	snapshotPath := fmt.Sprintf(JobClientSnapshotPath, group, clientName)

	log.Debugf("snapshotPath: %v", snapshotPath)
	value, err := PackJobSnapshot(snapshot)
	if err != nil {
		log.Warnf("Pack the snapshot %s error: %#v", group, err)
		return
	}
	if err = exec.node.etcd.Put(snapshotPath+snapshot.Id, string(value)); err != nil {
		log.Warnf("put the snapshot %s error: %#v", group, err)
	}
}

// push a new job snapshot
func (exec *JobExecutor) pushSnapshot(snapshot *JobSnapshot) {
	exec.snapshots <- snapshot
}
