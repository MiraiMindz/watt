// Package ssr provides server-side rendering transpilation for GoX components
package ssr

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/printer"
	"go/token"
	"strings"

	"github.com/user/gox/pkg/analyzer"
)

// Transpiler handles SSR code generation for GoX components
type Transpiler struct {
	fset        *token.FileSet
	packageName string
	imports     map[string]bool
}

// New creates a new SSR Transpiler
func New(packageName string) *Transpiler {
	return &Transpiler{
		fset:        token.NewFileSet(),
		packageName: packageName,
		imports:     make(map[string]bool),
	}
}

// Transpile converts a ComponentIR to Go code for SSR
func (t *Transpiler) Transpile(ir *analyzer.ComponentIR) ([]byte, error) {
	var buf bytes.Buffer

	// Generate component struct
	buf.WriteString(t.generateComponentStruct(ir))
	buf.WriteString("\n\n")

	// Generate constructor
	buf.WriteString(t.generateConstructor(ir))
	buf.WriteString("\n\n")

	// Generate Render method
	buf.WriteString(t.generateRenderMethod(ir))
	buf.WriteString("\n\n")

	// Generate setState methods
	buf.WriteString(t.generateStateSetters(ir))

	// Generate helper methods
	buf.WriteString(t.generateHelperMethods(ir))

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Return unformatted code with error for debugging
		return buf.Bytes(), fmt.Errorf("format error: %w", err)
	}

	return formatted, nil
}

// GenerateFile generates a complete Go file with imports
func (t *Transpiler) GenerateFile(components []*analyzer.ComponentIR) ([]byte, error) {
	var buf bytes.Buffer

	// Package declaration
	buf.WriteString(fmt.Sprintf("package %s\n\n", t.packageName))

	// Imports
	buf.WriteString(t.generateImports())
	buf.WriteString("\n")

	// Generate each component
	for _, ir := range components {
		code, err := t.Transpile(ir)
		if err != nil {
			return nil, fmt.Errorf("failed to transpile component %s: %w", ir.Name, err)
		}
		buf.Write(code)
		buf.WriteString("\n\n")
	}

	// Format the entire file
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return buf.Bytes(), err
	}

	return formatted, nil
}

// generateImports generates the import statements
func (t *Transpiler) generateImports() string {
	// Default imports for SSR
	t.imports["fmt"] = true
	t.imports["github.com/user/gox/runtime"] = true

	var imports []string
	for imp := range t.imports {
		if strings.Contains(imp, "/") {
			// Package import with path
			imports = append(imports, fmt.Sprintf("\t\"%s\"", imp))
		} else {
			// Standard library import
			imports = append(imports, fmt.Sprintf("\t\"%s\"", imp))
		}
	}

	if len(imports) == 0 {
		return ""
	}

	return fmt.Sprintf("import (\n%s\n)\n", strings.Join(imports, "\n"))
}

// generateComponentStruct generates the component struct definition
func (t *Transpiler) generateComponentStruct(ir *analyzer.ComponentIR) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("// %s is a GoX component\n", ir.Name))
	buf.WriteString(fmt.Sprintf("type %s struct {\n", ir.Name))
	buf.WriteString("\t*gox.Component\n")

	// Props as fields
	for _, prop := range ir.Props {
		buf.WriteString(fmt.Sprintf("\t%s %s\n",
			capitalizeFirst(prop.Name),
			t.typeToString(prop.Type)))
	}

	// State fields
	for _, state := range ir.State {
		buf.WriteString(fmt.Sprintf("\t%s %s\n",
			state.Name,
			t.typeToString(state.Type)))
	}

	// Refs
	for _, ref := range ir.Refs {
		buf.WriteString(fmt.Sprintf("\t%sRef *gox.Ref[%s]\n",
			ref.Name,
			t.typeToString(ref.Type)))
	}

	// Memos
	for _, memo := range ir.Memos {
		buf.WriteString(fmt.Sprintf("\t%sMemo %s\n",
			memo.Name,
			t.typeToString(memo.Type)))
	}

	buf.WriteString("}")

	return buf.String()
}

