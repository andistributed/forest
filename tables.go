package forest

const (
	TableJobExecuteSnapshot = `job_execute_snapshot`
)

var (
	InitSQLs = []string{
		"CREATE TABLE `job_execute_snapshot` (\n" +
			"`id` varchar(64) NOT NULL COMMENT '主键',\n" +
			"`job_id` varchar(32) NOT NULL COMMENT '任务定义id',\n" +
			"`name` varchar(32) NOT NULL COMMENT '任务名称',\n" +
			"`group` varchar(32) NOT NULL COMMENT '任务集群',\n" +
			"`cron` varchar(32) NOT NULL COMMENT 'cron表达式',\n" +
			"`target` varchar(255) NOT NULL COMMENT '目标任务',\n" +
			"`params` varchar(255) DEFAULT NULL COMMENT '参数',\n" +
			"`ip` varchar(32) NOT NULL COMMENT 'ip',\n" +
			"`status` tinyint(4) NOT NULL COMMENT '状态 1-执行中 2-完成  3-未知  4-错误',\n" +
			"`mobile` varchar(32) DEFAULT NULL COMMENT '手机号码',\n" +
			"`remark` varchar(255) DEFAULT NULL COMMENT '备注',\n" +
			"`create_time` varchar(32) NOT NULL COMMENT '创建时间',\n" +
			"`start_time` varchar(255) DEFAULT NULL COMMENT '开始时间',\n" +
			"`finish_time` varchar(32) DEFAULT NULL COMMENT '结束时间',\n" +
			"`times` bigint(20) DEFAULT '0' COMMENT '耗时',\n" +
			"`result` varchar(255) DEFAULT NULL COMMENT '返回结果',\n" +
			"PRIMARY KEY (`id`),\n" +
			"KEY `ip` (`ip`),\n" +
			"KEY `job_id` (`job_id`),\n" +
			"KEY `status` (`status`),\n" +
			"KEY `group` (`group`)\n" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='任务作业执行快照';\n",
	}
)
