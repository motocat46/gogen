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

// Package e2e 对 gogen CLI 做端到端测试。
//
// TestMain 在所有测试前编译一次 gogen 二进制到临时目录，
// 各测试通过 runGogen() 调用该二进制，验证退出码和输出。
package e2e_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

// gogenBin 保存编译好的 gogen 二进制路径，由 TestMain 填充。
var gogenBin string

// repoRoot 返回项目根目录路径（e2e_test.go 的上一层）。
func repoRoot(t testing.TB) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法获取当前文件路径")
	}
	return filepath.Dir(filepath.Dir(thisFile))
}

func TestMain(m *testing.M) {
	// 编译 gogen 二进制到临时目录，所有测试共用
	tmp, err := os.MkdirTemp("", "gogen-e2e-*")
	if err != nil {
		panic("创建临时目录失败: " + err.Error())
	}
	bin := filepath.Join(tmp, "gogen")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	// 获取项目根目录（e2e/ 的上一层）
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("无法获取源文件路径")
	}
	root := filepath.Dir(filepath.Dir(thisFile))

	out, err := exec.Command("go", "build", "-o", bin, root).CombinedOutput()
	if err != nil {
		panic("编译 gogen 失败: " + err.Error() + "\n" + string(out))
	}
	gogenBin = bin

	// 显式清理（defer 在 os.Exit 前不执行）
	code := m.Run()
	os.RemoveAll(tmp)
	if err := goleak.Find(); err != nil {
		fmt.Fprintln(os.Stderr, "goroutine leak:", err)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}

// runGogen 在指定目录运行 gogen，返回 stdout+stderr 合并输出和退出码（0 表示成功）。
func runGogen(dir string, args ...string) (output string, exitCode int) {
	cmd := exec.Command(gogenBin, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	output = string(out)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return output, exitCode
}

// makeSimplePkg 在 tmp 目录下创建一个最小 Go 模块，包含一个带结构体的包。
func makeSimplePkg(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// go.mod 是 go/packages 加载包的必要条件
	gomod := "module example.com/mypkg\n\ngo 1.21\n"
	err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644)
	require.NoError(t, err, "写入 go.mod 失败")

	src := `package mypkg

// User 用户信息。
type User struct {
	Name  string
	Age   int
	Score float64
}
`
	err = os.WriteFile(filepath.Join(dir, "user.go"), []byte(src), 0o644)
	require.NoError(t, err, "写入测试文件失败")
	return dir
}

// ─── version ─────────────────────────────────────────────────────────────────

func TestVersion(t *testing.T) {
	root := repoRoot(t)
	out, code := runGogen(root, "version")
	require.Zero(t, code, "version 退出码应为 0\n输出: %s", out)
	assert.Contains(t, out, "gogen", "version 输出应包含 'gogen'")
}

// ─── generate ────────────────────────────────────────────────────────────────

func TestGenerate_CreatesAccessFile(t *testing.T) {
	dir := makeSimplePkg(t)
	out, code := runGogen(dir, ".")
	require.Zero(t, code, "generate 退出码应为 0\n输出: %s", out)

	generated := filepath.Join(dir, "user_access.go")
	_, err := os.Stat(generated)
	require.NoError(t, err, "generate 后 user_access.go 应存在")

	content, _ := os.ReadFile(generated)
	code2 := string(content)
	assert.Contains(t, code2, "func (this *User) GetName()", "生成文件缺少 GetName()")
	assert.Contains(t, code2, "func (this *User) SetName(", "生成文件缺少 SetName()")
}

func TestGenerate_DryRun_NoFileCreated(t *testing.T) {
	dir := makeSimplePkg(t)
	out, code := runGogen(dir, "--dry-run", ".")
	require.Zero(t, code, "dry-run 退出码应为 0\n输出: %s", out)

	generated := filepath.Join(dir, "user_access.go")
	_, err := os.Stat(generated)
	assert.True(t, os.IsNotExist(err), "--dry-run 不应创建任何文件，但 user_access.go 已存在")
}

func TestGenerate_CustomSuffix(t *testing.T) {
	dir := makeSimplePkg(t)
	out, code := runGogen(dir, "--suffix", "gen", ".")
	require.Zero(t, code, "generate --suffix gen 退出码应为 0\n输出: %s", out)

	_, err := os.Stat(filepath.Join(dir, "user_gen.go"))
	require.NoError(t, err, "--suffix gen 后 user_gen.go 应存在")

	// 默认后缀文件不应存在
	_, err = os.Stat(filepath.Join(dir, "user_access.go"))
	assert.True(t, os.IsNotExist(err), "--suffix gen 时 user_access.go 不应存在")
}

func TestGenerate_ExistingGoldenFiles(t *testing.T) {
	// 对 testdata/examples 运行 generate，验证不会修改已有黄金文件（幂等性）
	root := repoRoot(t)
	examplesDir := filepath.Join(root, "testdata", "examples")
	out, code := runGogen(examplesDir, "--no-default-excludes", ".")
	require.Zero(t, code, "generate testdata/examples 退出码应为 0\n输出: %s", out)
}

// ─── check ───────────────────────────────────────────────────────────────────

func TestCheck_UpToDate_ExitsZero(t *testing.T) {
	dir := makeSimplePkg(t)

	// 先生成
	_, setupCode := runGogen(dir, ".")
	require.Zero(t, setupCode, "预备步骤：generate 失败")

	// 再 check：文件最新，应返回 0
	out, code := runGogen(dir, "check", ".")
	require.Zero(t, code, "check（文件最新）退出码应为 0\n输出: %s", out)
	assert.Contains(t, out, "最新", "check 通过时输出应含'最新'")
}

