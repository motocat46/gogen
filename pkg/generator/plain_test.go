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
// 创建日期: 2025/7/31

package generator_test

import (
	"bytes"
	"testing"

	"github.com/motocat46/gogen/pkg/analyzer"
	"github.com/motocat46/gogen/pkg/generator"
	"github.com/motocat46/gogen/pkg/loader"
)

// TestPlainMode 验证 gogen:"plain" tag 对各类型的生成行为：
//   - plain 模式只保留核心 Get/Set，跳过 Add/Sub/Toggle/Has 等扩展方法
//   - rich 模式（默认）生成完整方法集
//
// 测试结构：TagControl.PlainXxx 为 plain 字段，ReadWrite 为对照 rich 字段。
func TestPlainMode(t *testing.T) {
	dir := goldenDir(t)
	pkgs, err := loader.Load(dir, loader.Config{}, ".")
	if err != nil {
		t.Fatalf("加载 testdata/examples 失败: %v", err)
	}
	structs, err := analyzer.Analyze(pkgs, analyzer.Config{})
	if err != nil {
		t.Fatalf("分析 testdata/examples 失败: %v", err)
	}

	var tagControlCode []byte
	reg := generator.NewRegistry()
	for _, s := range structs {
		if s.Name == "TagControl" {
			code, genErr := reg.GenerateStruct(s)
			if genErr != nil {
				t.Fatalf("生成 TagControl 失败: %v", genErr)
			}
			tagControlCode = code
			break
		}
	}
	if tagControlCode == nil {
		t.Fatal("未找到 TagControl 结构体")
	}

	// ── plain 模式：不应出现的扩展方法 ──────────────────────────────────────
	shouldNotExist := []string{
		"TogglePlainBool",       // bool plain：无 Toggle
		"AddPlainInt",           // numeric plain：无 Add
		"SubPlainInt",           // numeric plain：无 Sub
		"HasPlainPtr",           // pointer plain：无 Has
		"GetPlainSliceLen",      // slice plain：无 Len
		"HasPlainSlice",         // slice plain：无 Has
		"GetPlainSliceCopy",     // slice plain：无 GetCopy
		"HasPlainMap",           // map plain：无 Has
		"HasPlainMapKey",        // map plain：无 HasKey
		"GetPlainMapLen",        // map plain：无 GetLen
		"GetPlainMapKeys",       // map plain：无 GetKeys
		"GetPlainMapValOrDefault", // map plain：无 ValOrDefault
		"GetPlainMapCopy",       // map plain：无 GetCopy
	}
	for _, method := range shouldNotExist {
		if bytes.Contains(tagControlCode, []byte("func (this *TagControl) "+method)) {
			t.Errorf("plain 模式下不应生成方法 %s，但在生成代码中找到了", method)
		}
	}

	// ── plain 模式：应该存在的核心方法 ──────────────────────────────────────
	shouldExist := []string{
		"GetPlainBool",        // bool plain：有 Get
		"SetPlainBool",        // bool plain：有 Set
		"GetPlainInt",         // numeric plain：有 Get
		"SetPlainInt",         // numeric plain：有 Set
		"GetPlainPtr",         // pointer plain：有 Get
		"SetPlainPtr",         // pointer plain：有 Set
		"GetPlainSliceAt",     // slice plain：有 At
		"RangePlainSlice",     // slice plain：有 Range
		"SetPlainSliceAt",     // slice plain：有 SetAt
		"AppendPlainSlice",    // slice plain：有 Append
		"DeletePlainSlice",    // slice plain：有 Delete
		"GetPlainMapVal",      // map plain：有 Val
		"RangePlainMap",       // map plain：有 Range
		"SetPlainMapVal",      // map plain：有 SetVal
		"DeletePlainMapKey",   // map plain：有 DeleteKey
	}
	for _, method := range shouldExist {
		if !bytes.Contains(tagControlCode, []byte("func (this *TagControl) "+method)) {
			t.Errorf("plain 模式下应生成方法 %s，但在生成代码中未找到", method)
		}
	}

	// ── rich 模式对照：ReadWrite int（无 tag）应生成全套方法 ─────────────────
	richShouldExist := []string{
		"GetReadWrite",
		"SetReadWrite",
		"AddReadWrite",
		"SubReadWrite",
	}
	for _, method := range richShouldExist {
		if !bytes.Contains(tagControlCode, []byte("func (this *TagControl) "+method)) {
			t.Errorf("rich 模式下应生成方法 %s，但未找到", method)
		}
	}
}
