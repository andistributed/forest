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
)

const (
	DefaultEndpoints   = "127.0.0.1:2379"
	DefaultHttpAddress = ":2856"
	DefaultDialTimeout = 5
	DefaultDbUrl       = "root:123456@tcp(127.0.0.1:3306)/forest?charset=utf8"
	DefaultEtcdCert    = `` //"/etc/kubernetes/pki/etcd/ca.crt"
	DefaultEtcdKey     = `` //"/etc/kubernetes/pki/etcd/ca.key"
)

var (
	errPasswordInvalid = errors.New("密码不正确")
)

func main() {

	ip := forest.GetLocalIpAddress()
	if ip == "" {
		log.Fatal("has no get the ip address")

	}
	etcdCertFile := flag.String("etcd-cert", DefaultEtcdCert, "etcd-cert file")
	etcdKeyFile := flag.String("etcd-key", DefaultEtcdKey, "etcd-key file")
	endpoints := flag.String("etcd-endpoints", DefaultEndpoints, "etcd endpoints")
	httpAddress := flag.String("http-address", DefaultHttpAddress, "http address")
	jwtKey := flag.String("jwtkey", com.ByteMd5(securecookie.GenerateRandomKey(32)), "jwt key")
	admName := flag.String("admin.name", "admin", "admin name")
	admPassword := flag.String("admin.password", "", "admin password")
	etcdDialTime := flag.Int64("etcd-dailtimeout", DefaultDialTimeout, "etcd dailtimeout")
	help := flag.String("help", "", "forest help")
	dbUrl := flag.String("db-url", DefaultDbUrl, "db-url for mysql")
	flag.Parse()
	if *help != "" {
		flag.Usage()
		return
	}

	endpoint := strings.Split(*endpoints, ",")
	dialTime := time.Duration(*etcdDialTime) * time.Second
	var etcdOpts []func(*clientv3.Config)
	if len(*etcdCertFile) > 0 && len(*etcdKeyFile) > 0 {
		etcdOpts = append(etcdOpts, etcd.TLSFile(*etcdCertFile, *etcdKeyFile))
	}
	etcd, err := forest.NewEtcd(endpoint, dialTime, etcdOpts...)
	if err != nil {
		log.Fatal(err)
	}
	if len(*admPassword) == 0 {
		*admPassword = os.Getenv("FOREST_ADMIN_PASSWORD")
	}
	if len(*admName) == 0 {
		*admName = os.Getenv("FOREST_ADMIN_NAME")
	}
	if len(*admName) == 0 {
		*admName = `admin`
	}
	auth := &forest.ApiAuth{
		Auth: func(user *forest.InputLogin) error {
			if user.Username != *admName {
				return fmt.Errorf("用户名不正确: %s", user.Username)
			}
			if user.Password != *admPassword {
				return errPasswordInvalid
			}
			return nil
		},
		JWTKey: *jwtKey,
	}
	node, err := forest.NewJobNode(ip, etcd, *httpAddress, *dbUrl, auth)
	if err != nil {
		log.Fatal(err)
	}

	node.Bootstrap()
}
