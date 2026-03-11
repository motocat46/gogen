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

// Package analyzer 负责将 go/packages 加载的包信息转换为领域模型（model.StructDef）。
//
// 分析策略：
//   - 类型信息（TypeInfo）来自 go/types，语义正确，支持别名、泛型、跨文件引用
//   - 注释和 struct tag 来自 AST（go/ast），因为 go/types 不保留这些信息
//   - 通过 pkg.TypesInfo.Defs 关联 AST 节点与类型对象，避免手工遍历
package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"

	"github.com/motocat46/gogen/pkg/model"

	"golang.org/x/tools/go/packages"
)

// Config 控制分析行为。
type Config struct {
	// FileFilter 为文件绝对路径列表：非空时只分析这些文件（用户指定具体 .go 文件的场景）。
	FileFilter []string
	// ExcludePaths 为需要跳过的路径前缀列表，支持目录或文件的绝对路径。
	// 例如 ["/project/mocks", "/project/proto"] 会跳过这两个目录下的所有文件。
	ExcludePaths []string
}

// Analyze 分析一组已加载的包，提取结构体定义，返回领域模型列表。
//
// 自动跳过规则（无需配置）：
//   - 带有 "// Code generated ... DO NOT EDIT." 标记的文件（mockgen/protobuf/gogen 等工具生成的文件）
//
// 手动排除规则（通过 Config.ExcludePaths 配置）：
//   - 指定目录或文件路径前缀下的所有文件
func Analyze(pkgs []*packages.Package, cfg Config) ([]*model.StructDef, error) {
	// 构建文件过滤集合，O(1) 查找
	filterSet := make(map[string]struct{}, len(cfg.FileFilter))
	for _, f := range cfg.FileFilter {
		filterSet[f] = struct{}{}
	}

	var result []*model.StructDef
	for _, pkg := range pkgs {
		structs, err := analyzePackage(pkg, filterSet, cfg.ExcludePaths)
		if err != nil {
			return nil, err
		}
		result = append(result, structs...)
	}
	return result, nil
}

// analyzePackage 分析单个包，提取结构体定义。
func analyzePackage(pkg *packages.Package, filterSet map[string]struct{}, excludePaths []string) ([]*model.StructDef, error) {
	// 构建文件名 → AST 文件的映射，供 collectManualMethods 使用。
	// 方法可能定义在包内任意文件，必须先建全量映射。
	fileMap := make(map[string]*ast.File, len(pkg.Syntax))
	for _, f := range pkg.Syntax {
		pos := pkg.Fset.Position(f.Pos())
		fileMap[pos.Filename] = f
	}

	var result []*model.StructDef

	for _, file := range pkg.Syntax {
		pos := pkg.Fset.Position(file.Pos())
		filename := pos.Filename

		// 跳过所有代码生成文件（mockgen/protobuf/gogen 等工具输出）
		// 标准标记：首行或文件开头附近的 "// Code generated ... DO NOT EDIT."
		if isGeneratedFile(file) {
			continue
		}

		// 跳过用户指定的排除路径
		if isExcluded(filename, excludePaths) {
			continue
		}

		// 过滤：若指定了文件列表，跳过不在列表中的文件
		if len(filterSet) > 0 {
			pos := pkg.Fset.Position(file.Pos())
			if _, ok := filterSet[pos.Filename]; !ok {
				continue
			}
		}

		structs, err := analyzeFile(pkg, file, fileMap)
		if err != nil {
			return nil, err
		}
		result = append(result, structs...)
	}
	return result, nil
}

