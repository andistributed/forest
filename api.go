package forest

import (
	"errors"
	"fmt"
	"time"

	"github.com/admpub/log"
	"github.com/dgrijalva/jwt-go"
	"github.com/robfig/cron"
	"github.com/webx-top/db"
	"github.com/webx-top/echo"
	"github.com/webx-top/echo/engine"
	"github.com/webx-top/echo/engine/standard"
	"github.com/webx-top/echo/middleware"
	mwjwt "github.com/webx-top/echo/middleware/jwt"
	"github.com/webx-top/echo/middleware/session"
)

type ApiAuth struct {
	Auth   func(*InputLogin) error
	JWTKey string
}

type JobAPI struct {
	node *JobNode
	echo *echo.Echo
	auth *ApiAuth
}

func NewJobAPi(node *JobNode, auth *ApiAuth) (api *JobAPI) {
	api = &JobAPI{
		node: node,
		auth: auth,
	}
	e := echo.New()
	e.Use(middleware.Recover(), middleware.Log())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAccessControlAllowOrigin, echo.HeaderAuthorization},
		AllowMethods: []string{echo.GET, echo.HEAD, echo.PUT, echo.POST, echo.DELETE},
	}))
	mwjwt.DefaultJWTConfig.Skipper = func(c echo.Context) bool {
		switch c.Path() {
		case `/login`, `/logout`:
			return true
		default:
			return false
		}
	}
	e.SetHTTPErrorHandler(func(err error, c echo.Context) {
		r := Result{Code: -1, Message: err.Error()}
		if rawErr, ok := err.(*echo.HTTPError); ok && rawErr.Raw() != nil {
			err = rawErr.Raw()
		}
		if errors.Is(err, mwjwt.ErrJWTMissing) || errors.Is(err, echo.ErrUnauthorized) {
			r.Code = -2
			r.Message = `请重新登录`
		} else {
			log.Debugf(err.Error())
		}
		c.JSON(r)
	})
	e.Use(mwjwt.JWT([]byte(auth.JWTKey)))
	e.Use(session.Middleware(nil))
	e.Post("/login", api.login)
	e.Post("/logout", api.logout)
	e.Post("/job/add", api.addJob)
	e.Post("/job/edit", api.editJob)
	e.Post("/job/delete", api.deleteJob)
	e.Post("/job/list", api.jobList)
	e.Post("/job/execute", api.manualExecute)
	e.Post("/group/add", api.addGroup)
	e.Post("/group/edit", api.editGroup)
	e.Post("/group/delete", api.deleteGroup)
	e.Post("/group/list", api.groupList)
	e.Post("/node/list", api.nodeList)
	e.Post("/plan/list", api.planList)
	e.Post("/client/list", api.clientList)
	e.Post("/snapshot/list", api.snapshotList)
	e.Post("/snapshot/delete", api.snapshotDelete)
	e.Post("/execute/snapshot/list", api.executeSnapshotList)
	api.echo = e
	return
}

func (e *JobAPI) Start(addr string, opts ...engine.ConfigSetter) {
	e.echo.Logger().Fatal(e.echo.Run(standard.New(addr, opts...)))
}

const sessionKey = `user`

func (api *JobAPI) login(context echo.Context) (err error) {
	var (
		message string
		signed  string
		claims  *jwt.StandardClaims
	)
	user := &InputLogin{}
	if err = context.MustBind(user); err != nil {
		message = "请求参数不能为空"
		goto ERROR
	}
	if user.Username == "" {
		message = "用户名不能为空"
		goto ERROR
	}
	if user.Password == "" {
		message = "用户密码为空"
		goto ERROR
	}
	if api.auth == nil || api.auth.Auth == nil {
		message = "未启用登录认证功能"
		goto ERROR
	}
	if err = api.auth.Auth(user); err != nil {
		message = err.Error()
		goto ERROR
	}
	claims = &jwt.StandardClaims{
		Audience:  context.Session().MustID(),
		ExpiresAt: 0,
		Id:        user.Username,
		IssuedAt:  0,
		Issuer:    `forest`,
		NotBefore: 0,
		Subject:   user.Username,
	}
	signed, err = mwjwt.BuildStandardSignedString(claims, []byte(api.auth.JWTKey))
	if err != nil {
		message = err.Error()
		goto ERROR
	}
	return context.JSON(Result{Code: 0, Data: echo.H{
		`token`: signed,
	}, Message: "登录成功"})

ERROR:
	return context.JSON(Result{Code: -1, Message: message})
}

