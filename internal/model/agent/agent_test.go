package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseWelcomeMessages(t *testing.T) {
	require.Nil(t, ParseWelcomeMessages(""))
	require.Nil(t, ParseWelcomeMessages("   "))
	require.Nil(t, ParseWelcomeMessages("not-json"))

	got := ParseWelcomeMessages(`["hi", " ", "  hello  "]`)
	require.Equal(t, []string{"hi", "hello"}, got)
	require.Empty(t, ParseWelcomeMessages(`["", "  "]`))
}

func TestEncodeWelcomeMessages(t *testing.T) {
	require.Equal(t, "[]", EncodeWelcomeMessages(nil))
	require.Equal(t, "[]", EncodeWelcomeMessages([]string{"", "  "}))
	require.Equal(t, `["a","b"]`, EncodeWelcomeMessages([]string{"a", " b "}))
}
