package forest

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/admpub/log"
	"github.com/andistributed/etcd"
	"github.com/andistributed/etcd/etcdevent"
	"github.com/andistributed/etcd/etcdresponse"
	"github.com/webx-top/db"
	"github.com/webx-top/db/lib/sqlbuilder"
	"github.com/webx-top/db/mysql"
	"github.com/webx-top/echo/engine"
)

const (
	JobNodePath      = "/forest/server/node/"
	JobNodeElectPath = "/forest/server/elect/leader"
	TTL              = 5
)

// JobNode job node
type JobNode struct {
	id           string
	registerPath string
	electPath    string
	etcd         *etcd.Etcd
	state        int
	manager      *JobManager
	scheduler    *JobScheduler
	groupManager *JobGroupManager
	exec         *JobExecutor
	collection   *JobCollection
	failOver     *JobSnapshotFailOver
	listeners    []NodeStateChangeListener
	close        chan bool

	// - db -
	db         sqlbuilder.Database
	dbSettings mysql.ConnectionURL
	once       sync.Once
}

// NodeStateChangeListener node state change listener
type NodeStateChangeListener interface {
	notify(int)
}

func NewJobNode(id string, etcd *etcd.Etcd, dsn string) (node *JobNode, err error) {
	node = &JobNode{
		id:           id,
		registerPath: JobNodePath + id,
		electPath:    JobNodeElectPath,
		etcd:         etcd,
		state:        NodeFollowerState,
		close:        make(chan bool),
		listeners:    []NodeStateChangeListener{},
		once:         sync.Once{},
	}
	if len(dsn) == 0 {
		dsn = os.Getenv(`FOREST_DSN`)
	}
	if len(dsn) > 0 {
		if err := node.SetDSN(dsn); err != nil {
			log.Error(err)
		}
	}
	node.failOver = NewJobSnapshotFailOver(node)
	node.collection = NewJobCollection(node)
	node.initNode()

	// create job executor
	node.exec = NewJobExecutor(node)

	// create  group manager
	node.groupManager = NewJobGroupManager(node)

	node.scheduler = NewJobScheduler(node)

	// create job manager
	node.manager = NewJobManager(node)

	node.addListeners()
	return
}

func (node *JobNode) Manager() *JobManager {
	return node.manager
}

func (node *JobNode) ETCD() *etcd.Etcd {
	return node.etcd
}

func (node *JobNode) SetDBConfig(conf mysql.ConnectionURL) (err error) {
	node.dbSettings = conf
	err = node.initConnectDB()
	return
}

func (node *JobNode) SetDSN(dsn string) (err error) {
	node.dbSettings, err = mysql.ParseURL(dsn)
	if err == nil {
		err = node.initConnectDB()
	}
	return
}

func (node *JobNode) initConnectDB() (err error) {
	if node.db != nil {
		node.db.Close()
	}
	node.db, err = mysql.Open(node.dbSettings)
	if err != nil {
		node.db, err = node.autoCreateDatabase(err)
	}
	return
}

func (node *JobNode) DB() sqlbuilder.Database {
	node.once.Do(node.connectDB)
	return node.db
}

func (node *JobNode) UseTable(table string) db.Collection {
	return node.DB().Collection(table)
}

func (node *JobNode) connectDB() {
	if node.db != nil {
		return
	}
	db, err := mysql.Open(node.dbSettings)
	if err != nil {
		log.Fatal(err)
	}
	node.db = db
}

func (node *JobNode) autoCreateDatabase(err error) (sqlbuilder.Database, error) {
	if !strings.Contains(err.Error(), `Unknown database`) {
		return nil, err
	}
	settings := node.dbSettings
	settings.Database = ``
	db, err := mysql.Open(settings)
	if err != nil {
		return nil, err
	}
	log.Info(`create database: `, node.dbSettings.Database)
	_, err = db.Exec("CREATE DATABASE `" + node.dbSettings.Database + "`")
	db.Close()
	if err != nil {
		return nil, err
	}
	db, err = mysql.Open(node.dbSettings)
	if err != nil {
		return nil, err
	}
	for _, sql := range InitSQLs {
		log.Info(`execution sql: `, sql)
		_, err = db.Exec(sql)
		if err != nil {
			db.Close()
			return nil, err
		}
	}
	return db, err
}

// StartAPIServer create a job http api and start service
func (node *JobNode) StartAPIServer(auth *APIAuth, address string, opts ...engine.ConfigSetter) {
	api := NewJobAPI(node, auth)
	api.Start(address, opts...)
}

func (node *JobNode) addListeners() {
	node.listeners = append(node.listeners, node.scheduler)
}

func (node *JobNode) changeState(state int) {
	node.state = state
	if len(node.listeners) == 0 {
		return
	}

	// notify all listener
	for _, listener := range node.listeners {
		listener.notify(state)
	}
}

