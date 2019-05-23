package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
)

func main() {
	http.HandleFunc("/", helloHandler)
	err := http.ListenAndServe(":5003", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	r.URL.Host = "192.168.0.160:9009"
	r.URL.Scheme = "http"
	r.Host = "192.168.0.160:9009"
	path := r.URL.Path
	remoteUrl := "/XM.JM/03/XM.GOV.YZ.JM.MSJM.rest_post_test180322" + path
	r.URL.Path = remoteUrl
	r.RequestURI = remoteUrl

	r.Header.Add("COLLAGEN-REQUESTER_ID", "XM.GOV.JM.APP.JMXT")
	r.Header.Add("COLLAGEN-AUTHORIZE_ID", "62540c73973832401da0fd3b4206f5de")
	r.Header.Add("COLLAGEN-SESSION_ID", "51231515")
	r.Header.Add("COLLAGEN-TIMEOUT", "30000")
	r.Header.Add("COLLAGEN-PROXY_FLOW_ID", "XM.JM::01::FLOW_C3_CALL_RESTFUL_PROXY")

	temp := `GET /dagital-china-service/publicsafety/getApplicationGeneralSituation HTTP/1.1
User-Agent: PostmanRuntime/7.13.0
Accept: */*
Cache-Control: no-cache
Postman-Token: 2a274ff5-a194-4771-8250-6b868de1412e
Host: 192.168.249.133:12345
accept-encoding: gzip, deflate
Connection: keep-alive


`
	data := []byte(temp)

	e, ok := w.(http.Hijacker)
	if !ok {
		return
	}

	conn, _, err := e.Hijack()
	if err != nil {
		return
	}

	go forwardConn(conn, data)

	//resp, err := http.DefaultTransport.RoundTrip(r)
	//if err != nil {
	//	fmt.Println(err.Error())
	//	return
	//}
	//copyHeader(w.Header(), resp.Header)
	//w.WriteHeader(resp.StatusCode)
	//_, _ = io.Copy(w, resp.Body)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func forwardConn(clientConn net.Conn, data []byte) {
	t := net.JoinHostPort("183.250.160.154", "16060")

	// Create a connection to server
	serverConn, err := net.Dial("tcp", t)
	if err != nil {
		fmt.Println(err)
		_ = clientConn.Close()
		return
	}

	// Client to server channel
	//c2s := make(chan []byte, 2048)
	// Server to client channel
	s2c := make(chan []byte, 2048)
	//go httpReadSocket(clientConn, c2s)
	//go httpWriteSocket(serverConn, c2s)
	_, _ = serverConn.Write(data)
	go httpReadSocket(serverConn, s2c)
	go httpWriteSocket(clientConn, s2c)
}

func httpReadSocket(conn net.Conn, c chan<- []byte) {
	buf := make([]byte, 2048)
	// Store remote IP:port for logging
	rAddr := conn.RemoteAddr().String()

	for {
		// Read from connection
		n, err := conn.Read(buf)
		// If connection is closed from the other side
		if err == io.EOF {
			// Close the connction and return
			fmt.Println("Connection closed from", rAddr)
			_ = conn.Close()
			return
		}
		// For other errors, print the error and return
		if err != nil {
			fmt.Println("Error reading from socket", err)
			return
		}
		// Print data read from socket
		// Note we are only printing and sending the first n bytes.
		// n is the number of bytes read from the connection
		fmt.Printf("Received from %v: %s\n", rAddr, buf[:n])
		// Send data to channel
		c <- buf[:n]
	}
}

func httpWriteSocket(conn net.Conn, c <-chan []byte) {
	// Create a buffer to hold data
	buf := make([]byte, 2048)
	// Store remote IP:port for logging
	rAddr := conn.RemoteAddr().String()

	for {
		// Read from channel and copy to buffer
		buf = <-c
		// Write buffer
		n, err := conn.Write(buf)
		// If connection is closed from the other side
		if err == io.EOF {
			// Close the connction and return
			fmt.Println("Connection closed from", rAddr)
			_ = conn.Close()
			return
		}
		// For other errors, print the error and return
		if err != nil {
			fmt.Println("Error writing to socket", err)
			return
		}
		// Log data sent
		fmt.Printf("Sent to %v: %s\n", rAddr, buf[:n])
	}
}
