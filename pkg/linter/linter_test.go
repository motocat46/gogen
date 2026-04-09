package linter_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/motocat46/gogen/pkg/linter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testdataDir 返回 testdata/lint 目录的绝对路径（作为 Lint 的 dir 参数）。
// pattern 使用 "./subdir" 相对形式，与 go/packages 官方格式一致。
func testdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	// pkg/linter/linter_test.go → 上两层是项目根
	root := filepath.Join(filepath.Dir(file), "..", "..")
	return filepath.Join(root, "testdata", "lint")
}

func TestLint(t *testing.T) {
	td := testdataDir(t)

	cases := []struct {
		name       string
		subdir     string
		wantErrors int
		wantWarns  int
	}{
		{
			name:       "拼写错误 tag",
			subdir:     "bad_tags",
			wantErrors: 3, // raedonly, unknownoption, dirty=（字段级 dirty 已废弃，视为未知选项）
			wantWarns:  0,
		},
		{
			name:       "矛盾组合",
			subdir:     "contradictions",
			wantErrors: 2, // readonly+writeonly, -+plain
			wantWarns:  0,
		},
		{
			name:       "dirty 方法不存在",
			subdir:     "dirty_missing",
			wantErrors: 1, // 结构体级 gogen:dirty=NonExistentMethod
			wantWarns:  0,
		},
		{
			name:       "modify= 无 dirty tracking",
			subdir:     "modify_no_dirty",
			wantErrors: 0,
			wantWarns:  2, // NoEffect + NoDirtyWithModify 各一条 Warning
		},
		{
			name:       "合法注解无问题",
			subdir:     "valid",
			wantErrors: 0,
			wantWarns:  0,
		},
		{
			name:       "dirty 方法在同包不同文件（跨文件检测）",
			subdir:     "multi_file",
			wantErrors: 0, // MakeDirty 定义在 methods.go，类型检查阶段已解析，应无 Error
			wantWarns:  0,
		},
		{
			name:       "多文件各含 issue（覆盖跨文件排序分支）",
			subdir:     "multi_file_errors",
			wantErrors: 14, // a.go 贡献 7 个 Error，b.go 贡献 7 个 Error；共 14 个触发 pdqsort 递归，覆盖 return -1/1
			wantWarns:  0,
		},
		{
			name:       "gogen tag 值为空字符串（跳过，不产生 issue）",
			subdir:     "empty_tag_value",
			wantErrors: 0,
			wantWarns:  0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			issues, err := linter.Lint(td, linter.Config{}, "./"+tc.subdir)
			require.NoError(t, err, "Lint 返回错误")

			var errors, warns int
			for _, iss := range issues {
				if iss.Severity == linter.Error {
					errors++
				} else {
					warns++
				}
			}

			if errors != tc.wantErrors || warns != tc.wantWarns {
				for _, iss := range issues {
					t.Logf("  %s", iss)
				}
			}
			assert.Equal(t, tc.wantErrors, errors, "Error 数量")
			assert.Equal(t, tc.wantWarns, warns, "Warning 数量")
		})
	}
}

// TestLint_LoadError 验证传入含编译错误的包时 Lint 返回非 nil error（覆盖 Load 错误路径）。
func TestLint_LoadError(t *testing.T) {
	td := testdataDir(t)
	_, err := linter.Lint(td, linter.Config{}, "./broken_syntax")
	assert.Error(t, err, "加载含编译错误的包应返回 error")
}
