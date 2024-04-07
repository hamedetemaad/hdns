package main

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"

	"strings"

	"github.com/fsnotify/fsnotify"
)

type Config struct {
	SERVICE_HOST      string   `json:"service_host" default:"127.0.0.1"`
	SERVICE_PORT      string   `json:"service_port" default:"1053"`
	SERVICE_PROTO     string   `json:"service_proto" default:"all"`
	SIDECAR_INJECTION string   `json:"sidecar_injection" default:"false"`
	DNS_HOST          string   `json:"dns_host" default:"9.9.9.9"`
	DNS_PORT          string   `json:"dns_port" default:"853"`
	DNS_SPKI          string   `json:"dns_spki"`
	BLOCK_DOMAINS     []string `json:"block_domains"`
}

var CONFIG = "/etc/hdns/hdns.cfg"
var dns_host string
var dns_port string
var dns_spki string
var block_domains []string

func addLengthToQuery(dnsQuery []byte) []byte {
	preLength := []byte{0x00, byte(len(dnsQuery))}
	return append(preLength, dnsQuery...)
}

func sendQuery(tlsConn *tls.Conn, dnsQuery []byte) ([]byte, error) {
	tcpQuery := addLengthToQuery(dnsQuery)
	_, err := tlsConn.Write(tcpQuery)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	result := make([]byte, 1024)
	n, err := tlsConn.Read(result)
	if err != nil {
		fmt.Println(err, 2)
		return nil, err
	}
	return result[:n], nil
}

func dtlsConnect(host, port string) (*tls.Conn, error) {
	config := &tls.Config{ServerName: host}
	conn, err := tls.Dial("tcp", host+":"+port, config)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func sha256Hash(data []byte) []byte {
	hasher := sha256.New()
	hasher.Write(data)
	return hasher.Sum(nil)
}

func verifySPKI(cert *x509.Certificate) bool {
	decodedSPKI, err := base64.StdEncoding.DecodeString(dns_spki)
	if err != nil {
		fmt.Println("Error decoding SPKI:", err)
		return false
	}

	certSPKI := sha256Hash(cert.RawSubjectPublicKeyInfo)
	return bytes.Equal(certSPKI, decodedSPKI)
}

func proxyTcpRequest(requestData []byte) ([]byte, error) {
	return proxyRequest(requestData)
}

func proxyUdpRequest(requestData []byte, udpConn *net.UDPConn, address *net.UDPAddr) {
	responseData, err := proxyRequest(requestData)
	if err != nil {
		fmt.Println("Error proxying DNS request:", err)
		return
	}

	udpResult := responseData[2:]
	_, err = udpConn.WriteToUDP(udpResult, address)
	if err != nil {
		fmt.Println("Error sending UDP response:", err)
		return
	}
	fmt.Println("200")
}

func proxyRequest(requestData []byte) ([]byte, error) {
	dtlsConn, err := dtlsConnect(dns_host, dns_port)
	if err != nil {
		return nil, err
	}
	defer dtlsConn.Close()

	tlsState := dtlsConn.ConnectionState()
	peerCert := tlsState.PeerCertificates[0]

	if !verifySPKI(peerCert) {
		return nil, fmt.Errorf("SPKI verification failed")
	}

	tcpResult, err := sendQuery(dtlsConn, requestData)
	if err != nil {
		fmt.Println("Error sending query:", err)
		return nil, nil
	}

	rcodeHex := hex.EncodeToString(tcpResult[:6])
	rcodeStr := rcodeHex[10:]

	rcode, err := hex.DecodeString(rcodeStr)
	if err != nil {
		fmt.Println("Error decoding rcode:", err)
		return nil, nil
	}

	if len(rcode) > 0 && rcode[0] == 1 {
		fmt.Println("Not a DNS query")
	} else {
		fmt.Println("200")
		return tcpResult, nil
	}
	return nil, nil
}

func main() {
	config := readConfig()
	if config.SIDECAR_INJECTION == "true" {

	}

	//var wg sync.WaitGroup
	switch config.SERVICE_PROTO {
	case "tcp":
		//wg.Add(1)
		go runTCP(config)
	case "udp":
		//go runUDP(config)
	case "all":
		//wg.Add(1)
		go runTCP(config)
		go runUDP(config)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println(err)
	}
	defer watcher.Close()
	err = watcher.Add(CONFIG)
	if err != nil {
		fmt.Println("CONFIG", err)
	}
	for {
		select {
		case event := <-watcher.Events:
			if event.Op == fsnotify.Remove {
				watcher.Remove(event.Name)
				watcher.Add(CONFIG)
				readConfig()
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				readConfig()
			}
		case err := <-watcher.Errors:
			fmt.Println(err)
		}
	}
	//wg.Wait()
}

func runTCP(config Config) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%s", config.SERVICE_HOST, config.SERVICE_PORT))
	tcpListener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		fmt.Println("Error creating TCP listener:", err)
		return
	}
	defer tcpListener.Close()
	fmt.Printf("TCP DNS proxy listening on port %v...\n", config.SERVICE_PORT)

	for {
		clientConn, err := tcpListener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go func(conn net.Conn) {
			defer conn.Close()
			sizeData := make([]byte, 2)
			_, err := conn.Read(sizeData)
			if err != nil {
				fmt.Println("Error reading request:", err)
				return
			}
			size := binary.BigEndian.Uint16(sizeData)
			requestData := make([]byte, size)
			_, err = conn.Read(requestData)
			domain := getDomain(requestData)
			if checkBlock(strings.TrimSuffix(domain, ".")) {
				fmt.Println("Domain Blocked")
				return
			}
			responseData, err := proxyTcpRequest(requestData)
			if err != nil {
				fmt.Println("Error proxying DNS request:", err)
				return
			}
			_, err = conn.Write(responseData)
			if err != nil {
				fmt.Println("Error writing response:", err)
				return
			}
		}(clientConn)
	}
}

func runUDP(config Config) {
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%s", config.SERVICE_HOST, config.SERVICE_PORT))
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println("Error creating UDP listener:", err)
		return
	}
	defer udpConn.Close()

	fmt.Printf("UDP DNS proxy listening on port %v...\n", config.SERVICE_PORT)
	for {
		data := make([]byte, 1024)
		n, addr, err := udpConn.ReadFromUDP(data)
		if err != nil {
			fmt.Println("Error reading from UDP:", err)
			continue
		}
		go proxyUdpRequest(data[:n], udpConn, addr)
	}
}

func readConfig() Config {
	configFile, err := os.Open(CONFIG)
	if err != nil {
		fmt.Println("Error opening configuration file:", err)
		return Config{}
	}
	defer configFile.Close()

	var config Config
	decoder := json.NewDecoder(configFile)
	if err := decoder.Decode(&config); err != nil {
		fmt.Println("Error decoding configuration:", err)
		return Config{}
	}

	dns_host = config.DNS_HOST
	dns_port = config.DNS_PORT
	dns_spki = config.DNS_SPKI
	block_domains = config.BLOCK_DOMAINS
	fmt.Println(dns_host, dns_port, dns_spki)
	return config
}

func getDomain(packet []byte) string {
	offset := 12

	var domain string
	for packet[offset] != 0 {
		labelLength := int(packet[offset])
		if labelLength == 0 {
			break
		}
		offset++

		label := string(packet[offset : offset+labelLength])
		domain += label + "."
		offset += labelLength
	}

	return domain
}

func checkBlock(s string) bool {
	for _, v := range block_domains {
		if v == s {
			return true
		}
	}
	return false
}
