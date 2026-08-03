// Copyright (c) 2023-2026, Nubificus LTD
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package unikernels

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinuxParseCmdLine(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		cmdLine     []string
		expectedApp string
		expectedCmd string
		expectErr   bool
	}{
		{
			name:      "empty cmdline",
			cmdLine:   []string{},
			expectErr: true,
		},
		{
			name:        "single word app only",
			cmdLine:     []string{"/urunit"},
			expectedApp: "/urunit",
			expectedCmd: "",
		},
		{
			name:        "multi-word argument gets quoted",
			cmdLine:     []string{"/urunit", "sh -c ls"},
			expectedApp: "/urunit",
			expectedCmd: "'sh -c ls'",
		},
		{
			name:      "argument with space and single quote is rejected",
			cmdLine:   []string{"/urunit", "sh -c \"echo it's broken\""},
			expectErr: true,
		},
		{
			name:      "argument with single quote and no space is rejected",
			cmdLine:   []string{"/urunit", "it's"},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			l := &Linux{}
			err := l.parseCmdLine(tc.cmdLine)

			if tc.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedApp, l.App)
			assert.Equal(t, tc.expectedCmd, l.Command)
		})
	}
}
