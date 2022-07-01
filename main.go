package main

import (
	"bytes"
	"compress/gzip"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// effectively const values defined by environment variables
var (
	host            = getenv("PN_HOST", "")
	port, _         = strconv.Atoi(getenv("PN_PORT", "0"))
	network         = getenv("PN_NETWORK", "tcp4")
	frontendUrl     = getenv("PN_FRONTEND_URL", "https://pastes.dev/")
	userAgent       = getenv("PN_USER_AGENT", "paste-netcat")
	apiKey          = getenv("PN_API_KEY", "")
	apiPostUrl      = getenv("PN_API_POST_URL", "https://api.pastes.dev/post")
	postContentType = getenv("PN_POST_CONTENT_TYPE", "text/plain")
)

func getenv(key string, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	} else {
		return def
	}
}

// Entrypoint
func main() {
	host := flag.String("h", host, "The host to bind to")
	port := flag.Int("p", port, "The port to bind to")
	network := flag.String("n", network, "The network to listen on, should be 'tcp', 'tcp4' or 'tcp6'")
	flag.Parse()

	startServer(*host, *port, *network)
}

// Starts a TCP socket server listening on the given host/port
func startServer(host string, port int, network string) {
	bind := fmt.Sprintf("%s:%d", host, port)

	listener, err := net.Listen(network, bind)
	if err != nil {
		panic(err)
	}

	fmt.Printf("listening on %s\n", listener.Addr().String())

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("error accepting connection: %s\n", err)
		} else {
			go process(conn)
		}
	}
}

// Process an incoming connection
func process(conn net.Conn) {
	defer conn.Close()

	// set a 30-second timeout for this connection
	_ = conn.SetDeadline(time.Now().Add(time.Second * 30))

	// parse remote ip address
	ipAddr := conn.RemoteAddr().String()
	if strings.Contains(ipAddr, ":") {
		ipAddr = strings.Split(ipAddr, ":")[0]
	}

	// create a gzip-wrapped buffer to read content into
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)

	// copy from the connection -> buf (and apply gzip compression)
	iterations := 0
	var totalWritten int64 = 0
	for {
		_ = conn.SetReadDeadline(time.Now().Add(time.Millisecond * 200))
		written, err := io.Copy(writer, conn)
		totalWritten += written
		iterations++

		// continue reading while: data is being written or nothing has been written yet, up to 5 iters
		if err != nil && errors.Is(err, os.ErrDeadlineExceeded) && (written > 0 || (totalWritten == 0 && iterations < 5)) {
			continue
		}

		break
	}

	_ = conn.SetDeadline(time.Now().Add(time.Second * 30))

	// flush+close the writer
	err := writer.Close()
	if err != nil {
		fmt.Printf("error reading from connection %s: %s\n", ipAddr, err)
		return
	}

	if totalWritten == 0 {
		fmt.Printf("no content from %s\n", ipAddr)
		_, _ = fmt.Fprintln(conn, "no content received!")
		return
	}

	if totalWritten < 100 {
		fmt.Printf("possible spam from %s\n", ipAddr)
		_, _ = fmt.Fprintln(conn, "request not ok")
		return
	}

	// perform an HTTP post request to the paste API to upload the content
	code, err := post(&buf, ipAddr)
	if err != nil {
		fmt.Printf("error uploading content from %s: %s\n", ipAddr, err)
		_, _ = fmt.Fprintln(conn, "error uploading")
		return
	}

	// reply via the socket with the URL
	_, err = fmt.Fprintf(conn, "%s%s\n", frontendUrl, code)
	if err != nil {
		fmt.Printf("error writing response to connection %s: %s\n", ipAddr, err)
		return
	}

	fmt.Printf("processed %s -> %s\n", ipAddr, code)
}

// Posts compressed content to the paste (Bytebin) API
func post(body io.Reader, ipAddr string) (string, error) {
	req, err := http.NewRequest("POST", apiPostUrl, body)
	if err != nil {
		return "", err
	}

	req.Header.Add("User-Agent", userAgent)

	if len(apiKey) > 0 {
		req.Header.Add("Bytebin-Api-Key", apiKey)
		req.Header.Add("Bytebin-Forwarded-For", ipAddr)
	}

	req.Header.Add("Content-Type", postContentType)
	req.Header.Add("Content-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	_ = resp.Body.Close()

	if resp.StatusCode != 201 {
		return "", fmt.Errorf("invalid response code %v", resp.StatusCode)
	}

	return resp.Header.Get("Location"), nil
}
