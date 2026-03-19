// 版权所有(Copyright)[yangyuan]
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package analyzer

import "testing"

func TestParseStructAnnotations(t *testing.T) {
	cases := []struct {
		name   string
		doc    string
		wantP  bool
		wantDM string
		wantND bool
	}{
		{"空", "", false, "", false},
		{"plain", "gogen:plain", true, "", false},
		{"dirty 默认", "gogen:dirty", false, "MakeDirty", false},
		{"dirty=自定义", "gogen:dirty=CustomMethod", false, "CustomMethod", false},
		{"nodirty", "gogen:nodirty", false, "", true},
		{"plain+dirty 同时", "gogen:plain\ngogen:dirty", true, "MakeDirty", false},
		{"dirty+nodirty 同时（nodirty 不取消 DirtyMethod 字段，由 analyzeFile 决策）",
			"gogen:dirty\ngogen:nodirty", false, "MakeDirty", true},
		{"dirty= 空值忽略", "gogen:dirty=", false, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseStructAnnotations(tc.doc)
			if got.Plain != tc.wantP {
				t.Errorf("Plain = %v, want %v", got.Plain, tc.wantP)
			}
			if got.DirtyMethod != tc.wantDM {
				t.Errorf("DirtyMethod = %q, want %q", got.DirtyMethod, tc.wantDM)
			}
			if got.NoDirty != tc.wantND {
				t.Errorf("NoDirty = %v, want %v", got.NoDirty, tc.wantND)
			}
		})
	}
}
