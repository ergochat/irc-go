package gircclient

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"runtime"
	"testing"
	"time"

	"github.com/goshuirc/irc-go/ircmap"
	"github.com/goshuirc/irc-go/ircmsg"
)

func TestPlainConnection(t *testing.T) {
	reactor := NewReactor()
	client := reactor.CreateServer("local")

	initialiseServerConnection(client)

	// we mock up a server connection to test the client
	listener, _ := net.Listen("tcp", ":0")

	client.Connect(listener.Addr().String(), false, nil)
	go client.ReceiveLoop()

	testServerConnection(t, reactor, client, listener)
}

func TestFailingConnection(t *testing.T) {
	reactor := NewReactor()
	client := reactor.CreateServer("local")

	// we mock up a server connection to test the client
	listener, _ := net.Listen("tcp", ":0")

	// Try to connect before setting InitialNick and InitialUser
	err := client.Connect(listener.Addr().String(), false, nil)

	if err == nil {
		t.Error(
			"ServerConnection allowed connection before InitialNick and InitialUser were set",
		)
	}

	// Actually set attributes and fail properly this time
	client.InitialNick = "test"
	client.InitialUser = "t"
	client.Connect("here is a malformed address:6667", false, nil)

	if err == nil {
		t.Error(
			"ServerConnection allowed connection with a blatently malformed address",
		)
	}
}

func TestTLSConnection(t *testing.T) {
	reactor := NewReactor()
	client := reactor.CreateServer("local")

	initialiseServerConnection(client)

	// generate a test certificate to use
	priv, _ := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)

	notBefore := time.Now().Add(-1 * time.Hour * 30) // valid 30 hours ago
	notAfter := notBefore.Add(time.Hour * 90)        // for 90 hours

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"gIRC-Go Co"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA: true,
	}

	template.IPAddresses = append(template.IPAddresses, net.ParseIP("127.0.0.1"))
	template.IPAddresses = append(template.IPAddresses, net.ParseIP("::"))
	template.DNSNames = append(template.DNSNames, "localhost")

	derBytes, _ := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)

	c := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	b, _ := x509.MarshalECPrivateKey(priv)
	k := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b})

	// we mock up a server connection to test the client
	listenerKeyPair, _ := tls.X509KeyPair(c, k)

	var listenerTLSConfig tls.Config
	listenerTLSConfig.Certificates = make([]tls.Certificate, 0)
	listenerTLSConfig.Certificates = append(listenerTLSConfig.Certificates, listenerKeyPair)
	listener, _ := tls.Listen("tcp", ":0", &listenerTLSConfig)

	// mock up the client side too
	clientTLSCertPool := x509.NewCertPool()
	clientTLSCertPool.AppendCertsFromPEM(c)

	var clientTLSConfig tls.Config
	clientTLSConfig.RootCAs = clientTLSCertPool
	clientTLSConfig.ServerName = "localhost"
	go client.Connect(listener.Addr().String(), true, &clientTLSConfig)
	go client.ReceiveLoop()

	testServerConnection(t, reactor, client, listener)
}

func sendMessage(conn net.Conn, tags *map[string]ircmsg.TagValue, prefix string, command string, params ...string) {
	ircmsg := ircmsg.MakeMessage(tags, prefix, command, params...)
	line, err := ircmsg.Line()
	if err != nil {
		return
	}
	fmt.Fprintf(conn, line)

	// need to wait for a quick moment here for TLS to process any changes this
	// message has caused
	runtime.Gosched()
	time.Sleep(10 * time.Millisecond)
}

func initialiseServerConnection(client *ServerConnection) {
	client.InitialNick = "coolguy"
	client.InitialUser = "c"
	client.InitialRealName = "girc-go Test Client  "
}

