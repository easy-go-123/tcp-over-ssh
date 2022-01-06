package tcpoverssh

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test1(t *testing.T) {
	cli, err := NewSSHClientEx(SSHClientConfig{
		User:            "root",
		Host:            "39.105.59.84",
		Port:            22,
		Passwords:       nil,
		Keys:            []string{"/Users/rubyist/.ssh/id_rsa"},
		HostKeyCallback: nil,
	})
	assert.Nil(t, err)
	d, err := cli.Output("ls")
	assert.Nil(t, err)
	t.Log(string(d))
}
