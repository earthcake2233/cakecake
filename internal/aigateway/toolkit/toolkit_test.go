package toolkit

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestS(t *testing.T) {
	m := S("a string property")
	require.Equal(t, "string", m["type"])
	require.Equal(t, "a string property", m["description"])
}

func TestI(t *testing.T) {
	m := I("an integer")
	require.Equal(t, "integer", m["type"])
	require.Equal(t, "an integer", m["description"])
}

func TestObject_NoRequired(t *testing.T) {
	props := map[string]interface{}{
		"name": S("Name"),
		"age":  I("Age"),
	}
	m := Object(props)
	require.Equal(t, "object", m["type"])
	require.Equal(t, props, m["properties"])
	_, hasRequired := m["required"]
	require.False(t, hasRequired)
}

func TestObject_WithRequired(t *testing.T) {
	props := map[string]interface{}{"name": S("Name")}
	m := Object(props, "name")
	require.Equal(t, "object", m["type"])
	require.Equal(t, []string{"name"}, m["required"])
}

func TestObject_EmptyProps(t *testing.T) {
	m := Object(map[string]interface{}{})
	require.Equal(t, "object", m["type"])
	require.Empty(t, m["properties"])
}

func TestAllToolNames(t *testing.T) {
	names := AllToolNames()
	require.Contains(t, names, ToolSearchVideos)
	require.Contains(t, names, ToolGetVideoDetail)
	require.Contains(t, names, ToolGetTrending)
	require.Contains(t, names, ToolGetVideoComments)
	require.Contains(t, names, ToolGetVideoDanmaku)
	require.Len(t, names, 5)
}

func TestDefaultEnabled(t *testing.T) {
	m := defaultEnabled()
	for _, name := range AllToolNames() {
		require.True(t, m[name], "expected %s to be enabled by default", name)
	}
}

func TestDefineTools_NilEnabled(t *testing.T) {
	tools := DefineTools(nil)
	require.Len(t, tools, 5)
	for _, td := range tools {
		require.Equal(t, "function", td.Type)
		require.NotEmpty(t, td.Function.Name)
		require.NotEmpty(t, td.Function.Description)
		require.NotNil(t, td.Function.Parameters)
	}
}

func TestDefineTools_SomeEnabled(t *testing.T) {
	enabled := map[string]bool{
		ToolSearchVideos: true,
		ToolGetTrending:  false,
	}
	tools := DefineTools(enabled)
	require.Len(t, tools, 1)
	require.Equal(t, ToolSearchVideos, tools[0].Function.Name)
}

func TestDefineTools_AllDisabled(t *testing.T) {
	enabled := map[string]bool{}
	tools := DefineTools(enabled)
	require.Empty(t, tools)
}

// -- truncateStr --

func TestTruncateStr_Short(t *testing.T) {
	result := truncateStr("hello", 10)
	require.Equal(t, "hello", result)
}

func TestTruncateStr_Exact(t *testing.T) {
	result := truncateStr("hello", 5)
	require.Equal(t, "hello", result)
}

func TestTruncateStr_Long(t *testing.T) {
	result := truncateStr("hello world", 5)
	require.Equal(t, "hello...", result)
}

func TestTruncateStr_Empty(t *testing.T) {
	require.Equal(t, "", truncateStr("", 5))
}
