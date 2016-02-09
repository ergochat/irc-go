package gircclient

import (
	"bufio"
	"net"
	"testing"
)

func TestServerConnection(t *testing.T) {
	reactor := NewReactor()
	newServer := reactor.CreateServer("local")

	newServer.Nick = "coolguy"
	newServer.InitialUser = "c"
	newServer.InitialRealName = "girc-go Test Client  "

	// we mock up a server connection to test the client
	listener, _ := net.Listen("tcp", ":0")

	newServer.Connect(listener.Addr().String(), false)

	conn, _ := listener.Accept()
	reader := bufio.NewReader(conn)

	// test each message in sequence
	var message string

	message, _ = reader.ReadString('\n')
	if message != "NICK coolguy\r\n" {
		t.Error(
			"Did not receive NICK message, received: [",
			message,
			"]",
		)
	}

	message, _ = reader.ReadString('\n')
	if message != "USER c 0 * :girc-go Test Client  \r\n" {
		t.Error(
			"Did not receive USER message, received: [",
			message,
			"]",
		)
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
	}

	// close connection and listener
	conn.Close()
	listener.Close()
}