// generateConstructor generates the component constructor function
func (t *Transpiler) generateConstructor(ir *analyzer.ComponentIR) string {
	var buf bytes.Buffer

	// Function signature
	buf.WriteString(fmt.Sprintf("// New%s creates a new %s component\n", ir.Name, ir.Name))
	buf.WriteString(fmt.Sprintf("func New%s(", ir.Name))

	// Parameters (props)
	for i, prop := range ir.Props {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("%s %s", prop.Name, t.typeToString(prop.Type)))
	}

	buf.WriteString(fmt.Sprintf(") *%s {\n", ir.Name))

	// Create instance
	buf.WriteString(fmt.Sprintf("\tc := &%s{\n", ir.Name))
	buf.WriteString("\t\tComponent: gox.NewComponent(),\n")

	// Initialize props
	for _, prop := range ir.Props {
		buf.WriteString(fmt.Sprintf("\t\t%s: %s,\n",
			capitalizeFirst(prop.Name),
			prop.Name))
	}

	buf.WriteString("\t}\n\n")

	// Initialize state
	for _, state := range ir.State {
		if state.InitValue != nil {
			buf.WriteString(fmt.Sprintf("\tc.%s = %s\n",
				state.Name,
				t.exprToString(state.InitValue)))
		}
	}

	// Initialize refs
	for _, ref := range ir.Refs {
		buf.WriteString(fmt.Sprintf("\tc.%sRef = gox.NewRef[%s](%s)\n",
			ref.Name,
			t.typeToString(ref.Type),
			t.exprToString(ref.InitValue)))
	}

	buf.WriteString("\treturn c\n")
	buf.WriteString("}")

	return buf.String()
}

// generateRenderMethod generates the Render method for SSR
func (t *Transpiler) generateRenderMethod(ir *analyzer.ComponentIR) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("// Render generates the HTML for the component\n"))
	buf.WriteString(fmt.Sprintf("func (c *%s) Render() string {\n", ir.Name))

	// Set current component for hooks
	buf.WriteString("\tgox.SetCurrentComponent(c.Component)\n")
	buf.WriteString("\tdefer func() { gox.SetCurrentComponent(nil) }()\n\n")

	// Reset hook state
	buf.WriteString("\tc.Component.hooks.Reset()\n\n")

	if ir.RenderLogic != nil && ir.RenderLogic.VNode != nil {
		// Generate HTML from VNode
		html := t.vnodeToHTML(ir.RenderLogic.VNode, "\t")
		buf.WriteString(fmt.Sprintf("\treturn `%s`\n", html))
	} else {
		buf.WriteString("\treturn \"\"\n")
	}

	buf.WriteString("}")

	return buf.String()
}

// vnodeToHTML converts a VNode IR to HTML template string
func (t *Transpiler) vnodeToHTML(vnode *analyzer.VNodeIR, indent string) string {
	switch vnode.Type {
	case analyzer.VNodeElement:
		return t.elementToHTML(vnode, indent)
	case analyzer.VNodeText:
		return vnode.Tag
	case analyzer.VNodeExpression:
		// For SSR, we need to interpolate the expression
		return fmt.Sprintf("${%s}", t.irExprToString(vnode.Props["expr"]))
	case analyzer.VNodeFragment:
		var parts []string
		for _, child := range vnode.Children {
			parts = append(parts, t.vnodeToHTML(child, indent))
		}
		return strings.Join(parts, "")
	case analyzer.VNodeConditional:
		// Simple conditional rendering for SSR
		return t.conditionalToHTML(vnode, indent)
	case analyzer.VNodeList:
		// List rendering for SSR
		return t.listToHTML(vnode, indent)
	default:
		return ""
	}
}

// elementToHTML converts an element VNode to HTML
func (t *Transpiler) elementToHTML(vnode *analyzer.VNodeIR, indent string) string {
	var buf bytes.Buffer

	// Opening tag
	buf.WriteString(fmt.Sprintf("<%s", vnode.Tag))

	// Attributes
	for name, value := range vnode.Props {
		// Skip event handlers for SSR
		if strings.HasPrefix(name, "on") {
			continue
		}

		if value.Type == analyzer.IRExprLiteral {
			buf.WriteString(fmt.Sprintf(` %s="%v"`, name, value.Value))
		} else {
			// Dynamic attribute - need to interpolate
			buf.WriteString(fmt.Sprintf(` %s="${%s}"`, name, t.irExprToString(value)))
		}
	}

	// Self-closing tags
	if isSelfClosingTag(vnode.Tag) && len(vnode.Children) == 0 {
		buf.WriteString(" />")
		return buf.String()
	}

	buf.WriteString(">")

	// Children
	for _, child := range vnode.Children {
		buf.WriteString(t.vnodeToHTML(child, indent+"\t"))
	}

	// Closing tag
	buf.WriteString(fmt.Sprintf("</%s>", vnode.Tag))

	return buf.String()
}

