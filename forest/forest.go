package main

import (
	"flag"
	"strings"
	"time"

	"github.com/admpub/log"
	"github.com/admpub/securecookie"
	"github.com/andistributed/etcd"
	"github.com/andistributed/etcd/etcdconfig"
	"github.com/andistributed/forest"
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
	defaultDebug        = false
)

// go run forest.go --dsn="root:root@tcp(127.0.0.1:3306)/forest?charset=utf8" --admin-password=root
func main() {
	log.SetFatalAction(log.ActionPanic)
	defer log.Close()

	ip := forest.GetLocalIpAddress()
	if len(ip) == 0 {
		log.Fatal("has no get the ip address")
	}

	// ETCD
	etcdCertFile := flag.String("etcd-cert", defaultEtcdCert, "--etcd-cert file")
	etcdKeyFile := flag.String("etcd-key", defaultEtcdKey, "--etcd-key file")
	etcdEndpoints := flag.String("etcd-endpoints", defaultEndpoints, "--etcd endpoints")
	etcdDialTime := flag.Int64("etcd-dailtimeout", defaultDialTimeout, "--etcd dailtimeout")

	// API Server
	apiCertFile := flag.String("api-tls-cert", defaultAPIHttpsCert, "--api-tls-cert file")
	apiKeyFile := flag.String("api-tls-key", defaultAPIHttpsKey, "--api-tls-key file")
	apiAddress := flag.String("api-address", defaultHTTPAddress, "---api-address "+defaultHTTPAddress)
	apiJWTKey := flag.String("api-jwtkey", com.ByteMd5(securecookie.GenerateRandomKey(32)), "--api-jwtkey 01234567890123456789012345678901")

	// - admin
	admName := flag.String("admin-name", "admin", "--admin-name admin")
	admPassword := flag.String("admin-password", "", "--admin-password root")

	// Database
	dsn := flag.String("dsn", defaultDSN, `--dsn="root:root@tcp(127.0.0.1:3306)/forest?charset=utf8"`)

	// Node
	currentIP := flag.String("current-ip", ip, "--current-ip "+ip)

	// Other
	debug := flag.Bool("debug", defaultDebug, "--debug false")
	help := flag.String("help", "", "--help")
	flag.Parse()

	if *debug {
		log.SetLevel(`Debug`)
	} else {
		log.SetLevel(`Info`)
	}

	if *help != "" {
		flag.Usage()
		return
	}

	endpoint := strings.Split(*etcdEndpoints, ",")
	dialTime := time.Duration(*etcdDialTime) * time.Second
	var etcdOpts []etcdconfig.Configer
	if len(*etcdCertFile) > 0 && len(*etcdKeyFile) > 0 {
		etcdOpts = append(etcdOpts, etcdconfig.TLSFile(*etcdCertFile, *etcdKeyFile))
	}
	etcd, err := etcd.New(endpoint, dialTime, etcdOpts...)
	if err != nil {
		log.Fatal(err)
	}
	node, err := forest.NewJobNode(*currentIP, etcd, *dsn)
	if err != nil {
		log.Fatal(err)
	}
	auth := forest.NewAPIAuth(*admName, *admPassword, *apiJWTKey)
	go startAPIServer(node, auth, *apiAddress, *apiCertFile, *apiKeyFile)

	node.Bootstrap()
}

func startAPIServer(node *forest.JobNode, auth *forest.APIAuth, httpAddress, apiCertFile, apiKeyFile string) {
	var httpServerOpts []engine.ConfigSetter
	httpServerOpts = append(httpServerOpts, engine.TLSCertFile(apiCertFile))
	httpServerOpts = append(httpServerOpts, engine.TLSKeyFile(apiKeyFile))
	node.StartAPIServer(auth, httpAddress, httpServerOpts...)
}