func (api *JobAPI) logout(context echo.Context) (err error) {
	context.Session().Delete(sessionKey)
	return context.JSON(Result{Code: 0, Message: "登出成功"})
}

// add a new job
func (api *JobAPI) addJob(context echo.Context) (err error) {
	var (
		message string
	)
	jobConf := new(JobConf)
	if err = context.MustBind(jobConf); err != nil {
		message = "请求参数不能为空"
		goto ERROR
	}

	if jobConf.Name == "" {
		message = "任务名称不能为空"
		goto ERROR
	}
	if jobConf.Group == "" {
		message = "任务分组不能为空"
		goto ERROR
	}

	if jobConf.Cron == "" {
		message = "任务Cron表达式不能为空"
		goto ERROR
	}

	if _, err = cron.Parse(jobConf.Cron); err != nil {
		message = "非法的Cron表达式"
		goto ERROR
	}

	if jobConf.Target == "" {
		message = "任务Target不能为空"
		goto ERROR
	}

	if jobConf.Status == 0 {
		message = "任务状态不能为空"
		goto ERROR
	}

	if err = api.node.manager.AddJob(jobConf); err != nil {
		message = err.Error()
		goto ERROR
	}

	return context.JSON(Result{Code: 0, Data: jobConf, Message: "创建成功"})

ERROR:
	return context.JSON(Result{Code: -1, Message: message})
}

// edit a job
func (api *JobAPI) editJob(context echo.Context) (err error) {
	var (
		message string
	)
	jobConf := new(JobConf)
	if err = context.MustBind(jobConf); err != nil {
		message = "请求参数不能为空"
		goto ERROR
	}

	if jobConf.Id == "" {
		message = "此任务记录不存在"
		goto ERROR
	}
	if jobConf.Name == "" {
		message = "任务名称不能为空"
		goto ERROR
	}
	if jobConf.Group == "" {
		message = "任务分组不能为空"
		goto ERROR
	}

	if jobConf.Cron == "" {
		message = "任务Cron表达式不能为空"
		goto ERROR
	}

	if _, err = cron.Parse(jobConf.Cron); err != nil {
		message = "非法的Cron表达式"
		goto ERROR
	}

	if jobConf.Target == "" {
		message = "任务Target不能为空"
		goto ERROR
	}

	if jobConf.Status == 0 {
		message = "任务状态不能为空"
		goto ERROR
	}

	if err = api.node.manager.editJob(jobConf); err != nil {
		message = err.Error()
		goto ERROR
	}

	return context.JSON(Result{Code: 0, Data: jobConf, Message: "修改成功"})

ERROR:
	return context.JSON(Result{Code: -1, Message: message})
}

// job  list
func (api *JobAPI) jobList(context echo.Context) (err error) {

	var (
		jobConfs []*JobConf
	)

	if jobConfs, err = api.node.manager.jobList(); err != nil {
		return context.JSON(Result{Code: -1, Message: err.Error()})
	}
	return context.JSON(Result{Code: 0, Data: jobConfs, Message: "查询成功"})
}

// delete a job
func (api *JobAPI) deleteJob(context echo.Context) (err error) {

	var (
		message string
	)

	jobConf := new(JobConf)
	if err = context.MustBind(jobConf); err != nil {
		message = "请求参数不能为空"
		goto ERROR
	}

	if jobConf.Id == "" {
		message = "此任务记录不存在"
		goto ERROR
	}

	if err = api.node.manager.deleteJob(jobConf); err != nil {
		message = err.Error()
		goto ERROR
	}

	return context.JSON(Result{Code: 0, Data: jobConf, Message: "删除成功"})

ERROR:
	return context.JSON(Result{Code: -1, Message: message})
}

