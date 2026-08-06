//go:build ignore

// api_not_migrated_scanner 由 api_route_inventory.mjs 调用，只使用 Go 标准库解析生产 Handler。
// build tag 避免该工具与 backend/scripts 中现有的 migration main 一同参与 go test 编译。
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const handlerxImportPath = "wplink/backend/app/internal/handler/handlerx"

type finding struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Label  string `json:"label"`
}

func main() {
	if len(os.Args) != 2 {
		fail(fmt.Errorf("用法: api_not_migrated_scanner <handler-dir>"))
	}
	root, err := filepath.Abs(os.Args[1])
	if err != nil {
		fail(err)
	}
	files := make([]string, 0)
	err = filepath.WalkDir(root, func(filePath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		files = append(files, filePath)
		return nil
	})
	if err != nil {
		fail(err)
	}
	sort.Strings(files)
	findings := make([]finding, 0)
	for _, filePath := range files {
		fileFindings, scanErr := scanFile(root, filePath)
		if scanErr != nil {
			fail(scanErr)
		}
		findings = append(findings, fileFindings...)
	}
	sort.Slice(findings, func(left, right int) bool {
		if findings[left].File != findings[right].File {
			return findings[left].File < findings[right].File
		}
		if findings[left].Line != findings[right].Line {
			return findings[left].Line < findings[right].Line
		}
		if findings[left].Column != findings[right].Column {
			return findings[left].Column < findings[right].Column
		}
		return findings[left].Label < findings[right].Label
	})
	if err := json.NewEncoder(os.Stdout).Encode(findings); err != nil {
		fail(err)
	}
}

func scanFile(root string, filePath string) ([]finding, error) {
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, filePath, nil, parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("解析 Go Handler 失败: %s: %w", filePath, err)
	}
	aliases := make(map[string]struct{})
	dotImported := false
	for _, importSpec := range parsed.Imports {
		importPath, unquoteErr := strconv.Unquote(importSpec.Path.Value)
		if unquoteErr != nil || importPath != handlerxImportPath {
			continue
		}
		alias := "handlerx"
		if importSpec.Name != nil {
			alias = importSpec.Name.Name
		}
		if alias == "." {
			dotImported = true
		} else if alias != "_" {
			aliases[alias] = struct{}{}
		}
	}
	if len(aliases) == 0 && !dotImported {
		return nil, nil
	}

	selectorNames := make(map[token.Pos]struct{})
	ignoredDotNames := make(map[token.Pos]struct{})
	directLabels := make(map[token.Pos]string)
	ast.Inspect(parsed, func(node ast.Node) bool {
		switch current := node.(type) {
		case *ast.SelectorExpr:
			selectorNames[current.Sel.Pos()] = struct{}{}
		case *ast.Field:
			for _, name := range current.Names {
				ignoredDotNames[name.Pos()] = struct{}{}
			}
		case *ast.ImportSpec:
			if current.Name != nil {
				ignoredDotNames[current.Name.Pos()] = struct{}{}
			}
		case *ast.KeyValueExpr:
			// 结构体字面量字段键不是标识符引用；map key 若需引用函数可显式加括号避免歧义。
			if identifier, ok := current.Key.(*ast.Ident); ok {
				ignoredDotNames[identifier.Pos()] = struct{}{}
			}
		case *ast.CallExpr:
			function := unwrapParentheses(current.Fun)
			switch reference := function.(type) {
			case *ast.SelectorExpr:
				if isProjectHandlerxSelector(reference, aliases) && reference.Sel.Name == "NotMigrated" {
					directLabels[reference.Sel.Pos()] = callLabel(current)
				}
			case *ast.Ident:
				if dotImported && reference.Name == "NotMigrated" && reference.Obj == nil {
					directLabels[reference.Pos()] = callLabel(current)
				}
			}
		}
		return true
	})

	byPosition := make(map[token.Pos]finding)
	ast.Inspect(parsed, func(node ast.Node) bool {
		switch current := node.(type) {
		case *ast.SelectorExpr:
			if isProjectHandlerxSelector(current, aliases) && current.Sel.Name == "NotMigrated" {
				label := directLabels[current.Sel.Pos()]
				if label == "" {
					label = "handlerx.NotMigrated"
				}
				byPosition[current.Sel.Pos()] = makeFinding(fileSet, root, current.Sel.Pos(), label)
			}
		case *ast.Ident:
			if !dotImported || current.Name != "NotMigrated" || current.Obj != nil {
				return true
			}
			if _, isSelectorName := selectorNames[current.Pos()]; isSelectorName {
				return true
			}
			if _, ignored := ignoredDotNames[current.Pos()]; ignored {
				return true
			}
			label := directLabels[current.Pos()]
			if label == "" {
				label = "handlerx.NotMigrated"
			}
			byPosition[current.Pos()] = makeFinding(fileSet, root, current.Pos(), label)
		}
		return true
	})
	findings := make([]finding, 0, len(byPosition))
	for _, item := range byPosition {
		findings = append(findings, item)
	}
	return findings, nil
}

func isProjectHandlerxSelector(selector *ast.SelectorExpr, aliases map[string]struct{}) bool {
	identifier, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	if _, imported := aliases[identifier.Name]; !imported {
		return false
	}
	// go/parser 会把参数、局部变量和短声明解析到各自 ast.Object；import package 使用保持未解析。
	return identifier.Obj == nil || identifier.Obj.Kind == ast.Pkg
}

func unwrapParentheses(expression ast.Expr) ast.Expr {
	for {
		parenthesized, ok := expression.(*ast.ParenExpr)
		if !ok {
			return expression
		}
		expression = parenthesized.X
	}
}

func callLabel(call *ast.CallExpr) string {
	if len(call.Args) == 0 {
		return "handlerx.NotMigrated"
	}
	literal, ok := call.Args[0].(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "handlerx.NotMigrated"
	}
	value, err := strconv.Unquote(literal.Value)
	if err != nil || value == "" {
		return "handlerx.NotMigrated"
	}
	return value
}

func makeFinding(fileSet *token.FileSet, root string, position token.Pos, label string) finding {
	location := fileSet.Position(position)
	relative, err := filepath.Rel(root, location.Filename)
	if err != nil {
		relative = location.Filename
	}
	return finding{
		File:   filepath.ToSlash(relative),
		Line:   location.Line,
		Column: location.Column,
		Label:  label,
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
