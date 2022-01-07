package tcpoverssh

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"

	"golang.org/x/crypto/ssh"
)

var (
	ErrInvalidArgs = errors.New("invalidArgs")
)

type SSHClient interface {
	Dial(network, addr string) (conn net.Conn, err error)
	Output(cmd string) ([]byte, error)
}

type SSHClientConfig struct {
	User            string
	Host            string
	Port            int
	Passwords       []string
	Keys            []string
	HostKeyCallback ssh.HostKeyCallback
}

func NewSSHClientEx(d SSHClientConfig) (SSHClient, error) {
	authMethods := make([]ssh.AuthMethod, 0)
	for _, password := range d.Passwords {
		authMethods = append(authMethods, ssh.Password(password))
	}

	for _, key := range d.Keys {
		keyD, err := ioutil.ReadFile(key)
		if err != nil {
			continue
		}

		rpk, err := ssh.ParsePrivateKey(keyD)
		if err != nil {
			continue
		}

		authMethods = append(authMethods, ssh.PublicKeys(rpk))
	}

	hostKeyCallback := d.HostKeyCallback
	if hostKeyCallback == nil {
		// nolint: gosec
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	cfg := &ssh.ClientConfig{
		User:            d.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
	}

	return &sshClientImpl{
		addr: fmt.Sprintf("%s:%d", d.Host, d.Port),
		cfg:  cfg,
	}, nil
}

type sshClientImpl struct {
	addr string
	cfg  *ssh.ClientConfig
}

func (impl *sshClientImpl) Dial(network, addr string) (conn net.Conn, err error) {
	sshConn, err := impl.newConn()
	if err != nil {
		return
	}

	return sshConn.Dial(network, addr)
}

func (impl *sshClientImpl) newConn() (*ssh.Client, error) {
	return ssh.Dial("tcp", impl.addr, impl.cfg)
}

func (impl *sshClientImpl) withConn(do func(conn *ssh.Client, err error) error) error {
	if do == nil {
		return ErrInvalidArgs
	}

	cli, err := impl.newConn()
	ret := do(cli, err)

	if err == nil {
		_ = cli.Close()
	}

	return ret
}

func (impl *sshClientImpl) withSession(do func(session *ssh.Session, err error) error) error {
	if do == nil {
		return ErrInvalidArgs
	}

	return impl.withConn(func(conn *ssh.Client, err error) error {
		if err != nil {
			return err
		}

		session, err := conn.NewSession()
		ret := do(session, err)
		if err == nil {
			_ = session.Close()
		}

		return ret
	})
}

func (impl *sshClientImpl) Output(cmd string) (output []byte, err error) {
	err = impl.withSession(func(session *ssh.Session, err error) error {
		if err != nil {
			return err
		}

		modes := ssh.TerminalModes{
			ssh.ECHO:          0,
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
		}
		err = session.RequestPty("xterm", 25, 80, modes)
		if err != nil {
			return err
		}

		output, err = session.CombinedOutput(cmd)

		return err
	})

	return
}
