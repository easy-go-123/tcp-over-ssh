package main

import (
	"context"
	"flag"

	"github.com/easy-go-123/tcp-over-ssh/tcpoverssh"
)

func main() {
	listen := flag.String("listen", ":11111", "listen")
	intranet := flag.String("intranet", "", "intranet")
	sshHost := flag.String("ssh_host", "", "ssh_host")
	sshPort := flag.Int("ssh_port", 0, "ssh_port")
	sshUser := flag.String("ssh_user", "", "ssh_user")
	sshKey := flag.String("ssh_key", "", "ssh_key")
	flag.Parse()

	if *listen == "" || *intranet == "" || *sshUser == "" || *sshHost == "" ||
		*sshPort == 0 || *sshKey == "" {
		panic("no input")
	}

	proxy := tcpoverssh.NewTCPFixProxyOverSSH(context.Background(), *listen, *intranet, tcpoverssh.SSHClientConfig{
		User: *sshUser,
		Host: *sshHost,
		Port: *sshPort,
		Keys: []string{*sshKey},
	})
	if proxy == nil {
		panic("createProxy")
	}

	proxy.Wait()
}
