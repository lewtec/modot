package modfile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTagSpec(t *testing.T) {
	t.Parallel()

	pattern, ok, err := TagSpec(`tag:^v?(?<version>\d+\.\d+\.\d+)$`)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, `^v?(?<version>\d+\.\d+\.\d+)$`, pattern)

	_, ok, err = TagSpec("4.0.0")
	require.NoError(t, err)
	require.False(t, ok)

	_, ok, err = TagSpec("HEAD")
	require.NoError(t, err)
	require.False(t, ok)

	_, _, err = TagSpec("tag:")
	require.ErrorIs(t, err, ErrEmptyTagPattern)

	_, _, err = TagSpec("tag:(")
	require.ErrorIs(t, err, ErrInvalidTagPattern)

	pattern, ok, err = TagSpec("  tag:^v1$")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "^v1$", pattern)

	_, ok, err = TagSpec(`tag:^v?(?P<version>\d+\.\d+\.\d+)$`)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestSelectNewestTag(t *testing.T) {
	t.Parallel()

	got, err := SelectNewestTag(
		[]string{"1.0.0", "1.2.0", "1.10.0", "main", "v2.0.0-rc.1", "v2.0.0"},
		`^v?(?<version>\d+\.\d+\.\d+)$`,
	)
	require.NoError(t, err)
	require.Equal(t, "v2.0.0", got)

	got, err = SelectNewestTag(
		[]string{"go1.9.0", "go1.27.0", "v1.2.3"},
		`^go(?<version>\d+\.\d+\.\d+)$`,
	)
	require.NoError(t, err)
	require.Equal(t, "go1.27.0", got)

	got, err = SelectNewestTag(
		[]string{"v1.9.0", "v1.10.0"},
		`^v(?<major>\d+)\.(?<minor>\d+)\.(?<patch>\d+)$`,
	)
	require.NoError(t, err)
	require.Equal(t, "v1.10.0", got)

	got, err = SelectNewestTag(
		[]string{"v2.0.0-rc.1", "v2.0.0", "v1.9.0"},
		`^v?(?<version>\d+\.\d+\.\d+(?:-[\w.]+)?)$`,
	)
	require.NoError(t, err)
	require.Equal(t, "v2.0.0", got)

	got, err = SelectNewestTag(
		[]string{"v1.0.0-rc.1", "v1.0.0-rc.2"},
		`^v?(?<version>\d+\.\d+\.\d+-[\w.]+)$`,
	)
	require.NoError(t, err)
	require.Equal(t, "v1.0.0-rc.2", got)

	_, err = SelectNewestTag([]string{"main"}, `^v?(?<version>\d+\.\d+\.\d+)$`)
	require.ErrorIs(t, err, ErrNoMatchingTag)

	_, err = SelectNewestTag(nil, "")
	require.ErrorIs(t, err, ErrEmptyTagPattern)

	_, err = SelectNewestTag([]string{"v1"}, "(")
	require.ErrorIs(t, err, ErrInvalidTagPattern)
}
