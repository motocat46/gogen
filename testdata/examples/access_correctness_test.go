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

// Package examples 的运行时访问正确性测试。
//
// 目标：验证生成的 getter/setter 操作的是正确的字段，而非同类型的其他字段。
// 测试方法：
//  1. 对某个字段调用 setter 写入唯一值
//  2. 通过 getter 读回，验证与写入值相同
//  3. 直接访问该字段，验证与写入值相同（证明 setter 没有写错字段）
//  4. 检查同类型的其他字段仍为零值（证明 setter 没有误写其他字段）
package examples

import (
	"testing"
)

// TestGetterSetterFieldCorrectness 验证每个 getter/setter 访问的是正确的字段。
//
// 核心策略：对同类型的多个字段逐一写入唯一值，每次写入后验证：
//   - getter 返回值 == 写入值
//   - 该字段直接读取 == 写入值
//   - 其他同类型字段仍为零值
func TestGetterSetterFieldCorrectness(t *testing.T) {
	t.Run("numeric_fields", func(t *testing.T) {
		// AllTypes 有多个同类型数值字段，是字段混淆 bug 的高风险场景
		s := &AllTypes{}

		// 逐一写入，每次确认只有目标字段变化
		s.SetFieldInt(1001)
		if s.GetFieldInt() != 1001 {
			t.Errorf("GetFieldInt() = %v, want 1001", s.GetFieldInt())
		}
		if s.FieldInt != 1001 {
			t.Errorf("SetFieldInt 写入了错误字段，FieldInt = %v", s.FieldInt)
		}
		if s.FieldInt8 != 0 || s.FieldInt16 != 0 || s.FieldInt32 != 0 || s.FieldInt64 != 0 {
			t.Errorf("SetFieldInt 误写了其他 int 字段: Int8=%v Int16=%v Int32=%v Int64=%v",
				s.FieldInt8, s.FieldInt16, s.FieldInt32, s.FieldInt64)
		}

		s.SetFieldInt8(102)
		if s.GetFieldInt8() != 102 {
			t.Errorf("GetFieldInt8() = %v, want 102", s.GetFieldInt8())
		}
		if s.FieldInt8 != 102 {
			t.Errorf("SetFieldInt8 写入了错误字段")
		}

		s.SetFieldInt64(1064)
		if s.GetFieldInt64() != 1064 {
			t.Errorf("GetFieldInt64() = %v, want 1064", s.GetFieldInt64())
		}
		if s.FieldInt64 != 1064 {
			t.Errorf("SetFieldInt64 写入了错误字段")
		}

		s.SetFieldFloat32(3.14)
		if s.GetFieldFloat32() != 3.14 {
			t.Errorf("GetFieldFloat32() = %v, want 3.14", s.GetFieldFloat32())
		}
		if s.FieldFloat32 != 3.14 {
			t.Errorf("SetFieldFloat32 写入了错误字段")
		}

		// Add/Sub 正确性
		s2 := &AllTypes{}
		s2.SetFieldInt(100)
		s2.AddFieldInt(50)
		if s2.GetFieldInt() != 150 {
			t.Errorf("AddFieldInt: GetFieldInt() = %v, want 150", s2.GetFieldInt())
		}
		if s2.FieldInt != 150 {
			t.Errorf("AddFieldInt 写入了错误字段")
		}
		s2.SubFieldInt(30)
		if s2.GetFieldInt() != 120 {
			t.Errorf("SubFieldInt: GetFieldInt() = %v, want 120", s2.GetFieldInt())
		}
	})

	t.Run("bool_field", func(t *testing.T) {
		s := &AllTypes{}

		s.SetFieldBool(true)
		if !s.GetFieldBool() {
			t.Error("GetFieldBool() 返回 false，期望 true")
		}
		if !s.FieldBool {
			t.Error("SetFieldBool 写入了错误字段")
		}

		// Toggle 正确性
		s.ToggleFieldBool()
		if s.GetFieldBool() {
			t.Error("ToggleFieldBool 后 GetFieldBool() 应为 false")
		}
		if s.FieldBool {
			t.Error("ToggleFieldBool 写入了错误字段")
		}
	})

	t.Run("string_field", func(t *testing.T) {
		s := &AllTypes{}

		s.SetFieldString("hello")
		if s.GetFieldString() != "hello" {
			t.Errorf("GetFieldString() = %q, want %q", s.GetFieldString(), "hello")
		}
		if s.FieldString != "hello" {
			t.Errorf("SetFieldString 写入了错误字段")
		}
	})

	t.Run("pointer_field", func(t *testing.T) {
		s := &AllTypes{}

		// Has 在设置前应返回 false
		if s.HasFieldPtrInt() {
			t.Error("HasFieldPtrInt() 应返回 false（未初始化）")
		}

		v := 42
		s.SetFieldPtrInt(&v)
		if !s.HasFieldPtrInt() {
			t.Error("HasFieldPtrInt() 在设置后应返回 true")
		}
		if s.GetFieldPtrInt() != &v {
			t.Error("GetFieldPtrInt() 返回了错误的指针")
		}
		if s.FieldPtrInt != &v {
			t.Error("SetFieldPtrInt 写入了错误字段")
		}

		// 确认另一个指针字段未受影响
		if s.FieldPtrString != nil || s.FieldPtrStruct != nil {
			t.Error("SetFieldPtrInt 误写了其他指针字段")
		}
	})
}

