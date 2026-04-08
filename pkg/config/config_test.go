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

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/motocat46/gogen/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoad_NoFile 验证配置文件不存在时返回零值且无错误。
func TestLoad_NoFile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Load(dir)
	require.NoError(t, err, "文件不存在应返回零值")
	assert.Empty(t, cfg.Suffix)
	assert.Empty(t, cfg.Output)
	assert.Empty(t, cfg.Excludes)
	assert.False(t, cfg.NoDefaultExcludes)
}

// TestLoad_FullConfig 验证完整配置文件的解析。
func TestLoad_FullConfig(t *testing.T) {
	dir := t.TempDir()
	content := `suffix: gen
output: ./out
excludes:
  - mock
  - proto
no-default-excludes: true
`
	err := os.WriteFile(filepath.Join(dir, config.FileName), []byte(content), 0o644)
	require.NoError(t, err, "写入配置文件失败")

	cfg, err := config.Load(dir)
	require.NoError(t, err, "Load 失败")
	assert.Equal(t, "gen", cfg.Suffix)
	assert.Equal(t, "./out", cfg.Output)
	require.Len(t, cfg.Excludes, 2)
	assert.Equal(t, "mock", cfg.Excludes[0])
	assert.Equal(t, "proto", cfg.Excludes[1])
	assert.True(t, cfg.NoDefaultExcludes)
}

// TestLoad_PartialConfig 验证部分字段配置文件（其余保持零值）。
func TestLoad_PartialConfig(t *testing.T) {
	dir := t.TempDir()
	content := `suffix: access
`
	err := os.WriteFile(filepath.Join(dir, config.FileName), []byte(content), 0o644)
	require.NoError(t, err, "写入配置文件失败")

	cfg, err := config.Load(dir)
	require.NoError(t, err, "Load 失败")
	assert.Equal(t, "access", cfg.Suffix)
	assert.Empty(t, cfg.Output)
}

// TestLoad_InvalidYAML 验证配置文件格式错误时返回 error。
func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	content := `this is: not: valid: yaml: [[[`
	err := os.WriteFile(filepath.Join(dir, config.FileName), []byte(content), 0o644)
	require.NoError(t, err, "写入配置文件失败")

	_, err = config.Load(dir)
	require.Error(t, err, "格式错误的 YAML 应返回 error")
}

// TestLoad_ReadError 验证路径存在但无法读取时返回 error（非 ErrNotExist 路径）。
// 使用同名目录代替文件：os.ReadFile 对目录返回 EISDIR，不属于 ErrNotExist。
func TestLoad_ReadError(t *testing.T) {
	dir := t.TempDir()
	// 创建与配置文件同名的目录，ReadFile 会报 EISDIR
	err := os.Mkdir(filepath.Join(dir, config.FileName), 0o755)
	require.NoError(t, err, "创建同名目录失败")

	_, err = config.Load(dir)
	require.Error(t, err, "读取目录应返回 error")
}

// TestLoad_EmptyFile 验证空配置文件返回零值且无错误。
func TestLoad_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, config.FileName), []byte(""), 0o644)
	require.NoError(t, err, "写入空配置文件失败")

	cfg, err := config.Load(dir)
	require.NoError(t, err, "空配置文件应返回零值")
	assert.Empty(t, cfg.Suffix)
	assert.Empty(t, cfg.Output)
}
