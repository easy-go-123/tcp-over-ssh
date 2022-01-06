package tcpoverssh

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTCPProxy1(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	proxy := NewTCPFixProxyOverSSH(ctx, ":9877", "r-2zey9uf66uyu7ftg2u.redis.rds.aliyuncs.com:6379", SSHClientConfig{
		User: "root",
		Host: "39.105.59.84",
		Port: 22,
		Keys: []string{"/Users/rubyist/.ssh/id_rsa"},
	})
	assert.NotNil(t, proxy)

	proxy.Wait()
}
