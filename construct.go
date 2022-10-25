package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"

	"github.com/spf13/pflag"
)

func main() {
	typeName, fileName, err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}

	err = realMain(fileName, typeName)
	if err != nil {
		log.Fatal(err)
	}
}

func realMain(fileName string, typeName string) error {
	fset, node, err := parseProgram(fileName)
	if err != nil {
		return err
	}

	data, err := inspectNode(node, typeName)
	if err != nil {
		return err
	}
	funcDecl := generateConstructor(typeName, data...)
	insertConstructorToAst(node, typeName, funcDecl)
	err = writeToFile(fset, node, fileName)
	if err != nil {
		return err
	}
	return nil
}

func generateConstructor(typeName string, fields ...*ast.Field) *ast.FuncDecl {
	var elts []ast.Expr
	for _, field := range fields {
		elts = append(elts, &ast.KeyValueExpr{
			Key:   field.Names[0],
			Value: field.Names[0],
		})
	}
	return &ast.FuncDecl{
		Doc:  nil,
		Recv: nil,
		Name: ast.NewIdent("New" + typeName),
		Type: &ast.FuncType{
			TypeParams: nil,
			Params: &ast.FieldList{
				List: fields,
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: &ast.UnaryExpr{
							Op: token.MUL,
							X:  ast.NewIdent(typeName),
						},
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.UnaryExpr{
							Op: token.AND,
							X: &ast.CompositeLit{
								Type: ast.NewIdent(typeName),
								Elts: elts,
							},
						},
					},
				},
			},
		},
	}
}

func writeToFile(fset *token.FileSet, node *ast.File, fileName string) error {
	tmpBuf := bytes.Buffer{}
	err := format.Node(&tmpBuf, fset, node)
	if err != nil {
		return fmt.Errorf("could not write the program into a buffer: %w", err)
	}
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, &tmpBuf)
	if err != nil {
		return fmt.Errorf("could not write program back to file: %w", err)
	}
	return nil
}

func inspectNode(node *ast.File, typeName string) ([]*ast.Field, error) {
	var data []*ast.Field
	var newFuncExists bool
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.TypeSpec:
			ident := x.Name
			if ident.Name == typeName {
				st, ok := x.Type.(*ast.StructType)
				if !ok {
					return false
				}
				data = st.Fields.List
			}
		case *ast.FuncDecl:
			ident := x.Name
			if ident.Name == "New"+typeName {
				newFuncExists = true
				return false
			}
		}
		return true
	})

	if newFuncExists {
		return nil, errors.New("constructor already exists")
	}
	return data, nil
}

func parseProgram(fileName string) (*token.FileSet, *ast.File, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, fileName, nil, parser.ParseComments)
	return fset, node, err
}

func parseFlags() (string, string, error) {
	var typeName string
	pflag.StringVarP(&typeName, "type", "t", "", "name of the type the constructor should return")
	pflag.Parse()
	args := pflag.Args()
	if len(args) > 1 || len(args) < 1 {
		return "", "", errors.New("no argument provided")
	}
	fileName := args[0]
	var err error
	if typeName == "" {
		err = errors.New("no type name provided")
	}
	return typeName, fileName, err
}

func insertConstructorToAst(node *ast.File, typeName string, funcDecl *ast.FuncDecl) {
	var declIndex int
	for i, decl := range node.Decls {
		x, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range x.Specs {
			sp, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if sp.Name.Name == typeName {
				declIndex = i
				break
			}
		}
	}
	node.Decls = insert[ast.Decl](node.Decls, declIndex+1, funcDecl)
}

func insert[T any](s []T, i int, val T) []T {
	if len(s) == i {
		return append(s, val)
	}
	s = append(s[:i+1], s[i:]...)
	s[i] = val
	return s
}
