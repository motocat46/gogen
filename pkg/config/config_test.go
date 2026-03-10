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
)

// TestLoad_NoFile 验证配置文件不存在时返回零值且无错误。
func TestLoad_NoFile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("文件不存在应返回零值，实际报错: %v", err)
	}
	if cfg.Suffix != "" || cfg.Output != "" || len(cfg.Excludes) != 0 || cfg.NoDefaultExcludes {
		t.Errorf("文件不存在时期望零值配置，实际: %+v", cfg)
	}
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
	if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte(content), 0o644); err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load 失败: %v", err)
	}
	if cfg.Suffix != "gen" {
		t.Errorf("Suffix 期望 'gen'，实际 '%s'", cfg.Suffix)
	}
	if cfg.Output != "./out" {
		t.Errorf("Output 期望 './out'，实际 '%s'", cfg.Output)
	}
	if len(cfg.Excludes) != 2 || cfg.Excludes[0] != "mock" || cfg.Excludes[1] != "proto" {
		t.Errorf("Excludes 期望 [mock, proto]，实际 %v", cfg.Excludes)
	}
	if !cfg.NoDefaultExcludes {
		t.Error("NoDefaultExcludes 期望 true，实际 false")
	}
}

// TestLoad_PartialConfig 验证部分字段配置文件（其余保持零值）。
func TestLoad_PartialConfig(t *testing.T) {
	dir := t.TempDir()
	content := `suffix: access
`
	if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte(content), 0o644); err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load 失败: %v", err)
	}
	if cfg.Suffix != "access" {
		t.Errorf("Suffix 期望 'access'，实际 '%s'", cfg.Suffix)
	}
	if cfg.Output != "" {
		t.Errorf("Output 未设置时期望空字符串，实际 '%s'", cfg.Output)
	}
}

// TestLoad_InvalidYAML 验证配置文件格式错误时返回 error。
func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	content := `this is: not: valid: yaml: [[[`
	if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte(content), 0o644); err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}

	_, err := config.Load(dir)
	if err == nil {
		t.Fatal("格式错误的 YAML 应返回 error，实际为 nil")
	}
}

// TestLoad_EmptyFile 验证空配置文件返回零值且无错误。
func TestLoad_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte(""), 0o644); err != nil {
		t.Fatalf("写入空配置文件失败: %v", err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("空配置文件应返回零值，实际报错: %v", err)
	}
	if cfg.Suffix != "" || cfg.Output != "" {
		t.Errorf("空配置文件期望零值，实际: %+v", cfg)
	}
}
