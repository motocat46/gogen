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

// Package config 负责加载 .gogen.yaml 项目配置文件。
//
// 优先级规则：CLI 参数 > 配置文件 > 内置默认值
// 配置文件放在项目工作目录（go.mod 所在目录或 gogen 运行目录），文件名 .gogen.yaml。
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FileName 是 gogen 配置文件的默认文件名。
const FileName = ".gogen.yaml"

// File 表示 .gogen.yaml 配置文件的内容。
// 所有字段均为可选，未设置时使用 CLI 默认值。
//
// 示例文件：
//
//	suffix: access
//	output: ./gen
//	excludes:
//	  - mock
//	  - proto
//	  - vendor
//	no-default-excludes: false
type File struct {
	// Suffix 生成文件名后缀。对应 --suffix 标志。
	// 例如 "access" → user_access.go；"gen" → user_gen.go。
	Suffix string `yaml:"suffix"`

	// Output 统一输出目录。对应 --output 标志。
	// 空字符串表示与源文件同目录（默认行为）。
	Output string `yaml:"output"`

	// Excludes 额外排除的路径列表。对应多个 --exclude 标志。
	// 支持纯目录名（如 "mock"）或路径前缀（如 "./proto"）。
	Excludes []string `yaml:"excludes"`

	// NoDefaultExcludes 禁用内置默认排除（vendor、testdata 等）。
	// 对应 --no-default-excludes 标志。
	NoDefaultExcludes bool `yaml:"no-default-excludes" gogen:"plain"`
}

// Load 从指定目录加载 .gogen.yaml 配置文件。
//
// 若文件不存在，返回空配置（所有字段为零值）和 nil error——
// 这是正常情况，调用方不应将"文件不存在"视为错误。
//
// 若文件存在但格式错误，返回 error。
func Load(dir string) (File, error) {
	path := filepath.Join(dir, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return File{}, nil // 无配置文件，使用零值
		}
		return File{}, fmt.Errorf("读取配置文件 %s 失败: %w", path, err)
	}

	var cfg File
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return File{}, fmt.Errorf("解析配置文件 %s 失败: %w", path, err)
	}
	return cfg, nil
}
