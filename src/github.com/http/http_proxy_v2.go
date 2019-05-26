package main

import (
	"bufio"
	"fmt"
	"github.com/go-ini/ini"
	"github.com/op/go-logging"
	"net"
	"net/http"
	"os"
)

const (
	COLLAGEN_REQUESTER_ID  string = "COLLAGEN-REQUESTER_ID"
	COLLAGEN_AUTHORIZE_ID  string = "COLLAGEN-AUTHORIZE_ID"
	COLLAGEN_SESSION_ID    string = "COLLAGEN-SESSION_ID"
	COLLAGEN_TIMEOUT       string = "COLLAGEN-TIMEOUT"
	COLLAGEN_PROXY_FLOW_ID string = "COLLAGEN-PROXY_FLOW_ID"

	ACCESS_IP     string = "accessIp"
	ACCESS_PORT   string = "accessPort"
	ACCESS_DOMAIN string = "accessDomain"
	SERVICE_ID    string = "serviceId"

	LOCAL_IP   string = "localIp"
	LOCAL_PORT string = "localPort"

	LEVEL string = "level"
)

var (
	accessIp, accessPort, accessDomain, serviceId string
	localIp, localPort                            string
	headerMap                                     map[string]string
)

var log = logging.MustGetLogger("http_proxy")
var format = logging.MustStringFormatter(
	`%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x} %{message}`,
)

func init() {
	cfg, err := ini.Load("config.ini")
	if err != nil {
		fmt.Printf("Fail to read file: %v\n", err)
		os.Exit(1)
	}

	logFile, err := os.OpenFile("http_proxy.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	fileBackend := logging.NewLogBackend(logFile, "", 0)
	stdBackend := logging.NewLogBackend(os.Stdout, "", 0)

	fileBackendFormatter := logging.NewBackendFormatter(fileBackend, format)
	stdBackendFormatter := logging.NewBackendFormatter(stdBackend, format)

	fileLeveled := logging.AddModuleLevel(fileBackendFormatter)
	stdLeveled := logging.AddModuleLevel(stdBackendFormatter)

	levelStr := cfg.Section("").Key(LEVEL).String()
	level, _ := logging.LogLevel(levelStr)
	fileLeveled.SetLevel(level, "")
	stdLeveled.SetLevel(level, "")

	logging.SetBackend(fileLeveled, stdLeveled)

	accessIp = cfg.Section("").Key(ACCESS_IP).String()
	accessPort = cfg.Section("").Key(ACCESS_PORT).String()
	accessDomain = cfg.Section("").Key(ACCESS_DOMAIN).String()
	serviceId = cfg.Section("").Key(SERVICE_ID).String()

	localIp = cfg.Section("").Key(LOCAL_IP).String()
	localPort = cfg.Section("").Key(LOCAL_PORT).String()

	headerMap = make(map[string]string)
	headerMap[COLLAGEN_REQUESTER_ID] = cfg.Section("").Key(COLLAGEN_REQUESTER_ID).String()
	headerMap[COLLAGEN_AUTHORIZE_ID] = cfg.Section("").Key(COLLAGEN_AUTHORIZE_ID).String()
	headerMap[COLLAGEN_SESSION_ID] = cfg.Section("").Key(COLLAGEN_SESSION_ID).String()
	headerMap[COLLAGEN_TIMEOUT] = cfg.Section("").Key(COLLAGEN_TIMEOUT).String()
	headerMap[COLLAGEN_PROXY_FLOW_ID] = cfg.Section("").Key(COLLAGEN_PROXY_FLOW_ID).String()
}

func main() {
	t := net.JoinHostPort(localIp, localPort)
	ln, err := net.Listen("tcp", t)
	if err != nil {
		panic(err)
	}

	log.Infof("Started listening on %v\n", t)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		log.Debugf("Received connection from %v\n", conn.RemoteAddr().String())

		go handleLoop(conn)
	}
}

func handleLoop(conn net.Conn) {
	defer conn.Close()
	brw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	var req *http.Request
	reqc := make(chan *http.Request, 1)
	errc := make(chan error, 1)

	go func() {
		r, err := http.ReadRequest(brw.Reader)
		if err != nil {
			errc <- err
			return
		}
		reqc <- r
	}()

	select {
	case err := <-errc:
		log.Debugf("failed to read request: %v\n", err)
		return
	case req = <-reqc:
	}
	defer req.Body.Close()

	req.URL.Scheme = "http"
	host := accessIp + ":" + accessPort
	req.Host = host
	req.URL.Host = host

	path := req.URL.Path
	remoteUrl := "/" + accessDomain + "/03/" + serviceId + path
	req.URL.Path = remoteUrl
	req.RequestURI = remoteUrl

	for k, v := range headerMap {
		req.Header.Add(k, v)
	}

	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		log.Errorf("failed to round trip: %v\n", err)
		return
	}
	defer res.Body.Close()

	err = res.Write(brw)
	if err != nil {
		log.Errorf("got error while writing response back to client: %v\n", err)
		return
	}
	err = brw.Flush()
	if err != nil {
		log.Errorf("got error while flushing response back to client: %v\n", err)
		return
	}
}
