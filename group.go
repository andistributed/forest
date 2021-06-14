package forest

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/admpub/log"
	"github.com/andistributed/etcd/etcdevent"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	GroupConfPath = "/forest/server/group/"
	ClientPath    = "/forest/client/%s/clients/"
)

type JobGroupManager struct {
	node   *JobNode
	groups map[string]*Group
	lk     *sync.RWMutex
}

func NewJobGroupManager(node *JobNode) (mgr *JobGroupManager) {
	mgr = &JobGroupManager{
		node:   node,
		groups: make(map[string]*Group),
		lk:     &sync.RWMutex{},
	}
	go mgr.watchGroupPath()
	go mgr.loopLoadGroups()
	return
}

// watch the group path
func (mgr *JobGroupManager) watchGroupPath() {
	keyChangeEventResponse := mgr.node.etcd.WatchWithPrefixKey(GroupConfPath)
	for ch := range keyChangeEventResponse.Event {
		mgr.handleGroupChangeEvent(ch)
	}
}

func (mgr *JobGroupManager) loopLoadGroups() {

RETRY:
	var (
		keys   [][]byte
		values [][]byte
		err    error
	)
	if keys, values, err = mgr.node.etcd.GetWithPrefixKey(GroupConfPath); err != nil {
		goto RETRY
	}

	if len(keys) == 0 {
		return
	}

	for index, key := range keys {
		path := string(key)
		groupConf, err := UnpackGroupConf(values[index])
		if err != nil {
			log.Warnf("unpack the group conf error: %#v", err)
			continue
		}
		mgr.addGroup(groupConf.Name, path)
	}
}

func (mgr *JobGroupManager) addGroup(name, path string) {
	mgr.lk.Lock()
	defer mgr.lk.Unlock()
	if _, ok := mgr.groups[path]; ok {
		return
	}
	group := NewGroup(name, path, mgr.node)
	mgr.groups[path] = group
	log.Infof("add a new group: %s, for path: %s", name, path)
}

// delete a group  for path
func (mgr *JobGroupManager) deleteGroup(path string) {
	var (
		group *Group
		ok    bool
	)
	mgr.lk.Lock()
	defer mgr.lk.Unlock()
	if group, ok = mgr.groups[path]; !ok {
		return
	}
	// cancel watch the clients
	group.watcher.Close()
	group.cancelFunc()
	delete(mgr.groups, path)
	log.Infof("delete a group: %s, for path: %s", group.name, path)
}

// handle the group change event
func (mgr *JobGroupManager) handleGroupChangeEvent(changeEvent *etcdevent.KeyChangeEvent) {
	switch changeEvent.Type {
	case etcdevent.KeyCreateChangeEvent:
		mgr.handleGroupCreateEvent(changeEvent)

	case etcdevent.KeyUpdateChangeEvent:
		// ignore

	case etcdevent.KeyDeleteChangeEvent:
		mgr.handleGroupDeleteEvent(changeEvent)
	}
}

func (mgr *JobGroupManager) handleGroupCreateEvent(changeEvent *etcdevent.KeyChangeEvent) {
	groupConf, err := UnpackGroupConf(changeEvent.Value)
	if err != nil {
		log.Warnf("unpack the group conf error: %#v", err)
		return
	}
	path := changeEvent.Key
	mgr.addGroup(groupConf.Name, path)
}

func (mgr *JobGroupManager) handleGroupDeleteEvent(changeEvent *etcdevent.KeyChangeEvent) {
	path := changeEvent.Key
	mgr.deleteGroup(path)
}

func (mgr *JobGroupManager) selectClient(name string) (client *Client, err error) {
	var (
		group *Group
		ok    bool
	)
	if group, ok = mgr.groups[GroupConfPath+name]; !ok {
		err = fmt.Errorf("the group: %s not found", name)
		return
	}
	return group.selectClient()
}

type Group struct {
	path       string
	name       string
	node       *JobNode
	watchPath  string
	clients    map[string]*Client
	watcher    clientv3.Watcher
	cancelFunc context.CancelFunc
	lk         *sync.RWMutex
}

// create a new group
func NewGroup(name, path string, node *JobNode) (group *Group) {
	group = &Group{
		name:      name,
		path:      path,
		node:      node,
		watchPath: fmt.Sprintf(ClientPath, name),
		clients:   make(map[string]*Client),
		lk:        &sync.RWMutex{},
	}
	go group.watchClientPath()
	go group.loopLoadAllClient()
	return
}

// watch the client path
func (group *Group) watchClientPath() {
	keyChangeEventResponse := group.node.etcd.WatchWithPrefixKey(group.watchPath)
	group.watcher = keyChangeEventResponse.Watcher
	group.cancelFunc = keyChangeEventResponse.CancelFunc
	for ch := range keyChangeEventResponse.Event {
		group.handleClientChangeEvent(ch)
	}
}

// loop load all client
func (group *Group) loopLoadAllClient() {

RETRY:
	var (
		keys   [][]byte
		values [][]byte
		err    error
	)

	prefix := fmt.Sprintf(ClientPath, group.name)
	if keys, values, err = group.node.etcd.GetWithPrefixKey(prefix); err != nil {
		time.Sleep(time.Second)
		goto RETRY
	}

	if len(keys) == 0 {
		return
	}

	for index, key := range keys {
		path := string(key)
		value := string(values[index])
		if value == "" {
			log.Warnf("the client value is nil for path: %s", path)
			continue
		}
		group.addClient(value, path)
	}
}

// handle the client change event
func (group *Group) handleClientChangeEvent(changeEvent *etcdevent.KeyChangeEvent) {
	switch changeEvent.Type {
	case etcdevent.KeyCreateChangeEvent:
		path := changeEvent.Key
		name := string(changeEvent.Value)
		group.addClient(name, path)

	case etcdevent.KeyUpdateChangeEvent:
		//ignore
	case etcdevent.KeyDeleteChangeEvent:
		path := changeEvent.Key
		group.deleteClient(path)
	}
}

// add  a new  client
func (group *Group) addClient(name, path string) {
	group.lk.Lock()
	defer group.lk.Unlock()

	if _, ok := group.clients[path]; ok {
		log.Warnf("name: %s, path: %s, the client exist", name, path)
		return
	}
	client := &Client{
		name: name,
		path: path,
	}
	group.clients[path] = client
	log.Infof("add a new client for path: %s", path)
}

// delete a client for path
func (group *Group) deleteClient(path string) {
	var (
		client *Client
		ok     bool
	)
	group.lk.Lock()
	defer group.lk.Unlock()
	if client, ok = group.clients[path]; !ok {
		log.Warnf("path: %s, the client not exist", path)
		return
	}
	delete(group.clients, path)
	log.Infof("delete a client for path: %s", path)
	// fail over
	if group.node.state == NodeLeaderState {
		group.node.failOver.deleteClientEventChans <- &JobClientDeleteEvent{Group: group, Client: client}
	}
}

func (group *Group) selectClient() (client *Client, err error) {
	group.lk.RLock()
	defer group.lk.RUnlock()

	if len(group.clients) == 0 {
		err = fmt.Errorf("the group: %s, has no client to select", group.name)
		return
	}

	num := len(group.clients)
	pos := rand.Intn(num)
	index := 0
	for _, c := range group.clients {
		if index == pos {
			client = c
			return
		}
		index++
	}
	return
}

// Client client
type Client struct {
	name string
	path string
}
