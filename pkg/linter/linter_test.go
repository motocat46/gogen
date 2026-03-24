package linter_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/motocat46/gogen/pkg/linter"
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
			name:       "合法注解无问题",
			subdir:     "valid",
			wantErrors: 0,
			wantWarns:  0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			issues, err := linter.Lint(td, linter.Config{}, "./"+tc.subdir)
			if err != nil {
				t.Fatalf("Lint 返回错误: %v", err)
			}

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
			if errors != tc.wantErrors {
				t.Errorf("Error 数量：got %d, want %d", errors, tc.wantErrors)
			}
			if warns != tc.wantWarns {
				t.Errorf("Warning 数量：got %d, want %d", warns, tc.wantWarns)
			}
		})
	}
}
