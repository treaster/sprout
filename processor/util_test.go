package processor_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/treaster/sprout/processor"
)

func TestSafeCutPrefix(t *testing.T) {
	testCases := []struct {
		inputS         string
		inputPrefix    string
		expectedOutput string
		expectPanic    bool
	}{
		{"./abc/def", "./abc", "def", false},
		{"./abc/def", "./abc/", "def", false},
		{"./abc/def", "abc/", "def", false},

		{"/abc/def", "/abc", "def", false},
		{"/abc/def", "/abc/", "def", false},

		{"./abc/def", "/abc/", "", true},
		{"abc/def", "/abc/", "", true},
		{"/abc/def", "abc/", "", true},
		{"/abc/def", "./abc/", "", true},
	}

	for testI, testCase := range testCases {
		if testCase.expectPanic {
			require.Panics(t, func() {
				_ = processor.SafeCutPrefix(testCase.inputS, testCase.inputPrefix)
			})
		} else {
			output := processor.SafeCutPrefix(testCase.inputS, testCase.inputPrefix)
			require.Equal(t, testCase.expectedOutput, output, "Test case %d (%q, %q)", testI, testCase.inputS, testCase.inputPrefix)
		}
	}
}
