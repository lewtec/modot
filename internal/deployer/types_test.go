package deployer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSortActions(t *testing.T) {
	actions := []Action{
		{Type: ActionCreate, Target: "/b"},
		{Type: ActionDelete, Target: "/a"},
		{Type: ActionNoop, Target: "/c"},
		{Type: ActionUpdate, Target: "/d"},
		{Type: ActionCreate, Target: "/a"},
		{Type: ActionDelete, Target: "/b"},
	}

	sorted := SortActions(actions)

	// Verify order: Delete < Update < Create < Noop, then by Target
	expected := []struct {
		actionType ActionType
		target     string
	}{
		{ActionDelete, "/a"},
		{ActionDelete, "/b"},
		{ActionUpdate, "/d"},
		{ActionCreate, "/a"},
		{ActionCreate, "/b"},
		{ActionNoop, "/c"},
	}

	require.Len(t, sorted, len(expected))

	for i, want := range expected {
		assert.Equal(t, want.actionType, sorted[i].Type, "sorted[%d] type", i)
		assert.Equal(t, want.target, sorted[i].Target, "sorted[%d] target", i)
	}

	// Verify original slice is not modified
	assert.Equal(t, ActionCreate, actions[0].Type, "SortActions mutated the original slice")
	assert.Equal(t, "/b", actions[0].Target, "SortActions mutated the original slice")
}

func TestActionTypeString(t *testing.T) {
	tests := []struct {
		action ActionType
		want   string
	}{
		{ActionCreate, "+"},
		{ActionUpdate, "*"},
		{ActionDelete, "-"},
		{ActionNoop, " "},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.action.String()
			assert.Equal(t, tt.want, got, "ActionType(%d).String()", tt.action)
		})
	}
}
