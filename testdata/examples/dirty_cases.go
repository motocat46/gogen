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
// 创建日期: 2026/3/18

package examples

// ── 场景 1：自动检测（方法集含 MakeDirty()，无标注）──────────────────────
// DirtyBase 是被嵌入的 dirty 基础类型，持有 dirty 状态。
// 业务层负责实现 MakeDirty()，gogen 不注入任何字段。
// Reset 手写实现，防止 gogen 生成 Reset 方法后被嵌入类型提升覆盖外层 Reset。
type DirtyBase struct{}

func (d *DirtyBase) MakeDirty() {}

// Reset 手写实现，防止 gogen 为 DirtyBase 生成 Reset（它本身无用户字段）。
// 避免通过提升机制干扰嵌入它的结构体（如 ResetWithDirtyPlayer）的 Reset 生成。
func (d *DirtyBase) Reset() {}

// AutoDirtyPlayer 无任何 gogen 注解，但嵌入 DirtyBase，
// 其方法集含 MakeDirty()，gogen 自动检测并注入。
type AutoDirtyPlayer struct {
	DirtyBase
	Gold  int64
	Score float64
	Tags  []string
	Attrs map[string]string
}

// ── 场景 2：自定义方法名 ────────────────────────────────────────────────

// gogen:dirty=MarkChanged
// CustomDirtyEntity 使用自定义 dirty 方法名 MarkChanged。
type CustomDirtyEntity struct {
	Name  string
	Level int32
}

func (e *CustomDirtyEntity) MarkChanged() {}

// ── 场景 3：nodirty（最高优先级）────────────────────────────────────────

// gogen:nodirty
// NoDirtyPlayer 显式禁用 dirty 注入，即使嵌入了 DirtyBase 也不生成 Modify()。
type NoDirtyPlayer struct {
	DirtyBase
	Gold  int64
	Score float64
}

// ── 场景 4：自定义 Modify 方法名 ─────────────────────────────────────────

// gogen:dirty
// gogen:modify=Apply
// FieldOverrideEntity 验证 gogen:modify=Apply 注解将生成的方法名从 Modify 改为 Apply。
type FieldOverrideEntity struct {
	Gold        int64
	ModuleScore int64
}

func (e *FieldOverrideEntity) MakeDirty() {}

// ── 场景 5：含 slice/map/array 字段的结构体，生成 Modify() 统一入口 ──────────

// AutoDirtyCollections 验证集合类型写方法的 dirty 注入行为。
// 使用自动检测（嵌入 DirtyBase）。
type AutoDirtyCollections struct {
	DirtyBase
	Tags  []string
	Attrs map[string]int32
	Nums  [4]int32
}

// ── 场景 6：Reset + dirty 交互 ───────────────────────────────────────────

// ResetWithDirtyPlayer 验证启用 dirty 的结构体 Reset() 末尾注入 dirty 调用。
// 不嵌入 DirtyBase（避免 DirtyBase.Reset 提升后阻止本结构体生成 Reset），
// 而是自行定义 MakeDirty() 触发自动检测。
type ResetWithDirtyPlayer struct {
	Name  string
	Level int32
}

func (r *ResetWithDirtyPlayer) MakeDirty() {}
