package queue

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConsumeTranscodeDead_NilChannel(t *testing.T) {
	c := &Client{}
	_, err := c.ConsumeTranscodeDead("tag")
	require.Error(t, err)
}

func TestNewTranscodeDeadConsumer_NilConn(t *testing.T) {
	c := &Client{}
	_, _, err := c.NewTranscodeDeadConsumer("tag")
	require.Error(t, err)
}
