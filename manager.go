package forest

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/admpub/log"
	"github.com/andistributed/etcd/etcdevent"
)

const (
	JobConfPath = "/forest/server/conf/"
)

type JobManager struct {
	node *JobNode
}

func NewJobManager(node *JobNode) (manager *JobManager) {
	manager = &JobManager{
		node: node,
	}
	go manager.watchJobConfPath()
	return
}

func (manager *JobManager) watchJobConfPath() {
	keyChangeEventResponse := manager.node.etcd.WatchWithPrefixKey(JobConfPath)
	for ch := range keyChangeEventResponse.Event {
		manager.handleJobConfChangeEvent(ch)
	}
}

func (manager *JobManager) loopLoadJobConf() {

RETRY:
	var (
		keys   [][]byte
		values [][]byte
		err    error
	)
	if keys, values, err = manager.node.etcd.GetWithPrefixKey(JobConfPath); err != nil {
		goto RETRY
	}

	if len(keys) == 0 {
		return
	}

	for i := 0; i < len(keys); i++ {
		jobConf, err := UnpackJobConf(values[i])
		if err != nil {
			log.Warnf("unpack the job conf error: %#v", err)
			continue
		}
		manager.node.scheduler.pushJobChangeEvent(&JobChangeEvent{
			Type: JobCreateChangeEvent,
			Conf: jobConf,
		})
	}
}

func (manager *JobManager) handleJobConfChangeEvent(changeEvent *etcdevent.KeyChangeEvent) {

	switch changeEvent.Type {
	case etcdevent.KeyCreateChangeEvent:
		manager.handleJobCreateEvent(changeEvent.Value)

	case etcdevent.KeyUpdateChangeEvent:
		manager.handleJobUpdateEvent(changeEvent.Value)

	case etcdevent.KeyDeleteChangeEvent:
		manager.handleJobDeleteEvent(changeEvent.Key)
	}
}

func (manager *JobManager) handleJobCreateEvent(value []byte) {
	var (
		err     error
		jobConf *JobConf
	)
	if len(value) == 0 {
		return
	}
	if jobConf, err = UnpackJobConf(value); err != nil {
		log.Errorf("unpack the job conf err: %#v", err)
		return
	}
	manager.node.scheduler.pushJobChangeEvent(&JobChangeEvent{
		Type: JobCreateChangeEvent,
		Conf: jobConf,
	})
}

func (manager *JobManager) handleJobUpdateEvent(value []byte) {

	var (
		err     error
		jobConf *JobConf
	)
	if len(value) == 0 {
		return
	}

	if jobConf, err = UnpackJobConf(value); err != nil {
		log.Errorf("unpack the job conf err: %#v", err)
		return
	}

	manager.node.scheduler.pushJobChangeEvent(&JobChangeEvent{
		Type: JobUpdateChangeEvent,
		Conf: jobConf,
	})

}

// handle the job delete event
func (manager *JobManager) handleJobDeleteEvent(key string) {
	if key == "" {
		return
	}
	pos := strings.LastIndex(key, "/")
	if pos == -1 {
		return
	}
	id := key[pos+1:]
	jobConf := &JobConf{
		Id:      id,
		Version: -1,
	}
	manager.node.scheduler.pushJobChangeEvent(&JobChangeEvent{
		Type: JobDeleteChangeEvent,
		Conf: jobConf,
	})
}

// AddJob add job conf
func (manager *JobManager) AddJob(jobConf *JobConf) (err error) {
	var (
		value   []byte
		v       []byte
		success bool
	)
	if value, err = manager.node.etcd.Get(GroupConfPath + jobConf.Group); err != nil {
		return
	}

	if len(value) == 0 {
		err = errors.New("任务集群不存在")
		return
	}

	jobConf.Id = GenerateSerialNo()
	jobConf.Version = 1

	if v, err = PackJobConf(jobConf); err != nil {
		return
	}
	if success, _, err = manager.node.etcd.PutNotExist(JobConfPath+jobConf.Id, string(v)); err != nil {
		return
	}
	if !success {
		err = errors.New("创建失败,请重试！")
		return
	}
	return
}

// edit job conf
func (manager *JobManager) EditJob(jobConf *JobConf) (err error) {
	var (
		value   []byte
		v       []byte
		success bool
		oldConf *JobConf
	)

	if value, err = manager.node.etcd.Get(GroupConfPath + jobConf.Group); err != nil {
		return
	}

	if len(value) == 0 {
		err = errors.New("任务集群不存在")
		return
	}

	if jobConf.Id == "" {
		err = errors.New("此记录任务配置记录不存在")
		return
	}

	if value, err = manager.node.etcd.Get(JobConfPath + jobConf.Id); err != nil {
		return
	}

	if len(value) == 0 {
		err = errors.New("此任务配置记录不存在")
		return
	}

	if oldConf, err = UnpackJobConf([]byte(value)); err != nil {
		return
	}

	jobConf.Version = oldConf.Version + 1
	if v, err = PackJobConf(jobConf); err != nil {
		return
	}

	if success, err = manager.node.etcd.Update(JobConfPath+jobConf.Id, string(v), string(value)); err != nil {
		return
	}
	if !success {
		err = errors.New("修改失败,请重试！")
		return
	}
	return
}

// delete job conf
func (manager *JobManager) DeleteJob(jobConf *JobConf) (err error) {
	var (
		value []byte
	)

	if jobConf.Id == "" {
		err = errors.New("此记录任务配置记录不存在")
		return
	}

	if value, err = manager.node.etcd.Get(JobConfPath + jobConf.Id); err != nil {
		return
	}

	if len(value) == 0 {
		err = errors.New("此任务配置记录不存在")
		return
	}
	err = manager.node.etcd.Delete(JobConfPath + jobConf.Id)
	return
}

