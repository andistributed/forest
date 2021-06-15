package forest

import (
	"fmt"

	"github.com/admpub/log"
)

const (
	JobSnapshotPath       = "/forest/client/snapshot/"
	JobSnapshotGroupPath  = "/forest/client/snapshot/%s/"    // %s:client.group
	JobClientSnapshotPath = "/forest/client/snapshot/%s/%s/" // %s:client.group %s:client.ip
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
		err := exec.handleJobSnapshot(snapshot)
		if err != nil {
			log.Error(err)
		}
	}
}

// handle the job snapshot
func (exec *JobExecutor) handleJobSnapshot(snapshot *JobSnapshot) error {
	var (
		err    error
		client *Client
	)
	group := snapshot.Group
	if client, err = exec.node.groupManager.selectClient(group); err != nil {
		return fmt.Errorf("the group: %s, select a client error: %w", group, err)
	}

	clientName := client.name
	snapshot.Ip = clientName

	log.Debugf("clientName: %v", clientName)
	snapshotPath := fmt.Sprintf(JobClientSnapshotPath, group, clientName)

	log.Debugf("snapshotPath: %v", snapshotPath)
	value, err := PackJobSnapshot(snapshot)
	if err != nil {
		return fmt.Errorf("pack the snapshot %s error: %w", group, err)
	}
	if err = exec.node.etcd.Put(snapshotPath+snapshot.Id, string(value)); err != nil {
		return fmt.Errorf("put the snapshot %s error: %w", group, err)
	}
	return nil
}

// push a new job snapshot
func (exec *JobExecutor) pushSnapshot(snapshot *JobSnapshot) {
	exec.snapshots <- snapshot
}