// add a job group
func (api *JobAPI) addGroup(context echo.Context) (err error) {

	var (
		message string
	)

	groupConf := new(GroupConf)
	if err = context.MustBind(groupConf); err != nil {
		message = "请求参数不能为空"
		goto ERROR
	}

	if groupConf.Name == "" {
		message = "任务集群名称不能为空"
		goto ERROR
	}

	if groupConf.Remark == "" {
		message = "任务集群描述不能为空"
		goto ERROR
	}

	if err = api.node.manager.addGroup(groupConf); err != nil {
		message = err.Error()
		goto ERROR
	}

	return context.JSON(Result{Code: 0, Data: groupConf, Message: "添加成功"})

ERROR:
	return context.JSON(Result{Code: -1, Message: message})
}

// edit a job group
func (api *JobAPI) editGroup(context echo.Context) (err error) {

	var (
		message string
	)

	groupConf := new(GroupConf)
	if err = context.MustBind(groupConf); err != nil {
		message = "请求参数不能为空"
		goto ERROR
	}

	if groupConf.Name == "" {
		message = "任务集群名称不能为空"
		goto ERROR
	}

	if groupConf.Remark == "" {
		message = "任务集群描述不能为空"
		goto ERROR
	}

	if err = api.node.manager.editGroup(groupConf); err != nil {
		message = err.Error()
		goto ERROR
	}

	return context.JSON(Result{Code: 0, Data: groupConf, Message: "添加成功"})

ERROR:
	return context.JSON(Result{Code: -1, Message: message})
}

// delete a group
func (api *JobAPI) deleteGroup(context echo.Context) (err error) {

	var (
		message string
	)

	groupConf := new(GroupConf)
	if err = context.MustBind(groupConf); err != nil {
		message = "请求参数不能为空"
		goto ERROR
	}

	if groupConf.Name == "" {
		message = "任务集群名称不能为空"
		goto ERROR
	}

	if err = api.node.manager.deleteGroup(groupConf); err != nil {
		message = err.Error()
		goto ERROR
	}

	return context.JSON(Result{Code: 0, Data: groupConf, Message: "删除成功"})

ERROR:
	return context.JSON(Result{Code: -1, Message: message})
}

// job group list
func (api *JobAPI) groupList(context echo.Context) (err error) {

	var (
		groupConfs []*GroupConf
	)

	if groupConfs, err = api.node.manager.groupList(); err != nil {
		return context.JSON(Result{Code: -1, Message: err.Error()})
	}
	return context.JSON(Result{Code: 0, Data: groupConfs, Message: "查询成功"})
}

// job node list
func (api *JobAPI) nodeList(context echo.Context) (err error) {

	var (
		nodes     []*Node
		leader    []byte
		nodeNames []string
	)

	if nodeNames, err = api.node.manager.nodeList(); err != nil {
		return context.JSON(Result{Code: -1, Message: err.Error()})
	}

	if leader, err = api.node.etcd.Get(JobNodeElectPath); err != nil {
		return context.JSON(Result{Code: -1, Message: err.Error()})
	}

	if len(nodeNames) == 0 {
		return context.JSON(Result{Code: 0, Data: nodes, Message: "查询成功"})
	}

	nodes = make([]*Node, 0)

	for _, name := range nodeNames {

		if name == string(leader) {
			nodes = append(nodes, &Node{Name: name, State: NodeLeaderState})
		} else {
			nodes = append(nodes, &Node{Name: name, State: NodeFollowerState})
		}

	}

	return context.JSON(Result{Code: 0, Data: nodes, Message: "查询成功"})
}

func (api *JobAPI) planList(context echo.Context) (err error) {

	var (
		plans []*SchedulePlan
	)

	schedulePlans := api.node.scheduler.schedulePlans
	if len(schedulePlans) == 0 {
		return context.JSON(Result{Code: 0, Data: plans})
	}

	plans = make([]*SchedulePlan, 0)

	for _, p := range schedulePlans {
		plans = append(plans, p)
	}

	return context.JSON(Result{Code: 0, Data: plans})
}