// TestSliceMethodFieldCorrectness 验证切片操作方法访问正确的字段。
func TestSliceMethodFieldCorrectness(t *testing.T) {
	s := &SliceOnly{}

	// AppendNames 追加到 Names，不影响 Scores 和 Items
	s.AppendNames("alice")
	s.AppendNames("bob")
	if len(s.Names) != 2 {
		t.Errorf("AppendNames 后 len(Names) = %d, want 2", len(s.Names))
	}
	if s.GetNamesAt(0) != "alice" {
		t.Errorf("GetNamesAt(0) = %q, want %q", s.GetNamesAt(0), "alice")
	}
	if s.GetNamesAt(1) != "bob" {
		t.Errorf("GetNamesAt(1) = %q, want %q", s.GetNamesAt(1), "bob")
	}
	if len(s.Scores) != 0 || len(s.Items) != 0 {
		t.Error("AppendNames 误写了其他切片字段")
	}

	// SetNamesAt 修改正确位置
	s.SetNamesAt(0, "charlie")
	if s.Names[0] != "charlie" {
		t.Errorf("SetNamesAt(0) 后 Names[0] = %q, want %q", s.Names[0], "charlie")
	}
	if s.GetNamesAt(0) != "charlie" {
		t.Errorf("GetNamesAt(0) 与直接访问不一致")
	}

	// AppendScores 不影响 Names
	s.AppendScores(9.9)
	if len(s.Names) != 2 {
		t.Error("AppendScores 误修改了 Names 的长度")
	}
	if len(s.Scores) != 1 || s.Scores[0] != 9.9 {
		t.Errorf("AppendScores 结果错误: %v", s.Scores)
	}

	// RemoveNames 删除正确元素
	s.RemoveNames(0) // 删除 "charlie"
	if len(s.Names) != 1 || s.Names[0] != "bob" {
		t.Errorf("RemoveNames(0) 后 Names = %v, want [bob]", s.Names)
	}

	// Range 遍历正确字段
	var ranged []string
	s.RangeNames(func(_ int, v string) bool {
		ranged = append(ranged, v)
		return true
	})
	if len(ranged) != 1 || ranged[0] != "bob" {
		t.Errorf("RangeNames 遍历结果 = %v, want [bob]", ranged)
	}

	// Has 正确性
	if !s.HasNames() {
		t.Error("HasNames() 在 Names 非 nil 时应返回 true")
	}
	var nilSlice *SliceOnly = &SliceOnly{}
	if nilSlice.HasNames() {
		t.Error("HasNames() 在 Names 为 nil 时应返回 false")
	}

	// GetNamesCopy 返回独立拷贝
	s.AppendNames("dave")
	copySlice := s.GetNamesCopy()
	copySlice[0] = "modified"
	if s.Names[0] == "modified" {
		t.Error("GetNamesCopy 返回的不是独立拷贝，修改拷贝影响了原始切片")
	}
}