// analyzeFile 分析单个 AST 文件，提取结构体定义。
// fileMap 为包内所有文件的映射，用于判断方法定义所在文件是否为手写文件。
func analyzeFile(pkg *packages.Package, file *ast.File, fileMap map[string]*ast.File) ([]*model.StructDef, error) {
	var result []*model.StructDef

	// 获取文件所在目录（用于生成文件的输出路径）
	pos := pkg.Fset.Position(file.Pos())
	dir := filepath.Dir(pos.Filename)

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			// 只处理结构体类型
			astStructType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// 从 go/types 获取对应的 types.Object，用于语义分析
			obj := pkg.TypesInfo.Defs[typeSpec.Name]
			if obj == nil {
				continue
			}
			typesNamed, ok := obj.Type().(*types.Named)
			if !ok {
				continue
			}
			typesStruct, ok := typesNamed.Underlying().(*types.Struct)
			if !ok {
				continue
			}

			// 解析结构体文档注释
			doc := extractDoc(genDecl.Doc, typeSpec.Comment)

			// 解析字段列表
			fields := analyzeFields(pkg, astStructType, typesStruct, typesNamed)

			// 收集已在手写文件中定义的方法，避免生成时产生重复声明冲突
			manualMethods := collectManualMethods(typesNamed, pkg, fileMap)

			// 收集所有字段名（含不导出字段），用于检测生成方法名与字段名的冲突。
			// Go 规范：方法名不能与同类型的字段名相同。
			fieldNames := make(map[string]bool, typesStruct.NumFields())
			for f := range typesStruct.Fields() {
				fieldNames[f.Name()] = true
			}

			// 收集通过嵌入字段可访问的提升方法名，防止生成同名方法覆盖提升语义
			promotedMethods := collectPromotedMethods(typesNamed)

			result = append(result, &model.StructDef{
				Name:            typeSpec.Name.Name,
				TypeParams:      extractTypeParams(typesNamed),
				PackageName:     pkg.Name,
				PackagePath:     pkg.PkgPath,
				Dir:             dir,
				Fields:          fields,
				Doc:             doc,
				ManualMethods:   manualMethods,
				FieldNames:      fieldNames,
				PromotedMethods: promotedMethods,
			})
		}
	}
	return result, nil
}

// analyzeFields 分析结构体的字段列表。
// 同时使用 AST（获取注释、tag）和 go/types（获取类型信息）。
func analyzeFields(
	pkg *packages.Package,
	astStruct *ast.StructType,
	typesStruct *types.Struct,
	_ *types.Named,
) []*model.FieldDef {
	var result []*model.FieldDef

	// 构建字段名 → go/types 字段的映射，方便按名查找
	typesFieldMap := make(map[string]*types.Var, typesStruct.NumFields())
	for f := range typesStruct.Fields() {
		typesFieldMap[f.Name()] = f
	}

	for _, astField := range astStruct.Fields.List {
		// 跳过匿名嵌入字段（无字段名）
		if len(astField.Names) == 0 {
			continue
		}

		// 解析 struct tag
		rawTag := ""
		if astField.Tag != nil {
			// Tag.Value 包含反引号，去掉后得到原始 tag 字符串
			rawTag = strings.Trim(astField.Tag.Value, "`")
		}
		cfg := model.ParseFieldConfig(rawTag)

		// 解析注释
		doc := extractDoc(astField.Doc, astField.Comment)

		for _, nameIdent := range astField.Names {
			// 跳过非导出字段（小写开头）
			if !nameIdent.IsExported() {
				continue
			}

			typesVar, ok := typesFieldMap[nameIdent.Name]
			if !ok {
				continue
			}

			// 用 go/types 解析类型，语义正确
			qualifier := qualifierFor(pkg)
			typeInfo := buildTypeInfo(typesVar.Type(), qualifier)

			result = append(result, &model.FieldDef{
				Name:    nameIdent.Name,
				Type:    typeInfo,
				Config:  cfg,
				Doc:     doc,
				Comment: "",
			})
		}
	}
	return result
}

