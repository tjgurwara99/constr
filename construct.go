package main

import (
	"bytes"
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
	typeName, fileName := parseFlags()

	realMain(fileName, typeName)
}

func realMain(fileName string, typeName string) {
	fset, node := parseProgram(fileName)

	data := inspectNode(node, typeName)
	funcDecl := generateConstructor(typeName, data...)
	insertConstructorToAst(node, typeName, funcDecl)
	writeToFile(fset, node, fileName)
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
			Lbrace: 1,
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.UnaryExpr{
							Op: token.AND,
							X: &ast.CompositeLit{
								Type:   ast.NewIdent(typeName),
								Elts:   elts,
								Rbrace: 1,
							},
						},
					},
				},
			},
			Rbrace: 2,
		},
	}
}

func writeToFile(fset *token.FileSet, node *ast.File, fileName string) {
	tmpBuf := bytes.Buffer{}
	err := format.Node(&tmpBuf, fset, node)
	if err != nil {
		log.Fatal("could not write the program into a buffer", err)
	}
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	_, err = io.Copy(file, &tmpBuf)
	if err != nil {
		log.Fatal("could not write program back to file", err)
	}
}

func inspectNode(node *ast.File, typeName string) []*ast.Field {
	var data []*ast.Field
	var constructExists bool
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
				constructExists = true
				return false
			}
		}
		return true
	})

	if constructExists {
		log.Fatal("constructor already exists")
	}
	return data
}

func parseProgram(fileName string) (*token.FileSet, *ast.File) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, fileName, nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}
	return fset, node
}

func parseFlags() (string, string) {
	var typeName string
	pflag.StringVarP(&typeName, "type", "t", "", "name of the type the constructor should return")
	pflag.Parse()
	args := pflag.Args()
	if len(args) > 1 || len(args) < 1 {
		log.Fatal("provide exactly one argument which will be the filename")
	}
	fileName := args[0]
	return typeName, fileName
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
