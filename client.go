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
	"syscall"
)

// RemoteCommand is the run parameters to be executed remotely
type RemoteCommand struct {
	Cmd     string
	Args    []string
	WorkDir string
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
	if command.WorkDir, err = os.Getwd(); err != nil {
		log.Printf("Warning: unable to get current working directory: %v", err)
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