// TestMapMethodFieldCorrectness 验证 map 操作方法访问正确的字段。
func TestMapMethodFieldCorrectness(t *testing.T) {
	s := &MapOnly{}

	// EnsureIndex 初始化
	s.EnsureIndex()
	if s.Index == nil {
		t.Error("EnsureIndex 后 Index 应非 nil")
	}
	if s.Config != nil || s.Nested != nil {
		t.Error("EnsureIndex 误初始化了其他 map 字段")
	}

	// SetIndexVal 写入正确字段
	s.SetIndexVal(1, "one")
	s.SetIndexVal(2, "two")
	if s.Index[1] != "one" || s.Index[2] != "two" {
		t.Errorf("SetIndexVal 写入错误: %v", s.Index)
	}

	// GetIndexVal 读取正确字段
	v, ok := s.GetIndexVal(1)
	if !ok || v != "one" {
		t.Errorf("GetIndexVal(1) = %q, %v; want %q, true", v, ok, "one")
	}
	_, ok = s.GetIndexVal(99)
	if ok {
		t.Error("GetIndexVal(99) 应返回 false（key 不存在）")
	}

	// GetIndexValOrDefault
	def := s.GetIndexValOrDefault(99, "default")
	if def != "default" {
		t.Errorf("GetIndexValOrDefault(99) = %q, want %q", def, "default")
	}
	existing := s.GetIndexValOrDefault(1, "default")
	if existing != "one" {
		t.Errorf("GetIndexValOrDefault(1) = %q, want %q", existing, "one")
	}

	// HasIndexKey 检查 key 存在性
	if !s.HasIndexKey(1) {
		t.Error("HasIndexKey(1) 应返回 true")
	}
	if s.HasIndexKey(99) {
		t.Error("HasIndexKey(99) 应返回 false")
	}

	// GetIndexLen
	if s.GetIndexLen() != 2 {
		t.Errorf("GetIndexLen() = %d, want 2", s.GetIndexLen())
	}

	// DelIndexKey 删除正确 key
	s.DelIndexKey(1)
	if _, exists := s.Index[1]; exists {
		t.Error("DelIndexKey(1) 后 key 1 仍存在")
	}
	if s.GetIndexLen() != 1 {
		t.Errorf("删除后 GetIndexLen() = %d, want 1", s.GetIndexLen())
	}

	// GetIndexKeys 返回正确 keys
	s.SetIndexVal(3, "three")
	keys := s.GetIndexKeys()
	if len(keys) != 2 {
		t.Errorf("GetIndexKeys() 长度 = %d, want 2", len(keys))
	}

	// GetIndexCopy 返回独立拷贝
	copyMap := s.GetIndexCopy()
	copyMap[999] = "injected"
	if _, exists := s.Index[999]; exists {
		t.Error("GetIndexCopy 返回的不是独立拷贝，修改影响了原始 map")
	}

	// Range 遍历正确字段（不影响其他 map）
	count := 0
	s.RangeIndex(func(_ int, _ string) bool {
		count++
		return true
	})
	if count != 2 {
		t.Errorf("RangeIndex 遍历 %d 次, want 2", count)
	}

	// Has 语义
	if !s.HasIndex() {
		t.Error("HasIndex() 在 Index 非 nil 时应返回 true")
	}
	empty := &MapOnly{}
	if empty.HasIndex() {
		t.Error("HasIndex() 在 Index 为 nil 时应返回 false")
	}
}

// TestPlainModeFieldCorrectness 验证 plain 模式下生成的方法访问正确字段。
func TestPlainModeFieldCorrectness(t *testing.T) {
	s := &TagControl{}

	// PlainInt：只有 Get/Set，验证字段访问正确
	s.SetPlainInt(777)
	if s.GetPlainInt() != 777 {
		t.Errorf("GetPlainInt() = %v, want 777", s.GetPlainInt())
	}
	if s.PlainInt != 777 {
		t.Errorf("SetPlainInt 写入了错误字段，PlainInt = %v", s.PlainInt)
	}
	// 确认 ReadWrite（同类型字段）未被误写
	if s.ReadWrite != 0 {
		t.Errorf("SetPlainInt 误写了 ReadWrite = %v", s.ReadWrite)
	}

	// PlainBool：只有 Get/Set（无 Toggle），验证字段访问正确
	s.SetPlainBool(true)
	if !s.GetPlainBool() {
		t.Error("GetPlainBool() 应返回 true")
	}
	if !s.PlainBool {
		t.Error("SetPlainBool 写入了错误字段")
	}

	// PlainSlice：验证 At/Range/Append/Remove 的字段隔离
	s.AppendPlainSlice("x")
	s.AppendPlainSlice("y")
	if s.GetPlainSliceAt(0) != "x" {
		t.Errorf("GetPlainSliceAt(0) = %q, want x", s.GetPlainSliceAt(0))
	}
	if s.PlainSlice[0] != "x" {
		t.Errorf("AppendPlainSlice 写入了错误字段")
	}

	s.SetPlainSliceAt(1, "z")
	if s.GetPlainSliceAt(1) != "z" {
		t.Errorf("GetPlainSliceAt(1) = %q, want z", s.GetPlainSliceAt(1))
	}

	s.RemovePlainSlice(0)
	if len(s.PlainSlice) != 1 || s.PlainSlice[0] != "z" {
		t.Errorf("RemovePlainSlice(0) 后 PlainSlice = %v, want [z]", s.PlainSlice)
	}

	// PlainMap：验证 Val/SetVal/DelKey 的字段隔离
	s.PlainMap = make(map[string]int)
	s.SetPlainMapVal("a", 1)
	v, ok := s.GetPlainMapVal("a")
	if !ok || v != 1 {
		t.Errorf("GetPlainMapVal(a) = %v, %v; want 1, true", v, ok)
	}
	if s.PlainMap["a"] != 1 {
		t.Error("SetPlainMapVal 写入了错误字段")
	}

	s.DelPlainMapKey("a")
	if _, exists := s.PlainMap["a"]; exists {
		t.Error("DelPlainMapKey 未正确删除")
	}
}