// buildTypeInfo 递归地将 go/types.Type 转换为 model.TypeInfo。
//
// 使用 types.TypeString 渲染类型字符串，天然支持：
//   - 所有基础类型、指针、slice、array、map
//   - 跨包引用（如 time.Time）
//   - 泛型实例（如 sync.Map、自定义 List[int]）
//   - 类型别名
func buildTypeInfo(t types.Type, qualifier types.Qualifier) *model.TypeInfo {
	typeStr := types.TypeString(t, qualifier)

	switch u := t.(type) {
	case *types.Basic:
		kind := model.KindBasic
		info := u.Info()
		switch {
		case info&types.IsBoolean != 0:
			kind = model.KindBool
		case info&(types.IsInteger|types.IsFloat|types.IsComplex) != 0:
			kind = model.KindNumeric
		}
		return &model.TypeInfo{Kind: kind, TypeStr: typeStr}

	case *types.Pointer:
		elem := buildTypeInfo(u.Elem(), qualifier)
		return &model.TypeInfo{Kind: model.KindPointer, TypeStr: typeStr, Elem: elem}

	case *types.Slice:
		elem := buildTypeInfo(u.Elem(), qualifier)
		return &model.TypeInfo{Kind: model.KindSlice, TypeStr: typeStr, Elem: elem}

	case *types.Array:
		elem := buildTypeInfo(u.Elem(), qualifier)
		return &model.TypeInfo{
			Kind:     model.KindArray,
			TypeStr:  typeStr,
			Elem:     elem,
			ArrayLen: fmt.Sprintf("%d", u.Len()),
		}

	case *types.Map:
		key := buildTypeInfo(u.Key(), qualifier)
		val := buildTypeInfo(u.Elem(), qualifier)
		return &model.TypeInfo{Kind: model.KindMap, TypeStr: typeStr, Key: key, Value: val}

	case *types.Alias:
		// Go 1.22+ 中类型别名（type X = T）由独立的 *types.Alias 节点表示。
		// 透明地递归解析被别名的类型，但保留别名名称作为 TypeStr，
		// 这样生成的代码仍使用 "MyTime" 而非 "time.Time"。
		info := buildTypeInfo(u.Rhs(), qualifier)
		// 复制一份避免修改共享节点
		copy := *info
		copy.TypeStr = typeStr
		copy.IsAlias = true
		return &copy

	case *types.Named:
		// 泛型实例化类型（如 List[int]）
		if u.TypeArgs() != nil && u.TypeArgs().Len() > 0 {
			var typeArgs []*model.TypeInfo
			for arg := range u.TypeArgs().Types() {
				typeArgs = append(typeArgs, buildTypeInfo(arg, qualifier))
			}
			return &model.TypeInfo{Kind: model.KindGeneric, TypeStr: typeStr, TypeArgs: typeArgs}
		}
		// 底层为结构体的具名类型（如 time.Time、自定义结构体）
		if _, ok := u.Underlying().(*types.Struct); ok {
			return &model.TypeInfo{Kind: model.KindStruct, TypeStr: typeStr, IsAlias: u.Obj().IsAlias()}
		}
		// 其他具名类型（如 type Status string、type UserID int64、type Tags []string）
		// 递归解析底层类型确定 Kind，TypeStr 保留具名类型名称
		underlying := buildTypeInfo(u.Underlying(), qualifier)
		result := *underlying // 复制，避免修改
		result.TypeStr = typeStr
		result.IsAlias = u.Obj().IsAlias()
		return &result

	case *types.Interface:
		// interface{}/any 及具名接口（如 io.Reader）：生成 Get/Set/Has，nil 表示未初始化
		return &model.TypeInfo{Kind: model.KindInterface, TypeStr: typeStr}

	case *types.Signature:
		// func 类型字段（如 func(int) string）：生成 Get/Set/Has，nil 表示未设置
		return &model.TypeInfo{Kind: model.KindFunc, TypeStr: typeStr}

	case *types.Chan:
		return &model.TypeInfo{Kind: model.KindUnsupported, TypeStr: typeStr}

	case *types.TypeParam:
		// 泛型类型参数（如 T、K、V）：在泛型结构体的接收者范围内是合法类型。
		// 复用 KindBasic 生成 Get/Set，TypeStr 即类型参数名（如 "T"）。
		// 生成代码：func (this *Cache[K, V]) GetItem() T { return this.Item }
		return &model.TypeInfo{Kind: model.KindBasic, TypeStr: typeStr}

	default:
		return &model.TypeInfo{Kind: model.KindUnsupported, TypeStr: typeStr}
	}
}

// qualifierFor 返回一个 types.Qualifier，使同包内的类型不带包名前缀，
// 跨包类型保留包名，生成的代码在目标包中直接可用。
func qualifierFor(pkg *packages.Package) types.Qualifier {
	return func(other *types.Package) string {
		if other.Path() == pkg.PkgPath {
			return "" // 同包内省略包名
		}
		return other.Name()
	}
}

// extractDoc 从文档注释和行注释中提取注释文本。
func extractDoc(doc, line *ast.CommentGroup) string {
	if doc != nil {
		return strings.TrimSpace(doc.Text())
	}
	if line != nil {
		return strings.TrimSpace(line.Text())
	}
	return ""
}