func (api *JobAPI) clientList(context echo.Context) (err error) {

	var (
		query     *QueryClientParam
		message   string
		group     *Group
		clients   []*JobClient
		groupPath string
	)

	query = new(QueryClientParam)
	if err = context.MustBind(query); err != nil {
		message = "请选择任务集群"
		goto ERROR
	}

	if query.Group == "" {
		message = "请选择任务集群"
		goto ERROR
	}

	groupPath = fmt.Sprintf("%s%s", GroupConfPath, query.Group)
	if group = api.node.groupManager.groups[groupPath]; group == nil {
		message = "此任务集群不存在"
		goto ERROR
	}

	clients = make([]*JobClient, 0)

	for _, c := range group.clients {
		clients = append(clients, &JobClient{Name: c.name, Path: c.path, Group: query.Group})
	}

	return context.JSON(Result{Code: 0, Data: clients, Message: "查询成功"})

ERROR:
	return context.JSON(Result{Code: -1, Message: message})
}

// 任务快照
func (api *JobAPI) snapshotList(context echo.Context) (err error) {

	var (
		query     *QuerySnapshotParam
		message   string
		keys      [][]byte
		values    [][]byte
		snapshots []*JobSnapshot
		prefix    string
	)

	query = new(QuerySnapshotParam)
	if err = context.MustBind(query); err != nil {
		message = "非法的请求参数"
		goto ERROR
	}

	prefix = JobSnapshotPath
	if query.Group != "" && query.Id != "" && query.Ip != "" {
		prefix = fmt.Sprintf(JobClientSnapshotPath, query.Group, query.Ip)
		prefix = fmt.Sprintf("%s/%s", prefix, query.Id)
	} else if query.Group != "" && query.Ip != "" {
		prefix = fmt.Sprintf(JobClientSnapshotPath, query.Group, query.Ip)
	} else if query.Group != "" && query.Ip == "" {
		prefix = fmt.Sprintf(JobSnapshotGroupPath, query.Group)
	}

	if keys, values, err = api.node.etcd.GetWithPrefixKeyLimit(prefix, 500); err != nil {
		message = err.Error()
		goto ERROR
	}

	snapshots = make([]*JobSnapshot, 0)
	if len(keys) == 0 {
		return context.JSON(Result{Code: 0, Data: snapshots, Message: "查询成功"})
	}

	for _, value := range values {

		if len(value) == 0 {
			continue
		}

		var snapshot *JobSnapshot

		if snapshot, err = UnpackJobSnapshot(value); err != nil {
			continue
		}

		snapshots = append(snapshots, snapshot)
	}

	return context.JSON(Result{Code: 0, Data: snapshots, Message: "查询成功"})

ERROR:
	return context.JSON(Result{Code: -1, Message: message})
}

// 任务删除任务快照
func (api *JobAPI) snapshotDelete(context echo.Context) (err error) {

	var (
		query   *QuerySnapshotParam
		message string
		key     string
	)

	query = new(QuerySnapshotParam)
	if err = context.MustBind(query); err != nil {
		message = "非法的请求参数"
		goto ERROR
	}

	if query.Group == "" || query.Id == "" || query.Ip == "" {
		message = "非法的请求参数"
		goto ERROR
	}

	key = fmt.Sprintf(JobClientSnapshotPath, query.Group, query.Ip)
	key = fmt.Sprintf("%s/%s", key, query.Id)
	if err = api.node.etcd.Delete(key); err != nil {
		message = err.Error()
		goto ERROR
	}
	return context.JSON(Result{Code: 0, Message: "删除成功"})

ERROR:
	return context.JSON(Result{Code: -1, Message: message})
}

