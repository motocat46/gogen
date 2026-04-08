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
//
// 作者:  yangyuan
// 创建日期: 2026/4/8

package generator_test

import (
	"testing"

	"github.com/motocat46/gogen/pkg/generator"
)

// TestStructGeneratorNames 验证结构体生成器的 Name() 返回值正确，
// 用于错误信息中标识生成器来源。
func TestStructGeneratorNames(t *testing.T) {
	tests := []struct {
		g    generator.StructGenerator
		want string
	}{
		{&generator.ModifyGenerator{}, "modify"},
		{&generator.ResetGenerator{}, "reset"},
	}
	for _, tt := range tests {
		if got := tt.g.Name(); got != tt.want {
			t.Errorf("%T.Name() = %q, want %q", tt.g, got, tt.want)
		}
	}
}
