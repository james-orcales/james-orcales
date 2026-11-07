// Package resolution rejects bare type names absent from their package.
package resolution

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"local/james-orcales/lint/internal/diagnostic"
	"local/james-orcales/lint/internal/source"
)

// TYPE_EXPRESSIONS_PER_FILE_MAX matches the source-line ceiling so malformed input cannot grow a
// second unbounded tree.
const TYPE_EXPRESSIONS_PER_FILE_MAX = 10_000

// Make closes over cross-file declarations because the common checker signature stays file-local.
func Make(index *source.Declaration_Index) (check_function func(
	file_set *token.FileSet, file *ast.File, source_text []byte,
) (diagnostics []diagnostic.Diagnostic)) {

	check_function = func(
		file_set *token.FileSet, file *ast.File, _ []byte,
	) (diagnostics []diagnostic.Diagnostic) {

		return check(file_set, file, index)
	}
	return check_function
}

func check(
	file_set *token.FileSet, file *ast.File, index *source.Declaration_Index,
) (diagnostics []diagnostic.Diagnostic) {

	if index == nil {
		return nil
	}
	file_path := file_set.Position(file.Package).Filename
	package_names := source.Package_Names(index, file_path)
	if package_names == nil {
		return nil
	}
	expressions := make([]ast.Expr, 0, TYPE_EXPRESSIONS_PER_FILE_MAX)
	overflow := false
	for _, declaration := range file.Decls {
		switch value := declaration.(type) {
		case *ast.GenDecl:
			expressions = append_declaration_types(expressions, value, &overflow)
		case *ast.FuncDecl:
			expressions = append_function_types(expressions, value, &overflow)
		}
	}
	for len(expressions) > 0 {
		last := len(expressions) - 1
		expression := expressions[last]
		expressions = expressions[:last]
		identifier, bare := expression.(*ast.Ident)
		if bare {
			if !name_is_type(index, package_names, file_path, identifier.Name) {
				diagnostics = append(diagnostics, diagnostic.Diagnostic{
					Position: file_set.Position(identifier.Pos()),
					Message: fmt.Sprintf(
						"%s does not resolve to type in package %s. "+
							"Declare type or correct name.",
						identifier.Name, file.Name.Name),
				})
			}
			continue
		}
		expressions = append_children(expressions, expression, &overflow)
	}
	if overflow {
		diagnostics = append(diagnostics, diagnostic.Diagnostic{
			Position: file_set.Position(file.Package),
			Message: fmt.Sprintf(
				"Type declarations exceed %d expressions. Split file.",
				TYPE_EXPRESSIONS_PER_FILE_MAX),
		})
	}
	return diagnostics
}

func append_declaration_types(
	expressions []ast.Expr, declaration *ast.GenDecl, overflow *bool,
) (output []ast.Expr) {

	output = expressions
	for _, specification := range declaration.Specs {
		switch value := specification.(type) {
		case *ast.TypeSpec:
			if value.TypeParams != nil {
				continue
			}
			output = append_expression(output, value.Type, overflow)
		case *ast.ValueSpec:
			output = append_expression(output, value.Type, overflow)
		}
	}
	return output
}

func append_function_types(
	expressions []ast.Expr, declaration *ast.FuncDecl, overflow *bool,
) (output []ast.Expr) {

	if declaration.Type.TypeParams != nil {
		return expressions
	}
	output = append_fields(expressions, declaration.Recv, overflow)
	output = append_fields(output, declaration.Type.Params, overflow)
	return append_fields(output, declaration.Type.Results, overflow)
}

func append_children(
	expressions []ast.Expr, expression ast.Expr, overflow *bool,
) (output []ast.Expr) {

	output = expressions
	switch value := expression.(type) {
	case *ast.ArrayType:
		output = append_expression(output, value.Elt, overflow)
	case *ast.ChanType:
		output = append_expression(output, value.Value, overflow)
	case *ast.Ellipsis:
		output = append_expression(output, value.Elt, overflow)
	case *ast.FuncType:
		output = append_fields(output, value.TypeParams, overflow)
		output = append_fields(output, value.Params, overflow)
		output = append_fields(output, value.Results, overflow)
	case *ast.IndexExpr:
		output = append_expression(output, value.X, overflow)
		output = append_expression(output, value.Index, overflow)
	case *ast.IndexListExpr:
		output = append_expression(output, value.X, overflow)
		for _, index := range value.Indices {
			output = append_expression(output, index, overflow)
		}
	case *ast.InterfaceType:
		output = append_fields(output, value.Methods, overflow)
	case *ast.MapType:
		output = append_expression(output, value.Key, overflow)
		output = append_expression(output, value.Value, overflow)
	case *ast.ParenExpr:
		output = append_expression(output, value.X, overflow)
	case *ast.StarExpr:
		output = append_expression(output, value.X, overflow)
	case *ast.StructType:
		output = append_fields(output, value.Fields, overflow)
	case *ast.UnaryExpr:
		output = append_expression(output, value.X, overflow)
	case *ast.BinaryExpr:
		output = append_expression(output, value.X, overflow)
		output = append_expression(output, value.Y, overflow)
	}
	return output
}

func append_fields(
	expressions []ast.Expr, fields *ast.FieldList, overflow *bool,
) (output []ast.Expr) {

	output = expressions
	if fields == nil {
		return output
	}
	for _, field := range fields.List {
		output = append_expression(output, field.Type, overflow)
	}
	return output
}

func append_expression(
	expressions []ast.Expr, expression ast.Expr, overflow *bool,
) (output []ast.Expr) {

	if expression == nil {
		return expressions
	}
	if len(expressions) == cap(expressions) {
		*overflow = true
		return expressions
	}
	return append(expressions, expression)
}

func name_is_type(
	index *source.Declaration_Index,
	package_names map[string]bool,
	file_path string,
	name string,
) (is_type bool) {

	object := types.Universe.Lookup(name)
	if object != nil {
		_, is_type = object.(*types.TypeName)
		return is_type
	}
	declaration, found := source.Resolve(&source.Resolve_Input{
		Index: index, Path: file_path, Name: name,
	})
	if found {
		return declaration.Kind == source.DECLARATION_KIND_TYPE
	}
	return package_names[name]
}