func testServerConnection(t *testing.T, reactor Reactor, client *ServerConnection, listener net.Listener) {
	// start our reader
	conn, _ := listener.Accept()
	reader := bufio.NewReader(conn)

	var message string

	// CAP
	message, _ = reader.ReadString('\n')
	if message != "CAP LS 302\r\n" {
		t.Error(
			"Did not receive CAP LS message, received: [",
			message,
			"]",
		)
		return
	}

	sendMessage(conn, nil, "example.com", "CAP", "*", "LS", "*", "multi-prefix userhost-in-names")
	sendMessage(conn, nil, "example.com", "CAP", "*", "LS", "chghost")

	message, _ = reader.ReadString('\n')
	if message != "CAP REQ :chghost multi-prefix userhost-in-names\r\n" {
		t.Error(
			"Did not receive CAP REQ message, received: [",
			message,
			"]",
		)
		return
	}

	// these should be silently ignored
	fmt.Fprintf(conn, "\r\n\r\n\r\n")

	sendMessage(conn, nil, "example.com", "CAP", "*", "ACK", "chghost multi-prefix userhost-in-names")

	message, _ = reader.ReadString('\n')
	if message != "CAP END\r\n" {
		t.Error(
			"Did not receive CAP END message, received: [",
			message,
			"]",
		)
		return
	}

	// NICK/USER
	message, _ = reader.ReadString('\n')
	if message != "NICK coolguy\r\n" {
		t.Error(
			"Did not receive NICK message, received: [",
			message,
			"]",
		)
		return
	}

	message, _ = reader.ReadString('\n')
	if message != "USER c 0 * :girc-go Test Client  \r\n" {
		t.Error(
			"Did not receive USER message, received: [",
			message,
			"]",
		)
		return
	}

	// make sure nick changes properly
	sendMessage(conn, nil, "example.com", "001", "dan", "Welcome to the gIRC-Go Test Network!")

	if client.Nick != "dan" {
		t.Error(
			"Nick was not set with 001, expected",
			"dan",
			"got",
			client.Nick,
		)
		return
	}

	// send 002/003/004
	sendMessage(conn, nil, "example.com", "002", "dan", "Your host is example.com, running version latest")
	sendMessage(conn, nil, "example.com", "003", "dan", "This server was created almost no time ago!")
	sendMessage(conn, nil, "example.com", "004", "dan", "example.com", "latest", "r", "b", "b")

	// make sure LINELEN gets set correctly
	sendMessage(conn, nil, "example.com", "005", "dan", "LINELEN=", "are available on this server")

	if client.Features["LINELEN"].(int) != 512 {
		t.Error(
			"LINELEN default was not set with 005, expected",
			512,
			"got",
			client.Features["LINELEN"],
		)
		return
	}

	// make sure casemapping and other ISUPPORT values are set properly
	sendMessage(conn, nil, "example.com", "005", "dan", "CASEMAPPING=rfc3454", "NICKLEN=27", "USERLEN=", "SAFELIST", "are available on this server")

	if client.Casemapping != ircmap.RFC3454 {
		t.Error(
			"Casemapping was not set with 005, expected",
			ircmap.RFC3454,
			"got",
			client.Casemapping,
		)
		return
	}

	if client.Features["NICKLEN"].(int) != 27 {
		t.Error(
			"NICKLEN was not set with 005, expected",
			27,
			"got",
			client.Features["NICKLEN"],
		)
		return
	}

	if client.Features["USERLEN"] != nil {
		t.Error(
			"USERLEN was not set with 005, expected",
			nil,
			"got",
			client.Features["USERLEN"],
		)
		return
	}

	if client.Features["SAFELIST"].(bool) != true {
		t.Error(
			"SAFELIST was not set with 005, expected",
			true,
			"got",
			client.Features["SAFELIST"],
		)
		return
	}

	// test PING
	sendMessage(conn, nil, "example.com", "PING", "3847362")

	message, _ = reader.ReadString('\n')
	if message != "PONG 3847362\r\n" {
		t.Error(
			"Did not receive PONG message, received: [",
			message,
			"]",
		)
		return
	}

	// test CAP NEW
	sendMessage(conn, nil, "example.com", "CAP", client.Nick, "NEW", "sasl=plain")

	message, _ = reader.ReadString('\n')
	if message != "CAP REQ sasl\r\n" {
		t.Error(
			"Did not receive CAP REQ sasl message, received: [",
			message,
			"]",
		)
		return
	}

	sendMessage(conn, nil, "example.com", "CAP", client.Nick, "ACK", "sasl")

	sendMessage(conn, nil, "example.com", "CAP", client.Nick, "DEL", "sasl")

	_, exists := client.Caps.Available["sasl"]
	if exists {
		t.Error(
			"SASL cap is still available on client after CAP DEL sasl",
		)
	}

	_, exists = client.Caps.Enabled["sasl"]
	if exists {
		t.Error(
			"SASL cap still enabled on client after CAP DEL sasl",
		)
	}

	sendMessage(conn, nil, "example.com", "CAP", client.Nick, "ACK", "-chghost")

	_, exists = client.Caps.Enabled["chghost"]
	if exists {
		t.Error(
			"chghost cap still enabled on client after ACK -chghost",
		)
	}

	// test actions
	client.Msg(nil, "coalguys", "Isn't this such an $bamazing$r day?!", true)

	message, _ = reader.ReadString('\n')
	if message != "PRIVMSG coalguys :Isn't this such an \x02amazing\x0f day?!\r\n" {
		t.Error(
			"Did not receive PRIVMSG message, received: [",
			message,
			"]",
		)
		return
	}

	client.Notice(nil, "coalguys", "Isn't this such a $c[red]great$c day?", true)

	message, _ = reader.ReadString('\n')
	if message != "NOTICE coalguys :Isn't this such a \x034great\x03 day?\r\n" {
		t.Error(
			"Did not receive NOTICE message, received: [",
			message,
			"]",
		)
		return
	}

	// test casefolding
	target, _ := client.Casefold("#beßtchannEL")
	if target != "#besstchannel" {
		t.Error(
			"Channel name was not casefolded correctly, expected",
			"#besstchannel",
			"got",
			target,
		)
		return
	}

	// shutdown client
	reactor.Shutdown(" Get mad!  ")

	message, _ = reader.ReadString('\n')
	if message != "QUIT : Get mad!  \r\n" {
		t.Error(
			"Did not receive QUIT message, received: [",
			message,
			"]",
		)
		return
	}

	// test malformed Send
	err := client.Send(nil, "", "PRIVMSG", "MyFriend", "", "param with spaces", "Hey man!")
	if err == nil {
		t.Error(
			"ServerConnection allowed a Send with empty and params with spaces before the last param",
		)
	}

	// close connection and listener
	conn.Close()
	listener.Close()
}
