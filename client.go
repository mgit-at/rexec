package main

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"syscall"
)

// RemoteCommand is the run parameters to be executed remotely
type RemoteCommand struct {
	Cmd  string
	Args []string
}

// CommandResponse is the returned response object from the remote execution
type CommandResponse struct {
	Status int
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("usage: <socket> <command> [ <args>... ]")
	}

	conn, err := net.Dial("unix", os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	unixConn := conn.(*net.UnixConn)

	command := &RemoteCommand{
		Cmd:  os.Args[2],
		Args: os.Args[3:],
	}

	rights := syscall.UnixRights(int(os.Stdin.Fd()), int(os.Stdout.Fd()), int(os.Stderr.Fd()))
	if _, oobn, err := unixConn.WriteMsgUnix([]byte{0}, rights, nil); err != nil { // send at least 1 byte of payload
		log.Fatal(err)
	} else if oobn != len(rights) {
		log.Fatal("Error sending file descriptors via unix socket failed: short write")
	}

	e := json.NewEncoder(unixConn)
	if err := e.Encode(command); err != nil {
		log.Fatal("encoding/sending command: ", err)
	}

	var response CommandResponse
	d := json.NewDecoder(unixConn)
	if err := d.Decode(&response); err != nil {
		log.Fatal("receiving/decoding result: ", err)
	}

	os.Exit(response.Status)
}
