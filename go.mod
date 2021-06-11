module github.com/andistributed/forest

go 1.16

replace github.com/andistributed/etcd => ../etcd

require (
	github.com/admpub/log v0.3.1
	github.com/admpub/securecookie v1.1.2
	github.com/admpub/sessions v0.1.1 // indirect
	github.com/andistributed/etcd v0.1.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/robfig/cron v1.2.0
	github.com/webx-top/com v0.2.2
	github.com/webx-top/db v1.4.2
	github.com/webx-top/echo v2.14.4+incompatible
	go.etcd.io/etcd/client/v3 v3.5.0-rc.1
	go.uber.org/multierr v1.7.0 // indirect
)
