package dialog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestItemLabel(t *testing.T) {
	t.Parallel()
	require.Equal(t, "L", ItemLabel(Item{Label: "L", Value: "V"}))
	require.Equal(t, "V", ItemLabel(Item{Value: "V"}))
}

func TestFormatChoiceLines(t *testing.T) {
	t.Parallel()
	items := []Item{
		{Label: "Alpha", Value: "a", Icon: "icon-a"},
		{Value: "b"},
	}
	require.Equal(t, "Alpha\nb\n", FormatChoiceLines(items, false), "without icons")
	require.Equal(t, "Alpha\x00icon\x1ficon-a\nb\n", FormatChoiceLines(items, true), "with icons")
}

func TestMatchSelected(t *testing.T) {
	t.Parallel()
	items := []Item{
		{Label: "Alpha", Value: "a"},
		{Value: "b"},
	}

	require.Nil(t, MatchSelected(items, "   "))

	got := MatchSelected(items, "Alpha")
	require.NotNil(t, got)
	require.Equal(t, "a", got.Value, "match label")
	got = MatchSelected(items, "b")
	require.NotNil(t, got)
	require.Equal(t, "b", got.Value, "match value-as-label")

	got = MatchSelected(items, "other")
	require.NotNil(t, got)
	require.Equal(t, "other", got.Label, "unknown selected")
	require.Equal(t, "other", got.Value, "unknown selected")
}
