//
// Copyright: 2016, mgIT GmbH <office@mgit.at>
// All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

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

	b := make([]byte, 1)     // the client sends the OOB message together with one dummy byte
	oob := make([]byte, 128) // OOB should be 32 bytes
	_, oobn, _, _, err := unixConn.ReadMsgUnix(b, oob)
	if err != nil {
		log.Fatal("ReadMsgUnix(): ", err)
	}

	scms, err := syscall.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		log.Fatal("ParseSocketControlMessage(): ", err)
	}
	if len(scms) != 1 {
		log.Fatalf("expected 1 SocketControlMessage, got %d", len(scms))
	}

	scm := scms[0]
	fds, err := syscall.ParseUnixRights(&scm)
	if err != nil {
		log.Fatal("ParseUnixRights: ", err)
	}
	if len(fds) != 3 {
		log.Fatalf("wanted 3 file descriptors, got %d", len(fds))
	}

	var command RemoteCommand
	d := json.NewDecoder(unixConn)
	if err := d.Decode(&command); err != nil {
		log.Fatal("receiving/decoding command: ", err)
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

	e := json.NewEncoder(unixConn)
	if err := e.Encode(result); err != nil {
		log.Fatal("encoding/sending result: ", err)
	}
}