func (api *JobAPI) executeSnapshotList(context echo.Context) (err error) {

	var (
		query     *QueryExecuteSnapshotParam
		message   string
		count     uint64
		snapshots []*JobExecuteSnapshot
		totalPage uint64
		where     = db.NewCompounds()
	)

	query = new(QueryExecuteSnapshotParam)
	if err = context.MustBind(query); err != nil {
		message = "非法的请求参数"
		goto ERROR
	}

	if query.PageSize <= 0 {
		query.PageSize = 10
	}

	if query.PageNo <= 0 {
		query.PageNo = 1
	}

	snapshots = make([]*JobExecuteSnapshot, 0)
	if query.Id != "" {
		where.AddKV(`id`, query.Id)
	}
	if query.Group != "" {
		where.AddKV(`group`, query.Group)
	}
	if query.Ip != "" {
		where.AddKV(`ip`, query.Ip)
	}
	if query.Name != "" {
		where.AddKV(`name`, query.Name)
	}
	if query.Status != 0 {
		where.AddKV(`status`, query.Status)
	}
	if count, err = api.node.DB().Collection(`job_execute_snapshot`).Find(where.And()).Count(); err != nil {
		log.Errorf("err: %#v", err)
		message = "查询失败"
		goto ERROR
	}

	if count > 0 {
		err = api.node.DB().Collection(`job_execute_snapshot`).Find(where.And()).OrderBy(`-create_time`).Limit(query.PageSize).Offset((query.PageNo - 1) * query.PageSize).All(&snapshots)
		if err != nil {
			log.Errorf("err: %#v", err)
			message = "查询失败"
			goto ERROR
		}

		if count%uint64(query.PageSize) == 0 {
			totalPage = count / uint64(query.PageSize)
		} else {
			totalPage = count/uint64(query.PageSize) + 1
		}

	}

	return context.JSON(Result{Code: 0, Data: &PageResult{TotalCount: int(count), TotalPage: int(totalPage), List: &snapshots}, Message: "查询成功"})

ERROR:
	return context.JSON(Result{Code: -1, Message: message})
}

// manual execute
func (api *JobAPI) manualExecute(context echo.Context) (err error) {

	var (
		conf         *JobConf
		value        []byte
		client       *Client
		snapshotPath string
		snapshot     *JobSnapshot
		success      bool
	)

	conf = new(JobConf)
	if err = context.MustBind(conf); err != nil {
		return context.JSON(Result{Code: -1, Message: "非法的参数"})
	}

	// 检查任务配置id是否存在
	if conf.Id == "" {
		return context.JSON(Result{Code: -1, Message: "此任务配置不存在"})
	}

	// 查询任务配置
	if value, err = api.node.etcd.Get(JobConfPath + conf.Id); err != nil {
		return context.JSON(Result{Code: -1, Message: "查询任务配置出现异常:" + err.Error()})
	}

	// 任务配置是否为空
	if len(value) == 0 {
		return context.JSON(Result{Code: -1, Message: "此任务配置内容为空"})
	}

	if conf, err = UnpackJobConf(value); err != nil {
		return context.JSON(Result{Code: -1, Message: "非法的任务配置内容"})
	}

	// select a execute  job client for group
	if client, err = api.node.groupManager.selectClient(conf.Group); err != nil {
		return context.JSON(Result{Code: -1, Message: "没有找到可以执行此任务的作业节点"})
	}

	// build the job snapshot path
	snapshotPath = fmt.Sprintf(JobClientSnapshotPath, conf.Group, client.name)

	// build job snapshot
	snapshotId := GenerateSerialNo() + conf.Id
	snapshot = &JobSnapshot{
		Id:         snapshotId,
		JobId:      conf.Id,
		Name:       conf.Name,
		Ip:         client.name,
		Group:      conf.Group,
		Cron:       conf.Cron,
		Target:     conf.Target,
		Params:     conf.Params,
		Mobile:     conf.Mobile,
		Remark:     conf.Remark,
		CreateTime: ToDateString(time.Now()),
	}

	// park the job snapshot
	if value, err = PackJobSnapshot(snapshot); err != nil {
		return context.JSON(Result{Code: -1, Message: err.Error()})
	}

	// dispatch the job snapshot the client
	if success, _, err = api.node.etcd.PutNotExist(snapshotPath, string(value)); err != nil {
		return context.JSON(Result{Code: -1, Message: err.Error()})
	}

	if !success {
		return context.JSON(Result{Code: -1, Message: "手动执行任务失败,请重试"})
	}

	return context.JSON(Result{Code: 0, Message: "手动执行任务请求已提交"})
}
