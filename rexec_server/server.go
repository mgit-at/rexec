package main

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"os/exec"
	"syscall"

	"github.com/coreos/go-systemd/activation"
)

// RemoteCommand is the received command parameters to execute locally and return
type RemoteCommand struct {
	Cmd  string
	Args []string
}

// CommandResponse is the response struct to return to the client
type CommandResponse struct {
	Status int
}

func main() {
	files := activation.Files(true)
	if len(files) != 1 {
		log.Fatal("invalid number of sockets passed from systemd")
	}

	conn, err := net.FileConn(files[0])
	if err != nil {
		log.Fatal(err)
	}

	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		log.Fatalf("got invalid socket type from systemd: %T\n", conn)
	}

	b := make([]byte, 8192)  // TODO: hardcoded value...
	oob := make([]byte, 128) // TODO: hardcoded value...
	bn, oobn, _, _, err := unixConn.ReadMsgUnix(b, oob)

	var command RemoteCommand
	if err := json.Unmarshal(b[:bn], &command); err != nil {
		log.Fatal(err)
	}

	scms, err := syscall.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		log.Fatal(err)
	}
	if len(scms) != 1 {
		log.Fatalf("expected 1 SocketControlMessage, got %d", len(scms))
	}

	scm := scms[0]
	fds, err := syscall.ParseUnixRights(&scm)
	if err != nil {
		log.Fatal(err)
	}
	if len(fds) != 3 {
		log.Fatalf("wanted 3 file descriptors, got %d", len(fds))
	}

	cmd := exec.Command(command.Cmd, command.Args...)
	cmd.Stdin = os.NewFile(uintptr(fds[0]), "stdin")
	cmd.Stdout = os.NewFile(uintptr(fds[1]), "stdout")
	cmd.Stderr = os.NewFile(uintptr(fds[2]), "stderr")

	result := &CommandResponse{}
	if res := cmd.Run(); res != nil {
		if exiterr, ok := res.(*exec.ExitError); ok {
			result.Status = exiterr.Sys().(syscall.WaitStatus).ExitStatus()
		} else {
			log.Print(res)
			result.Status = 10
		}
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		log.Fatal(err)
	}

	if n, err := unixConn.Write(resultJSON); err != nil {
		log.Fatal(err)
	} else if n != len(resultJSON) {
		log.Fatal(err)
	}
}
