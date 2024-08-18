// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2024.08.

// This file includes software developed by the gorilla/websocket project.
// Copyright (c) 2010 The Gorilla WebSocket Authors. All rights reserved.
//
// See the BSD-3-Clause license for more details.

package adminweb

import (
	"io"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/ssh"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func HandleWebSocket(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	ws.SetCloseHandler(func(code int, text string) error {
		cblog.Errorf("WebSocket closed with code: %d, text: %s\n", code, text)
		return nil
	})

	user := c.QueryParam("user")
	ip := c.QueryParam("ip")
	privateKey := c.QueryParam("privatekey")

	sshClient, err := connectToSSH(user, ip, privateKey)
	if err != nil {
		cblog.Error("Failed to connect to SSH:", err)
		ws.WriteMessage(websocket.TextMessage, []byte("Failed to connect to SSH: "+err.Error()))
		ws.Close()
		return nil
	}
	defer sshClient.Close()

	session, err := sshClient.NewSession()
	if err != nil {
		cblog.Error("Failed to create SSH session:", err)
		return nil
	}
	defer session.Close()

	sessionStdout, err := session.StdoutPipe()
	if err != nil {
		cblog.Error("Failed to pipe stdout:", err)
		return nil
	}
	sessionStdin, err := session.StdinPipe()
	if err != nil {
		cblog.Error("Failed to pipe stdin:", err)
		return nil
	}

	// Request a PTY with "xterm-256color" and some terminal modes
	if err := session.RequestPty("xterm-256color", 80, 24, ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}); err != nil {
		cblog.Error("Failed to request PTY:", err)
		return nil
	}

	// Start the shell
	if err := session.Shell(); err != nil {
		log.Println("Failed to start shell:", err)
		return nil
	}

	_, err = sessionStdin.Write([]byte("stty cols 80 rows 24\n"))
	if err != nil {
		cblog.Error("Failed to send stty command:", err)
		return nil
	}

	done := make(chan struct{})
	sshDone := make(chan error, 1) // Buffer size 1 to avoid blocking

	go func() {
		defer close(done)
		for {
			_, message, err := ws.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					cblog.Error("WebSocket read error:", err)
				}
				return
			}
			if _, err := sessionStdin.Write(message); err != nil {
				cblog.Error("Failed to write to SSH stdin:", err)
				return
			}
		}
	}()

	go func() {
		buf := make([]byte, 1024*1024)
		for {
			n, err := sessionStdout.Read(buf)
			if err != nil {
				if err != io.EOF {
					cblog.Error("Failed to read from SSH stdout:", err)
				}
				return
			}
			if err := ws.WriteMessage(websocket.TextMessage, buf[:n]); err != nil {
				cblog.Error("WebSocket write error:", err)
				return
			}
		}
	}()

	go func() {
		sshDone <- session.Wait() // Signal when SSH session is done
	}()

	select {
	case <-done: // WebSocket closed
		session.Close()
		return nil
	case err := <-sshDone: // SSH session finished
		if err != nil {
			cblog.Error("SSH session ended with error:", err)
		}
		ws.Close()
		return nil
	}
}

func connectToSSH(user, ip, privateKey string) (*ssh.Client, error) {
	key, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", ip+":22", config)
	if err != nil {
		return nil, err
	}

	return client, nil
}