// start register node
func (node *JobNode) initNode() {
	var (
		txResponse *etcdresponse.TxResponse
		err        error
	)
	for i := 1; i <= 10; i++ {
		txResponse, err = node.registerJobNode()
		if err != nil {
			err = fmt.Errorf("the job node: %s, fail register to: %s error: %w", node.id, node.registerPath, err)
			time.Sleep(time.Second * time.Duration(i))
			continue
		}
		if !txResponse.Success {
			err = fmt.Errorf("the job node: %s, fail register to: %s, the job node id exist", node.id, node.registerPath)
			time.Sleep(time.Second * time.Duration(i))
			continue
		}
		break
	}
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("the job node: %s, success register to: %s", node.id, node.registerPath)
	node.watchRegisterJobNode()
	node.watchElectPath()
	go node.loopStartElect()
}

// Bootstrap bootstrap
func (node *JobNode) Bootstrap() {
	go node.groupManager.loopLoadGroups()
	go node.manager.loopLoadJobConf()
	<-node.close
}

func (node *JobNode) Close() {
	node.close <- true
}

// watch the register job node
func (node *JobNode) watchRegisterJobNode() {
	keyChangeEventResponse := node.etcd.Watch(node.registerPath)
	go func() {
		for ch := range keyChangeEventResponse.Event {
			node.handleRegisterJobNodeChangeEvent(ch)
		}
	}()
}

// handle the register job node change event
func (node *JobNode) handleRegisterJobNodeChangeEvent(changeEvent *etcdevent.KeyChangeEvent) {
	switch changeEvent.Type {
	case etcdevent.KeyCreateChangeEvent:
	case etcdevent.KeyUpdateChangeEvent:
	case etcdevent.KeyDeleteChangeEvent:
		log.Infof("found the job node: %s register to path: %s has lose", node.id, node.registerPath)
		go node.loopRegisterJobNode()
	}
}

func (node *JobNode) registerJobNode() (txResponse *etcdresponse.TxResponse, err error) {
	return node.etcd.TxKeepaliveWithTTL(node.registerPath, node.id, TTL)
}

// loop register the job node
func (node *JobNode) loopRegisterJobNode() {

RETRY:
	var (
		txResponse *etcdresponse.TxResponse
		err        error
	)
	if txResponse, err = node.registerJobNode(); err != nil {
		log.Infof("the job node: %s, fail register to: %s", node.id, node.registerPath)
		time.Sleep(time.Second)
		goto RETRY
	}

	if txResponse.Success {
		log.Infof("the job node: %s, success register to: %s", node.id, node.registerPath)
	} else {
		v := txResponse.Value
		if v != node.id {
			time.Sleep(time.Second)
			log.Fatalf("the job node: %s, the other job node: %s has already register to: %s", node.id, v, node.registerPath)
		}
		log.Infof("the job node: %s, has already success register to: %s", node.id, node.registerPath)
	}
}

// elect the leader
func (node *JobNode) elect() (txResponse *etcdresponse.TxResponse, err error) {
	return node.etcd.TxKeepaliveWithTTL(node.electPath, node.id, TTL)
}

// watch the job node elect path
func (node *JobNode) watchElectPath() {
	keyChangeEventResponse := node.etcd.Watch(node.electPath)
	go func() {
		for ch := range keyChangeEventResponse.Event {
			node.handleElectLeaderChangeEvent(ch)
		}
	}()
}

// handle the job node leader change event
func (node *JobNode) handleElectLeaderChangeEvent(changeEvent *etcdevent.KeyChangeEvent) {
	switch changeEvent.Type {
	case etcdevent.KeyDeleteChangeEvent:
		node.changeState(NodeFollowerState)
		node.loopStartElect()

	case etcdevent.KeyCreateChangeEvent:

	case etcdevent.KeyUpdateChangeEvent:

	}
}

// loop start elect
func (node *JobNode) loopStartElect() {

RETRY:
	var (
		txResponse *etcdresponse.TxResponse
		err        error
	)
	if txResponse, err = node.elect(); err != nil {
		log.Infof("the job node: %s, elect fail to: %s", node.id, node.electPath)
		time.Sleep(time.Second)
		goto RETRY
	}

	if txResponse.Success {
		node.changeState(NodeLeaderState)
		log.Infof("the job node: %s, elect success to: %s", node.id, node.electPath)
	} else {
		v := txResponse.Value
		if v != node.id {
			log.Infof("the job node: %s, give up elect request because the other job node: %s elect to: %s", node.id, v, node.electPath)
			node.changeState(NodeFollowerState)
		} else {
			log.Infof("the job node: %s, has already elect  success to: %s", node.id, node.electPath)
			node.changeState(NodeLeaderState)
		}
	}
}
