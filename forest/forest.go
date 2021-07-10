package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
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
	defaultVersion      = `0.2.4`
)

var defaultAPISecret = os.Getenv("FOREST_API_SECRET")

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
	etcdEndpoints := flag.String("etcd-endpoints", defaultEndpoints, "--etcd endpoints "+defaultEndpoints)
	etcdDialTime := flag.Int64("etcd-dailtimeout", defaultDialTimeout, "--etcd dailtimeout "+strconv.Itoa(defaultDialTimeout))

	// API Server
	apiCertFile := flag.String("api-tls-cert", defaultAPIHttpsCert, "--api-tls-cert file")
	apiKeyFile := flag.String("api-tls-key", defaultAPIHttpsKey, "--api-tls-key file")
	apiAddress := flag.String("api-address", defaultHTTPAddress, "---api-address "+defaultHTTPAddress)
	apiJWTKey := flag.String("api-jwtkey", com.ByteMd5(securecookie.GenerateRandomKey(32)), "--api-jwtkey 01234567890123456789012345678901")

	flag.StringVar(&defaultAPISecret, "api-secret", defaultAPISecret, "--api-secret 01234567890123456789012345678901 (也可以通过环境变量FOREST_API_SECRET来指定)")

	flag.DurationVar(&forest.ExecuteSnapshotCanRetry, "api-can-retry", forest.ExecuteSnapshotCanRetry, "--api-can-retry 6h") // 指定开始多长时间后可以重试，默认6h

	// - admin
	admName := flag.String("admin-name", "admin", "--admin-name admin (也可以通过环境变量FOREST_ADMIN_NAME来指定)")
	admPassword := flag.String("admin-password", "", "--admin-password root (也可以通过环境变量FOREST_ADMIN_PASSWORD来指定)")

	// Database
	dsn := flag.String("dsn", defaultDSN, `--dsn="root:root@tcp(127.0.0.1:3306)/forest?charset=utf8" (也可以通过环境变量FOREST_DSN来指定)`)

	// Node
	currentIP := flag.String("current-ip", ip, "--current-ip "+ip)

	// Other
	debug := flag.Bool("debug", defaultDebug, "--debug false")
	help := flag.Bool("help", false, "--help")
	version := flag.Bool("version", false, "--version")
	flag.Parse()

	if *debug {
		log.SetLevel(`Debug`)
	} else {
		log.SetLevel(`Info`)
	}

	if *help {
		flag.Usage()
		return
	}
	if *version {
		fmt.Println(`v` + defaultVersion)
		return
	}

	if defaultAPISecret != os.Getenv("FOREST_API_SECRET") {
		os.Setenv("FOREST_API_SECRET", defaultAPISecret)
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
