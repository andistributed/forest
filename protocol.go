package forest

import (
	"fmt"
	"time"

	"github.com/robfig/cron"
)

const (
	JobCreateChangeEvent = iota
	JobUpdateChangeEvent
	JobDeleteChangeEvent
)

const (
	JobRunningStatus = iota + 1
	JobStopStatus
)

const (
	NodeFollowerState = iota
	NodeLeaderState
)

const (
	JobExecuteSnapshotDoingStatus   = 1
	JobExecuteSnapshotSuccessStatus = 2
	JobExecuteSnapshotUnknownStatus = 3
	JobExecuteSnapshotErrorStatus   = -1
)

type JobClientDeleteEvent struct {
	Client *Client
	Group  *Group
}

// JobConf job
type JobConf struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Group   string `json:"group"`
	Cron    string `json:"cron"`
	Status  int    `json:"status"`
	Target  string `json:"target"`
	Params  string `json:"params"`
	Mobile  string `json:"mobile"`
	Remark  string `json:"remark"`
	Version int    `json:"version"`
}

type Result struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

type GroupConf struct {
	Name   string `json:"name"`
	Remark string `json:"remark"`
}

type JobChangeEvent struct {
	Type int
	Conf *JobConf
}

type SchedulePlan struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	Group      string `json:"group"`
	Cron       string `json:"cron"`
	Status     int    `json:"status"`
	Target     string `json:"target"`
	Params     string `json:"params"`
	Mobile     string `json:"mobile"`
	Remark     string `json:"remark"`
	schedule   cron.Schedule
	NextTime   time.Time `json:"nextTime"`
	BeforeTime time.Time `json:"beforeTime"`
	Version    int       `json:"version"`
}

type JobSnapshotWithPath struct {
	*JobSnapshot
	Path string
}

type JobSnapshot struct {
	Id         string `json:"id"`
	JobId      string `json:"jobId"`
	Name       string `json:"name"`
	Ip         string `json:"ip"`
	Group      string `json:"group"`
	Cron       string `json:"cron"`
	Target     string `json:"target"`
	Params     string `json:"params"`
	Remark     string `json:"remark"`
	CreateTime string `json:"createTime"`
}

func (s *JobSnapshot) Path() string {
	return fmt.Sprintf(JobClientSnapshotPath, s.Group, s.Ip)
}

type QueryClientParam struct {
	Group string `json:"group"`
}

type JobClient struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Group string `json:"group"`
}
type QuerySnapshotParam struct {
	Group string `json:"group"`
	Id    string `json:"id"`
	Ip    string `json:"ip"`
}

// Node node
type Node struct {
	Name  string `json:"name"`
	State int    `json:"state"`
}

type JobExecuteSnapshot struct {
	Id         string `json:"id" db:"id"`
	JobId      string `json:"jobId" db:"job_id"`
	Name       string `json:"name" db:"name"`
	Ip         string `json:"ip" db:"ip"`
	Group      string `json:"group" db:"group"`
	Cron       string `json:"cron" db:"cron"`
	Target     string `json:"target" db:"target"`
	Params     string `json:"params" db:"params"`
	Remark     string `json:"remark" db:"remark"`
	CreateTime string `json:"createTime" db:"create_time"`
	StartTime  string `json:"startTime" db:"start_time"`
	FinishTime string `json:"finishTime" db:"finish_time"`
	Times      int    `json:"times" db:"times"`
	Status     int    `json:"status" db:"status"`
	Result     string `json:"result" db:"result"`
}

func (s *JobExecuteSnapshot) Path() string {
	return JobExecuteStatusCollectionPath + `/` + s.Group + `/` + s.Ip + `/`
}

type QueryExecuteSnapshotParam struct {
	Group    string `json:"group"`
	Id       string `json:"id"`
	Ip       string `json:"ip"`
	JobId    string `json:"jobId"`
	Name     string `json:"name"`
	Status   int    `json:"status"`
	PageSize int    `json:"pageSize"`
	PageNo   int    `json:"pageNo"`
}

type PageResult struct {
	TotalPage  int         `json:"totalPage"`
	TotalCount int         `json:"totalCount"`
	List       interface{} `json:"list"`
}

// ManualExecuteJobParam manual execute job
type ManualExecuteJobParam struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Group string `json:"group"`
}

type InputLogin struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
