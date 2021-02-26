package main

import (
	"flag"
	"strings"
	"time"

	"github.com/andistributed/forest"
	"github.com/andistributed/forest/etcd"
	"github.com/coreos/etcd/clientv3"
	"github.com/prometheus/common/log"
)

const (
	DefaultEndpoints   = "127.0.0.1:2379"
	DefaultHttpAddress = ":2856"
	DefaultDialTimeout = 5
	DefaultDbUrl       = "root:123456@tcp(127.0.0.1:3306)/forest?charset=utf8"
	DefaultEtcdCert    = `` //"/etc/kubernetes/pki/etcd/ca.crt"
	DefaultEtcdKey     = `` //"/etc/kubernetes/pki/etcd/ca.key"
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

	node, err := forest.NewJobNode(ip, etcd, *httpAddress, *dbUrl)
	if err != nil {

		log.Fatal(err)
	}

	node.Bootstrap()
}