// conditionalToHTML generates conditional HTML
func (t *Transpiler) conditionalToHTML(vnode *analyzer.VNodeIR, indent string) string {
	// For SSR, we need to evaluate the condition at runtime
	// This is a simplified version
	if len(vnode.Children) > 0 {
		return t.vnodeToHTML(vnode.Children[0], indent)
	}
	return ""
}

// listToHTML generates list HTML
func (t *Transpiler) listToHTML(vnode *analyzer.VNodeIR, indent string) string {
	// For SSR, we need to iterate over the collection at runtime
	// This is a simplified version that generates a placeholder
	if vnode.Loop != nil && vnode.Loop.Body != nil {
		return fmt.Sprintf("<!-- list: %s -->", t.irExprToString(vnode.Loop.Collection))
	}
	return ""
}

// generateStateSetters generates setter methods for state variables
func (t *Transpiler) generateStateSetters(ir *analyzer.ComponentIR) string {
	var buf bytes.Buffer

	for _, state := range ir.State {
		buf.WriteString(fmt.Sprintf("// %s updates the %s state\n", state.Setter, state.Name))
		buf.WriteString(fmt.Sprintf("func (c *%s) %s(value %s) {\n",
			ir.Name,
			state.Setter,
			t.typeToString(state.Type)))
		buf.WriteString(fmt.Sprintf("\tc.%s = value\n", state.Name))
		buf.WriteString("\tc.RequestUpdate()\n")
		buf.WriteString("}\n\n")
	}

	return buf.String()
}

// generateHelperMethods generates additional helper methods
func (t *Transpiler) generateHelperMethods(ir *analyzer.ComponentIR) string {
	var buf bytes.Buffer

	// Generate effect methods if needed
	for i, effect := range ir.Effects {
		if effect.Setup != nil {
			buf.WriteString(fmt.Sprintf("// runEffect%d runs effect %d\n", i, i))
			buf.WriteString(fmt.Sprintf("func (c *%s) runEffect%d() {\n", ir.Name, i))
			buf.WriteString(t.blockStmtToString(effect.Setup, "\t"))
			buf.WriteString("}\n\n")
		}
	}

	// Generate memo compute functions
	for _, memo := range ir.Memos {
		buf.WriteString(fmt.Sprintf("// compute%s computes the memoized value\n", capitalizeFirst(memo.Name)))
		buf.WriteString(fmt.Sprintf("func (c *%s) compute%s() %s {\n",
			ir.Name,
			capitalizeFirst(memo.Name),
			t.typeToString(memo.Type)))
		buf.WriteString(fmt.Sprintf("\treturn %s\n", t.exprToString(memo.Compute)))
		buf.WriteString("}\n\n")
	}

	return buf.String()
}

// Helper methods for code generation

func (t *Transpiler) typeToString(expr ast.Expr) string {
	if expr == nil {
		return "interface{}"
	}

	var buf bytes.Buffer
	err := printer.Fprint(&buf, t.fset, expr)
	if err != nil {
		return "interface{}"
	}
	return buf.String()
}

func (t *Transpiler) exprToString(expr ast.Expr) string {
	if expr == nil {
		return "nil"
	}

	var buf bytes.Buffer
	err := printer.Fprint(&buf, t.fset, expr)
	if err != nil {
		return "nil"
	}
	return buf.String()
}

func (t *Transpiler) irExprToString(expr analyzer.IRExpr) string {
	switch expr.Type {
	case analyzer.IRExprLiteral:
		return fmt.Sprintf("%v", expr.Value)
	case analyzer.IRExprIdentifier:
		if expr.Original != nil {
			// For simple identifiers in SSR context, add "c." prefix
			if ident, ok := expr.Original.(*ast.Ident); ok {
				return fmt.Sprintf("c.%s", ident.Name)
			}
			return t.exprToString(expr.Original)
		}
		return fmt.Sprintf("c.%v", expr.Value)
	default:
		if expr.Original != nil {
			return t.exprToString(expr.Original)
		}
		return "nil"
	}
}

func (t *Transpiler) blockStmtToString(block *ast.BlockStmt, indent string) string {
	if block == nil {
		return ""
	}

	var buf bytes.Buffer
	for _, stmt := range block.List {
		buf.WriteString(indent)
		printer.Fprint(&buf, t.fset, stmt)
		buf.WriteString("\n")
	}
	return buf.String()
}

func capitalizeFirst(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}

func isSelfClosingTag(tag string) bool {
	selfClosing := map[string]bool{
		"area": true, "base": true, "br": true, "col": true,
		"embed": true, "hr": true, "img": true, "input": true,
		"link": true, "meta": true, "param": true, "source": true,
		"track": true, "wbr": true,
	}
	return selfClosing[strings.ToLower(tag)]
}