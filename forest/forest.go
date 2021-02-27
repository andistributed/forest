package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/admpub/securecookie"
	"github.com/andistributed/forest"
	"github.com/andistributed/forest/etcd"
	"github.com/coreos/etcd/clientv3"
	"github.com/prometheus/common/log"
	"github.com/webx-top/com"
	"github.com/webx-top/echo/engine"
)

const (
	defaultEndpoints    = "127.0.0.1:2379"
	defaultHTTPAddress  = ":2856"
	defaultDialTimeout  = 5
	defaultDSN          = "root:123456@tcp(127.0.0.1:3306)/forest?charset=utf8"
	defaultEtcdCert     = `` // ca.crt
	defaultEtcdKey      = `` // ca.key
	defaultAPIHttpsCert = ``
	defaultAPIHttpsKey  = ``
)

var (
	errPasswordInvalid = errors.New("密码不正确")
)

// go run forest.go --dsn="root:root@tcp(127.0.0.1:3306)/forest?charset=utf8" --admin-password=root
func main() {

	ip := forest.GetLocalIpAddress()
	if ip == "" {
		log.Fatal("has no get the ip address")
	}

	// ETCD
	etcdCertFile := flag.String("etcd-cert", defaultEtcdCert, "etcd-cert file")
	etcdKeyFile := flag.String("etcd-key", defaultEtcdKey, "etcd-key file")
	etcdEndpoints := flag.String("etcd-endpoints", defaultEndpoints, "etcd endpoints")
	etcdDialTime := flag.Int64("etcd-dailtimeout", defaultDialTimeout, "etcd dailtimeout")

	// API Server
	apiCertFile := flag.String("api-tls-cert", defaultAPIHttpsCert, "api-tls-cert file")
	apiKeyFile := flag.String("api-tls-key", defaultAPIHttpsKey, "api-tls-key file")
	apiAddress := flag.String("api-address", defaultHTTPAddress, "http address")
	apiJWTKey := flag.String("api-jwtkey", com.ByteMd5(securecookie.GenerateRandomKey(32)), "jwt key")

	// - admin
	admName := flag.String("admin-name", "admin", "admin name")
	admPassword := flag.String("admin-password", "", "admin password")

	// Database
	dsn := flag.String("dsn", defaultDSN, "dsn for mysql")

	help := flag.String("help", "", "forest help")
	flag.Parse()
	if *help != "" {
		flag.Usage()
		return
	}

	endpoint := strings.Split(*etcdEndpoints, ",")
	dialTime := time.Duration(*etcdDialTime) * time.Second
	var etcdOpts []func(*clientv3.Config)
	if len(*etcdCertFile) > 0 && len(*etcdKeyFile) > 0 {
		etcdOpts = append(etcdOpts, etcd.TLSFile(*etcdCertFile, *etcdKeyFile))
	}
	etcd, err := forest.NewEtcd(endpoint, dialTime, etcdOpts...)
	if err != nil {
		log.Fatal(err)
	}
	node, err := forest.NewJobNode(ip, etcd, *dsn)
	if err != nil {
		log.Fatal(err)
	}
	auth := newAPIAuth(*admName, *admPassword, *apiJWTKey)
	go startAPIServer(node, auth, *apiAddress, *apiCertFile, *apiKeyFile)

	node.Bootstrap()
}

func startAPIServer(node *forest.JobNode, auth *forest.ApiAuth, httpAddress, apiCertFile, apiKeyFile string) {
	var httpServerOpts []engine.ConfigSetter
	httpServerOpts = append(httpServerOpts, engine.TLSCertFile(apiCertFile))
	httpServerOpts = append(httpServerOpts, engine.TLSKeyFile(apiKeyFile))
	node.StartAPIServer(auth, httpAddress, httpServerOpts...)
}

func newAPIAuth(admName, admPassword, jwtKey string) *forest.ApiAuth {
	if len(admPassword) == 0 {
		admPassword = os.Getenv("FOREST_ADMIN_PASSWORD")
	}
	if len(admName) == 0 {
		admName = os.Getenv("FOREST_ADMIN_NAME")
	}
	if len(admName) == 0 {
		admName = `admin`
	}
	auth := &forest.ApiAuth{
		Auth: func(user *forest.InputLogin) error {
			if user.Username != admName {
				return fmt.Errorf("用户名不正确: %s", user.Username)
			}
			if user.Password != admPassword {
				return errPasswordInvalid
			}
			return nil
		},
		JWTKey: jwtKey,
	}
	return auth
}
