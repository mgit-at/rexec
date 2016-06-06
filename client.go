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

	cmdJSON, err := json.Marshal(command)
	if err != nil {
		log.Fatal(err)
	}
	rights := syscall.UnixRights(int(os.Stdin.Fd()), int(os.Stdout.Fd()), int(os.Stderr.Fd()))

	if bn, oobn, err := unixConn.WriteMsgUnix(cmdJSON, rights, nil); err != nil {
		log.Fatal(err)
	} else if bn != len(cmdJSON) {
		log.Fatal("Error sending command via unix socket")
	} else if oobn != len(rights) {
		log.Fatal("Error sending file descriptors via unix socket")
	}

	var response CommandResponse
	d := json.NewDecoder(unixConn)
	if err := d.Decode(&response); err != nil {
		log.Fatal(err)
	}

	os.Exit(response.Status)
}