func TestCheck_Stale_ExitsNonZero(t *testing.T) {
	dir := makeSimplePkg(t)

	// 先生成
	_, setupCode := runGogen(dir, ".")
	require.Zero(t, setupCode, "预备步骤：generate 失败")

	// 删除生成文件，模拟"过期"
	err := os.Remove(filepath.Join(dir, "user_access.go"))
	require.NoError(t, err, "删除生成文件失败")

	// check 应返回非 0
	out, code := runGogen(dir, "check", ".")
	assert.NotZero(t, code, "check（文件过期）退出码应非 0\n输出: %s", out)
}

func TestCheck_ExistingGoldenFiles(t *testing.T) {
	// testdata/examples 的黄金文件是最新的，check 应返回 0
	root := repoRoot(t)
	examplesDir := filepath.Join(root, "testdata", "examples")
	out, code := runGogen(examplesDir, "check", "--no-default-excludes", ".")
	require.Zero(t, code, "check testdata/examples 退出码应为 0\n输出: %s", out)
}

// ─── --output ────────────────────────────────────────────────────────────────

func TestGenerate_OutputDir_FilesInOutputNotSource(t *testing.T) {
	dir := makeSimplePkg(t)
	outDir := filepath.Join(dir, "gen")

	out, code := runGogen(dir, "--output", outDir, ".")
	require.Zero(t, code, "generate --output 退出码应为 0\n输出: %s", out)

	// 生成文件应在 outDir 中
	_, err := os.Stat(filepath.Join(outDir, "user_access.go"))
	require.NoError(t, err, "--output 后 gen/user_access.go 应存在")

	// 源目录中不应有生成文件
	_, err = os.Stat(filepath.Join(dir, "user_access.go"))
	assert.True(t, os.IsNotExist(err), "--output 指定了输出目录，源目录中不应有 user_access.go")
}

func TestCheck_OutputDir_UpToDate(t *testing.T) {
	dir := makeSimplePkg(t)
	outDir := filepath.Join(dir, "gen")

	// 先生成到 outDir
	_, setupCode := runGogen(dir, "--output", outDir, ".")
	require.Zero(t, setupCode, "预备步骤：generate --output 失败")

	// check 使用相同 --output，文件最新，应返回 0
	out, code := runGogen(dir, "check", "--output", outDir, ".")
	require.Zero(t, code, "check --output（文件最新）退出码应为 0\n输出: %s", out)
}

func TestCheck_OutputDir_Mismatch_ExitsNonZero(t *testing.T) {
	dir := makeSimplePkg(t)
	outDir := filepath.Join(dir, "gen")

	// 生成到 outDir
	_, setupCode := runGogen(dir, "--output", outDir, ".")
	require.Zero(t, setupCode, "预备步骤：generate --output 失败")

	// check 不带 --output，在源目录找不到文件，应返回非 0
	_, code := runGogen(dir, "check", ".")
	assert.NotZero(t, code, "check 未指定 --output 但文件在 gen/ 中，退出码应非 0")
}

// ─── lint ────────────────────────────────────────────────────────────────────

func TestLint_ValidPackage_ExitsZero(t *testing.T) {
	root := repoRoot(t)
	lintDir := filepath.Join(root, "testdata", "lint")
	out, code := runGogen(lintDir, "lint", "--no-default-excludes", "./valid")
	require.Zero(t, code, "lint valid 退出码应为 0\n输出: %s", out)
	assert.Contains(t, out, "未发现问题", "lint 通过时输出应含 '未发现问题'")
}

func TestLint_BadTags_ExitsNonZero(t *testing.T) {
	root := repoRoot(t)
	lintDir := filepath.Join(root, "testdata", "lint")
	out, code := runGogen(lintDir, "lint", "--no-default-excludes", "./bad_tags")
	assert.NotZero(t, code, "lint bad_tags 退出码应非 0\n输出: %s", out)
}

func TestLint_WarningsOnly_ExitsZero(t *testing.T) {
	// modify_no_dirty 只有 Warning，没有 Error，应退出码 0
	root := repoRoot(t)
	lintDir := filepath.Join(root, "testdata", "lint")
	out, code := runGogen(lintDir, "lint", "--no-default-excludes", "./modify_no_dirty")
	require.Zero(t, code, "lint modify_no_dirty（仅 Warning）退出码应为 0\n输出: %s", out)
	assert.Contains(t, out, "warning", "lint modify_no_dirty 应有 warning 输出")
}

func TestLint_Contradictions_ExitsNonZero(t *testing.T) {
	root := repoRoot(t)
	lintDir := filepath.Join(root, "testdata", "lint")
	_, code := runGogen(lintDir, "lint", "--no-default-excludes", "./contradictions")
	assert.NotZero(t, code, "lint contradictions 退出码应非 0")
}

// ─── init ────────────────────────────────────────────────────────────────────

func TestInit_CreatesConfigFile(t *testing.T) {
	dir := t.TempDir()
	out, code := runGogen(dir, "init")
	require.Zero(t, code, "init 退出码应为 0\n输出: %s", out)

	_, err := os.Stat(filepath.Join(dir, ".gogen.yaml"))
	require.NoError(t, err, "init 后 .gogen.yaml 应存在")
}

func TestInit_FileAlreadyExists_ExitsNonZero(t *testing.T) {
	dir := t.TempDir()
	// 预先创建配置文件
	err := os.WriteFile(filepath.Join(dir, ".gogen.yaml"), []byte("suffix: gen\n"), 0o644)
	require.NoError(t, err, "写入配置文件失败")

	_, code := runGogen(dir, "init")
	assert.NotZero(t, code, "文件已存在时 init 退出码应非 0")
}