// collectManualMethods 收集结构体类型上在**手写文件**中定义的方法名集合。
//
// 判定逻辑：
//   - 遍历 named 类型的所有显式声明方法（含值接收者和指针接收者）
//   - 通过方法位置查找对应的 AST 文件
//   - 若该文件带有 "Code generated ... DO NOT EDIT." 标记，则视为生成文件，跳过
//   - 其余文件中定义的方法视为手写方法，加入集合
//
// 生成器根据此集合跳过已有手写实现的方法，避免产生"方法重复声明"编译错误。
func collectManualMethods(named *types.Named, pkg *packages.Package, fileMap map[string]*ast.File) map[string]bool {
	methods := make(map[string]bool)
	for m := range named.Methods() {
		pos := pkg.Fset.Position(m.Pos())
		astFile, ok := fileMap[pos.Filename]
		if !ok {
			// 方法定义在当前文件集合之外（不应发生），安全跳过
			continue
		}
		if isGeneratedFile(astFile) {
			// 生成文件中的方法（含 gogen 自身上次的输出）不算手写，可覆盖
			continue
		}
		methods[m.Name()] = true
	}
	return methods
}

// isGeneratedFile 检查文件是否为代码生成工具的输出。
//
// 依据 Go 官方约定（https://pkg.go.dev/cmd/go#hdr-Generate_Go_files_by_processing_source）：
// 生成文件的首行注释必须匹配 "^// Code generated .* DO NOT EDIT\.$"
// mockgen、protoc-gen-go、gogen 等主流工具均遵循此约定。
func isGeneratedFile(file *ast.File) bool {
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			if strings.HasPrefix(c.Text, "// Code generated") &&
				strings.Contains(c.Text, "DO NOT EDIT") {
				return true
			}
		}
	}
	return false
}

// collectPromotedMethods 收集结构体通过嵌入字段可访问的方法名集合。
//
// 关键设计：直接遍历 struct 的嵌入字段（Anonymous()=true），收集其方法集，
// 而非查询外层类型的方法集。原因：若外层类型已有同名直接方法（含已生成方法），
// 外层方法集会显示直接方法（Index 长度为 1），遮蔽提升方法，导致无法自愈。
//
// 同时处理值嵌入（Outer struct { Inner }）和指针嵌入（Outer struct { *Inner }）。
// 使用 *T 的方法集（值接收者 + 指针接收者全集），确保不漏报指针接收者方法。
func collectPromotedMethods(named *types.Named) map[string]bool {
	underlying, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil
	}
	promoted := make(map[string]bool)
	for field := range underlying.Fields() {
		if !field.Anonymous() {
			continue // 只处理嵌入字段（匿名字段）
		}
		ft := field.Type()
		if ptr, ok := ft.(*types.Pointer); ok {
			ft = ptr.Elem() // 指针嵌入：去指针取基础类型
		}
		// 使用 *ft 的方法集：包含 ft 自身及其所有深层嵌入的方法（递归覆盖）
		mset := types.NewMethodSet(types.NewPointer(ft))
		for sel := range mset.Methods() {
			promoted[sel.Obj().Name()] = true
		}
	}
	return promoted
}

// extractTypeParams 提取泛型结构体的类型参数名列表，返回如 "[K, V]" 的字符串。
// 非泛型结构体返回空字符串。
//
// 注意：接收者中只需要类型参数名（不含约束），例如：
//
//	type Cache[K comparable, V any] struct{} → 接收者 Cache[K, V]
func extractTypeParams(named *types.Named) string {
	tparams := named.TypeParams()
	if tparams == nil || tparams.Len() == 0 {
		return ""
	}
	names := make([]string, tparams.Len())
	for i := range tparams.Len() {
		names[i] = tparams.At(i).Obj().Name()
	}
	return "[" + strings.Join(names, ", ") + "]"
}

// isExcluded 判断文件路径是否匹配排除规则。
//   - 含路径分隔符或绝对路径的规则：前缀匹配，支持目录和具体文件路径
//   - 纯目录名（如 "mock"、"mocks"）：匹配路径中任意一段，覆盖嵌套任意层级
func isExcluded(filename string, excludePaths []string) bool {
	for _, ex := range excludePaths {
		if strings.ContainsRune(ex, filepath.Separator) || filepath.IsAbs(ex) {
			if strings.HasPrefix(filename, ex) {
				return true
			}
		} else {
			for seg := range strings.SplitSeq(filename, string(filepath.Separator)) {
				if seg == ex {
					return true
				}
			}
		}
	}
	return false
}
