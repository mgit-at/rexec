package main

import (
	"io"
	"log"
	"net"
	"os/exec"
	"syscall"

	"github.com/coreos/go-systemd/activation"
	"github.com/docker/libchan"
	"github.com/docker/libchan/spdy"
)

// RemoteCommand is the received command parameters to execute locally and return
type RemoteCommand struct {
	Cmd        string
	Args       []string
	Stdin      io.Reader
	Stdout     io.WriteCloser
	Stderr     io.WriteCloser
	StatusChan libchan.Sender
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

	p, err := spdy.NewSpdyStreamProvider(conn, true)
	if err != nil {
		log.Fatal(err)
	}
	t := spdy.NewTransport(p)

	receiver, err := t.WaitReceiveChannel()
	if err != nil {
		log.Fatal(err)
	}

	command := &RemoteCommand{}
	err = receiver.Receive(command)
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command(command.Cmd, command.Args...)
	cmd.Stdout = command.Stdout
	cmd.Stderr = command.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		io.Copy(stdin, command.Stdin)
		stdin.Close()
	}()

	res := cmd.Run()
	command.Stdout.Close()
	command.Stderr.Close()
	returnResult := &CommandResponse{}
	if res != nil {
		if exiterr, ok := res.(*exec.ExitError); ok {
			returnResult.Status = exiterr.Sys().(syscall.WaitStatus).ExitStatus()
		} else {
			log.Print(res)
			returnResult.Status = 10
		}
	}

	err = command.StatusChan.Send(returnResult)
	if err != nil {
		log.Print(err)
	}
}