// job list
func (manager *JobManager) JobList() (jobConfs []*JobConf, err error) {
	var (
		keys   [][]byte
		values [][]byte
	)
	if keys, values, err = manager.node.etcd.GetWithPrefixKey(JobConfPath); err != nil {
		return
	}

	if len(keys) == 0 {
		return
	}

	jobConfs = make([]*JobConf, 0)
	for i := 0; i < len(values); i++ {
		jobConf, err := UnpackJobConf(values[i])
		if err != nil {
			log.Errorf("unpack the job conf errror: %#v", err)
			continue
		}
		jobConfs = append(jobConfs, jobConf)
	}
	return
}

// add group
func (manager *JobManager) AddGroup(groupConf *GroupConf) (err error) {
	var (
		value   []byte
		success bool
	)
	if value, err = PackGroupConf(groupConf); err != nil {
		return
	}

	if success, _, err = manager.node.etcd.PutNotExist(GroupConfPath+groupConf.Name, string(value)); err != nil {
		return
	}

	if !success {
		err = errors.New("此任务集群已存在")
	}
	return
}

// edit group
func (manager *JobManager) EditGroup(groupConf *GroupConf) (err error) {
	var (
		value   []byte
		newV    []byte
		success bool
	)
	if value, err = manager.node.etcd.Get(GroupConfPath + groupConf.Name); err != nil {
		return
	}

	if len(value) == 0 {
		err = errors.New("此任务集群不存在")
		return
	}

	if newV, err = PackGroupConf(groupConf); err != nil {
		return
	}

	if success, err = manager.node.etcd.Update(GroupConfPath+groupConf.Name, string(newV), string(value)); err != nil {
		return
	}

	if !success {
		err = errors.New("此任务集群已存在")
	}
	return
}

// delete group
func (manager *JobManager) DeleteGroup(groupConf *GroupConf) (err error) {

	var (
		value []byte
	)

	if groupConf.Name == "" {
		err = errors.New("此任务集群不存在")
		return
	}

	if value, err = manager.node.etcd.Get(GroupConfPath + groupConf.Name); err != nil {
		return
	}

	if len(value) == 0 {
		err = errors.New("此任务集群不存在")
		return
	}

	err = manager.node.etcd.Delete(GroupConfPath + groupConf.Name)
	return
}

// group list
func (manager *JobManager) GroupList() (groupConfs []*GroupConf, err error) {
	var (
		keys   [][]byte
		values [][]byte
	)
	if keys, values, err = manager.node.etcd.GetWithPrefixKey(GroupConfPath); err != nil {
		return
	}

	if len(keys) == 0 {
		return
	}

	groupConfs = make([]*GroupConf, 0)
	for i := 0; i < len(values); i++ {
		groupConf, err := UnpackGroupConf(values[i])
		if err != nil {
			log.Errorf("unpack the group conf errror: %#v", err)
			continue
		}

		groupConfs = append(groupConfs, groupConf)

	}
	return
}

// node list
func (manager *JobManager) NodeList() (nodes []string, err error) {
	var (
		keys   [][]byte
		values [][]byte
	)
	if keys, values, err = manager.node.etcd.GetWithPrefixKey(JobNodePath); err != nil {
		return
	}

	if len(keys) == 0 {
		return
	}

	nodes = make([]string, 0)
	for i := 0; i < len(values); i++ {
		nodes = append(nodes, string(values[i]))
	}
	return
}

func (manager *JobManager) ManualExecuteJob(jobId string) error {
	// 查询任务配置
	value, err := manager.node.etcd.Get(JobConfPath + jobId)
	if err != nil {
		return fmt.Errorf("查询任务配置出现异常: %w", err)
	}

	// 任务配置是否为空
	if len(value) == 0 {
		return errors.New("此任务配置内容为空")
	}

	var conf *JobConf
	conf, err = UnpackJobConf(value)
	if err != nil {
		return fmt.Errorf("非法的任务配置内容: %w", err)
	}

	// build job snapshot
	snapshotId := GenerateSerialNo() + conf.Id
	snapshot := &JobSnapshot{
		Id:         snapshotId,
		JobId:      conf.Id,
		Name:       conf.Name,
		Group:      conf.Group,
		Cron:       conf.Cron,
		Target:     conf.Target,
		Params:     conf.Params,
		Mobile:     conf.Mobile,
		Remark:     conf.Remark,
		CreateTime: ToDateString(time.Now()),
	}
	return manager.ManualExecute(snapshot)
}

func (manager *JobManager) ManualExecute(snapshot *JobSnapshot) error {
	var (
		client  *Client
		success bool
		value   []byte
	)

	// select a execute  job client for group
	client, err := manager.node.groupManager.selectClient(snapshot.Group)
	if err != nil {
		return fmt.Errorf("没有找到可以执行此任务的作业节点: %w", err)
	}

	// build the job snapshot path
	snapshotPath := fmt.Sprintf(JobClientSnapshotPath, snapshot.Group, client.name)

	// build job snapshot
	if len(snapshot.Id) == 0 {
		snapshot.Id = GenerateSerialNo()
	}
	snapshot.Ip = client.name

	// pack the job snapshot
	if value, err = PackJobSnapshot(snapshot); err != nil {
		return err
	}

	// dispatch the job snapshot the client
	success, _, err = manager.node.etcd.PutNotExist(snapshotPath, string(value))
	if err != nil {
		return err
	}
	if !success {
		return errors.New("手动执行任务失败, 请重试")
	}
	return err
}
