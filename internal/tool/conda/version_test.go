package conda

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCondaVersionOrder(t *testing.T) {
	t.Parallel()
	ordered := []string{
		"0.4",
		"0.4.1.rc",
		"0.4.1",
		"0.5a1",
		"0.5b3",
		"0.5C1",
		"0.5",
		"1.0",
		"1.1dev1",
		"1.1_",
		"1.1a1",
		"1.1.0dev1",
		"1.1.0rc1",
		"1.1",
		"1.1post1",
		"1996.07.12",
		"1!0.4.1",
		"2!0.4.1",
	}
	for i := 1; i < len(ordered); i++ {
		cmp, err := compareVersions(ordered[i-1], ordered[i])
		require.NoError(t, err, ordered[i-1])
		require.Negative(t, cmp, "%s should be older than %s", ordered[i-1], ordered[i])
	}

	equal := [][2]string{
		{"0.4", "0.4.0"},
		{"0.4.1.rc", "0.4.1.RC"},
		{"1.1", "1.1.0"},
		{"1.1.0dev1", "1.1.dev1"},
		{"1.1.0post1", "1.1.post1"},
	}
	for _, pair := range equal {
		cmp, err := compareVersions(pair[0], pair[1])
		require.NoError(t, err)
		require.Zero(t, cmp, "%s vs %s", pair[0], pair[1])
	}
}

func TestVersionPrefixAndSpecs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		spec    string
		version string
		want    bool
	}{
		{spec: "=1.11", version: "1.11.2", want: true},
		{spec: "=1.11", version: "1.12.0", want: false},
		{spec: "=1.11", version: "1.110", want: false},
		{spec: "==1.1", version: "1.1.0", want: true},
		{spec: "==1.1.1", version: "1.1.0", want: false},
		{spec: ">=1.2,<2.0", version: "1.9.0", want: true},
		{spec: ">=1.2,<2.0", version: "2.0.0", want: false},
		{spec: ">=2.17,<3.0.a0", version: "2.28", want: true},
		{spec: ">=2.17,<3.0.a0", version: "3.0", want: false},
		{spec: "1.2.*", version: "1.2.9", want: true},
		{spec: "1.2.*", version: "1.3.0", want: false},
		{spec: "*", version: "9.9.9", want: true},
		{spec: "~=1.4.2", version: "1.4.9", want: true},
		{spec: "~=1.4.2", version: "1.5.0", want: false},
		{spec: "~=1.4.2", version: "1.4.1", want: false},
	}
	for _, tc := range cases {
		ok, err := matchVersion(tc.spec, tc.version)
		require.NoError(t, err, tc.spec)
		require.Equal(t, tc.want, ok, "%s against %s", tc.spec, tc.version)
	}
}

func TestInvalidVersion(t *testing.T) {
	t.Parallel()
	_, err := parseCondaVersion("")
	require.ErrorIs(t, err, ErrInvalidVersion)
	_, err = parseCondaVersion("1.0 with spaces")
	require.ErrorIs(t, err, ErrInvalidVersion)
}
