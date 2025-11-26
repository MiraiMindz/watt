// Package csr provides Client-Side Rendering transpilation for GoX components
package csr

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"strings"

	"github.com/user/gox/pkg/analyzer"
)

// Transpiler handles CSR/WASM transpilation
type Transpiler struct {
	packageName string
	imports     map[string]bool
}

// New creates a new CSR transpiler
func New(packageName string) *Transpiler {
	if packageName == "" {
		packageName = "main"
	}
	return &Transpiler{
		packageName: packageName,
		imports:     make(map[string]bool),
	}
}

// Transpile converts a ComponentIR to Go code for WASM
func (t *Transpiler) Transpile(ir *analyzer.ComponentIR) ([]byte, error) {
	var buf bytes.Buffer

	// Package declaration
	buf.WriteString(fmt.Sprintf("package %s\n\n", t.packageName))

	// Generate imports
	buf.WriteString(t.generateImports())
	buf.WriteString("\n")

	// Generate component struct
	buf.WriteString(t.generateComponentStruct(ir))
	buf.WriteString("\n\n")

	// Generate constructor
	buf.WriteString(t.generateConstructor(ir))
	buf.WriteString("\n\n")

	// Generate render method for WASM
	buf.WriteString(t.generateRenderMethod(ir))
	buf.WriteString("\n\n")

	// Generate update methods
	buf.WriteString(t.generateUpdateMethods(ir))
	buf.WriteString("\n\n")

	// Generate state setters
	buf.WriteString(t.generateStateSetters(ir))
	buf.WriteString("\n\n")

	// Generate event handlers
	buf.WriteString(t.generateEventHandlers(ir))
	buf.WriteString("\n\n")

	// Generate WASM exports
	buf.WriteString(t.generateWASMExports(ir))

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Print the generated code for debugging
		fmt.Printf("Generated code (before format):\n%s\n", buf.String())
		return buf.Bytes(), fmt.Errorf("format error: %w", err)
	}

	return formatted, nil
}

// generateImports generates import statements for CSR
func (t *Transpiler) generateImports() string {
	// Default imports for CSR/WASM
	t.imports["fmt"] = true
	t.imports["syscall/js"] = true
	t.imports["github.com/user/gox/runtime/wasm"] = true
	t.imports["github.com/user/gox/runtime"] = true

	var imports []string
	for imp := range t.imports {
		imports = append(imports, fmt.Sprintf("\t\"%s\"", imp))
	}

	return fmt.Sprintf("import (\n%s\n)\n", strings.Join(imports, "\n"))
}

// generateComponentStruct generates the component struct
func (t *Transpiler) generateComponentStruct(ir *analyzer.ComponentIR) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("// %s is a GoX component for WASM\n", ir.Name))
	buf.WriteString(fmt.Sprintf("type %s struct {\n", ir.Name))
	buf.WriteString("\t*wasm.Component\n")

	// Props
	for _, prop := range ir.Props {
		buf.WriteString(fmt.Sprintf("\t%s %s\n",
			capitalizeFirst(prop.Name),
			typeToString(prop.Type)))
	}

	// State
	for _, state := range ir.State {
		buf.WriteString(fmt.Sprintf("\t%s %s\n",
			state.Name,
			typeToString(state.Type)))
	}

	// DOM element reference
	buf.WriteString("\telement js.Value\n")
	buf.WriteString("\tdocument js.Value\n")

	buf.WriteString("}")

	return buf.String()
}

// generateConstructor generates the component constructor
func (t *Transpiler) generateConstructor(ir *analyzer.ComponentIR) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("// New%s creates a new %s component for WASM\n", ir.Name, ir.Name))

	// Generate function signature with props
	if len(ir.Props) > 0 {
		var params []string
		for _, prop := range ir.Props {
			params = append(params, fmt.Sprintf("%s %s",
				prop.Name,
				typeToString(prop.Type)))
		}
		buf.WriteString(fmt.Sprintf("func New%s(%s) *%s {\n",
			ir.Name, strings.Join(params, ", "), ir.Name))
	} else {
		buf.WriteString(fmt.Sprintf("func New%s() *%s {\n", ir.Name, ir.Name))
	}

	// Create instance
	buf.WriteString(fmt.Sprintf("\tc := &%s{\n", ir.Name))
	buf.WriteString("\t\tComponent: wasm.NewComponent(),\n")
	buf.WriteString("\t\tdocument: js.Global().Get(\"document\"),\n")

	// Set props
	for _, prop := range ir.Props {
		buf.WriteString(fmt.Sprintf("\t\t%s: %s,\n",
			capitalizeFirst(prop.Name), prop.Name))
	}
	buf.WriteString("\t}\n\n")

	// Initialize state with default values
	// Note: State initialization is typically handled via hooks in WASM
	// The initial values are passed as props or set via UseState

	// Initialize hooks
	if len(ir.Effects) > 0 || len(ir.State) > 0 {
		buf.WriteString("\n\t// Initialize hooks\n")
		buf.WriteString("\twasm.SetCurrentComponent(c.Component)\n")
		buf.WriteString("\tdefer wasm.SetCurrentComponent(nil)\n")
	}

	buf.WriteString("\n\treturn c\n")
	buf.WriteString("}")

	return buf.String()
}

// generateRenderMethod generates the render method for WASM
func (t *Transpiler) generateRenderMethod(ir *analyzer.ComponentIR) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("// Render renders the component to the DOM\n"))
	buf.WriteString(fmt.Sprintf("func (c *%s) Render(container js.Value) {\n", ir.Name))

	// Store container reference
	buf.WriteString("\tc.element = container\n\n")

	// Clear container
	buf.WriteString("\t// Clear existing content\n")
	buf.WriteString("\tcontainer.Set(\"innerHTML\", \"\")\n\n")

	if ir.RenderLogic != nil && ir.RenderLogic.VNode != nil {
		// Generate VDOM and render
		buf.WriteString("\t// Create virtual DOM\n")
		buf.WriteString("\tvdom := c.createVDOM()\n\n")
		buf.WriteString("\t// Render to actual DOM\n")
		buf.WriteString("\tc.renderVDOM(vdom, container)\n")
	}

	buf.WriteString("}\n")

	// Generate Hydrate method for SSR hydration
	buf.WriteString(fmt.Sprintf("\n// Hydrate hydrates server-rendered HTML\n"))
	buf.WriteString(fmt.Sprintf("func (c *%s) Hydrate(container js.Value) error {\n", ir.Name))
	buf.WriteString("\tc.element = container\n\n")
	buf.WriteString("\t// Create virtual DOM\n")
	buf.WriteString("\tvdom := c.createVDOM()\n\n")
	buf.WriteString("\t// Hydrate existing DOM with event listeners\n")
	buf.WriteString("\treturn c.Component.Hydrate(container, vdom)\n")
	buf.WriteString("}\n")

	// Generate createVDOM method
	buf.WriteString(fmt.Sprintf("\n// createVDOM creates the virtual DOM for the component\n"))
	buf.WriteString(fmt.Sprintf("func (c *%s) createVDOM() *wasm.VNode {\n", ir.Name))

	if ir.RenderLogic != nil && ir.RenderLogic.VNode != nil {
		vdomCode := t.generateVDOMCreation(ir.RenderLogic.VNode, "\t")
		buf.WriteString(vdomCode)
		buf.WriteString("\treturn vnode\n")
	} else {
		buf.WriteString("\treturn nil\n")
	}

	buf.WriteString("}\n")

	// Generate renderVDOM method
	buf.WriteString(fmt.Sprintf("\n// renderVDOM renders virtual DOM to actual DOM\n"))
	buf.WriteString(fmt.Sprintf("func (c *%s) renderVDOM(vnode *wasm.VNode, container js.Value) {\n", ir.Name))
	buf.WriteString("\tif vnode == nil {\n")
	buf.WriteString("\t\treturn\n")
	buf.WriteString("\t}\n\n")
	buf.WriteString("\t// Create DOM element\n")
	buf.WriteString("\telem := c.document.Call(\"createElement\", vnode.Tag)\n\n")
	buf.WriteString("\t// Set attributes\n")
	buf.WriteString("\tfor name, value := range vnode.Attrs {\n")
	buf.WriteString("\t\telem.Call(\"setAttribute\", name, value)\n")
	buf.WriteString("\t}\n\n")
	buf.WriteString("\t// Set event listeners\n")
	buf.WriteString("\tfor event, handler := range vnode.Events {\n")
	buf.WriteString("\t\telem.Call(\"addEventListener\", event, handler)\n")
	buf.WriteString("\t}\n\n")
	buf.WriteString("\t// Render children\n")
	buf.WriteString("\tfor _, child := range vnode.Children {\n")
	buf.WriteString("\t\tif child.Tag == \"\" {\n")
	buf.WriteString("\t\t\t// Text node\n")
	buf.WriteString("\t\t\ttext := c.document.Call(\"createTextNode\", child.Text)\n")
	buf.WriteString("\t\t\telem.Call(\"appendChild\", text)\n")
	buf.WriteString("\t\t} else {\n")
	buf.WriteString("\t\t\t// Element node\n")
	buf.WriteString("\t\t\tc.renderVDOM(child, elem)\n")
	buf.WriteString("\t\t}\n")
	buf.WriteString("\t}\n\n")
	buf.WriteString("\t// Append to container\n")
	buf.WriteString("\tcontainer.Call(\"appendChild\", elem)\n")
	buf.WriteString("}")

	return buf.String()
}

// generateVDOMCreation generates code to create VDOM from IR
func (t *Transpiler) generateVDOMCreation(vnode *analyzer.VNodeIR, indent string) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("%svnode := &wasm.VNode{\n", indent))
	buf.WriteString(fmt.Sprintf("%s\tTag: \"%s\",\n", indent, vnode.Tag))
	buf.WriteString(fmt.Sprintf("%s\tAttrs: make(map[string]string),\n", indent))
	buf.WriteString(fmt.Sprintf("%s\tEvents: make(map[string]js.Func),\n", indent))
	buf.WriteString(fmt.Sprintf("%s\tChildren: []*wasm.VNode{},\n", indent))
	buf.WriteString(fmt.Sprintf("%s}\n\n", indent))

	// Add attributes
	for name, value := range vnode.Props {
		if name == "onClick" {
			// Handle event listeners - extract identifier name from ast.Expr
			handlerName := "handleClick"
			if value.Original != nil {
				if ident, ok := value.Original.(*ast.Ident); ok {
					handlerName = ident.Name
				} else if call, ok := value.Original.(*ast.CallExpr); ok {
					// Handle inline function calls
					if ident, ok := call.Fun.(*ast.Ident); ok {
						handlerName = ident.Name
					}
				}
			} else if value.Value != nil && value.Value != true {
				handlerName = fmt.Sprintf("%v", value.Value)
			}
			buf.WriteString(fmt.Sprintf("%svnode.Events[\"click\"] = js.FuncOf(func(this js.Value, args []js.Value) interface{} {\n", indent))
			buf.WriteString(fmt.Sprintf("%s\tc.%s()\n", indent, handlerName))
			buf.WriteString(fmt.Sprintf("%s\treturn nil\n", indent))
			buf.WriteString(fmt.Sprintf("%s})\n", indent))
		} else if name == "className" {
			// Convert className to class
			classValue := ""
			if str, ok := value.Value.(string); ok {
				classValue = str
			} else if value.Value != nil {
				classValue = fmt.Sprintf("%v", value.Value)
			}
			buf.WriteString(fmt.Sprintf("%svnode.Attrs[\"class\"] = \"%s\"\n",
				indent, classValue))
		} else if name != "expr" { // Skip the special expr prop
			// Regular attributes
			attrValue := ""
			if str, ok := value.Value.(string); ok {
				attrValue = str
			} else if value.Value != nil {
				attrValue = fmt.Sprintf("%v", value.Value)
			}
			buf.WriteString(fmt.Sprintf("%svnode.Attrs[\"%s\"] = \"%s\"\n",
				indent, name, attrValue))
		}
	}

	// Add children
	if len(vnode.Children) > 0 {
		buf.WriteString(fmt.Sprintf("\n%s// Add children\n", indent))
		for _, child := range vnode.Children {
			if child.Type == analyzer.VNodeText {
				buf.WriteString(fmt.Sprintf("%svnode.Children = append(vnode.Children, &wasm.VNode{\n", indent))
				buf.WriteString(fmt.Sprintf("%s\tText: \"%s\",\n", indent, child.Tag))
				buf.WriteString(fmt.Sprintf("%s})\n", indent))
			} else if child.Type == analyzer.VNodeExpression {
				// Handle expressions - extract identifier from Original ast.Expr
				exprValue := "nil"
				if expr, ok := child.Props["expr"]; ok {
					if ident, ok := expr.Original.(*ast.Ident); ok {
						exprValue = fmt.Sprintf("c.%s", ident.Name)
					} else if expr.Value != nil {
						exprValue = fmt.Sprintf("c.%s", expr.Value)
					}
				} else if child.Tag != "" {
					exprValue = fmt.Sprintf("c.%s", child.Tag)
				}
				buf.WriteString(fmt.Sprintf("%svnode.Children = append(vnode.Children, &wasm.VNode{\n", indent))
				buf.WriteString(fmt.Sprintf("%s\tText: fmt.Sprint(%s),\n", indent, exprValue))
				buf.WriteString(fmt.Sprintf("%s})\n", indent))
			} else if child.Type == analyzer.VNodeElement {
				// Generate child VNode inline
				buf.WriteString(fmt.Sprintf("%svnode.Children = append(vnode.Children, func() *wasm.VNode {\n", indent))
				childCode := t.generateVDOMCreation(child, indent+"\t")
				buf.WriteString(childCode)
				buf.WriteString(fmt.Sprintf("%s\treturn vnode\n", indent))
				buf.WriteString(fmt.Sprintf("%s}())\n", indent))
			}
		}
	}

	return buf.String()
}

// generateUpdateMethods generates update/re-render methods
func (t *Transpiler) generateUpdateMethods(ir *analyzer.ComponentIR) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("// Update re-renders the component\n"))
	buf.WriteString(fmt.Sprintf("func (c *%s) Update() {\n", ir.Name))
	buf.WriteString("\tif c.element.IsUndefined() {\n")
	buf.WriteString("\t\treturn\n")
	buf.WriteString("\t}\n")
	buf.WriteString("\tc.Render(c.element)\n")
	buf.WriteString("}")

	return buf.String()
}

// generateStateSetters generates state setter methods
func (t *Transpiler) generateStateSetters(ir *analyzer.ComponentIR) string {
	var buf bytes.Buffer

	for _, state := range ir.State {
		setterName := "set" + capitalizeFirst(state.Name)
		buf.WriteString(fmt.Sprintf("// %s updates the %s state\n", setterName, state.Name))
		buf.WriteString(fmt.Sprintf("func (c *%s) %s(value %s) {\n",
			ir.Name, setterName, typeToString(state.Type)))
		buf.WriteString(fmt.Sprintf("\tc.%s = value\n", state.Name))
		buf.WriteString("\tc.Update()\n")
		buf.WriteString("}\n\n")
	}

	return buf.String()
}

// generateEventHandlers generates event handler methods
func (t *Transpiler) generateEventHandlers(ir *analyzer.ComponentIR) string {
	var buf bytes.Buffer

	// Generate basic event handlers
	buf.WriteString(fmt.Sprintf("// Event handlers\n"))

	// Example handlers based on common patterns
	for _, state := range ir.State {
		if strings.HasPrefix(state.Name, "count") {
			// Generate increment handler
			buf.WriteString(fmt.Sprintf("func (c *%s) handleIncrement(this js.Value, args []js.Value) interface{} {\n", ir.Name))
			buf.WriteString(fmt.Sprintf("\tc.set%s(c.%s + 1)\n",
				capitalizeFirst(state.Name), state.Name))
			buf.WriteString("\treturn nil\n")
			buf.WriteString("}\n\n")

			// Generate decrement handler
			buf.WriteString(fmt.Sprintf("func (c *%s) handleDecrement(this js.Value, args []js.Value) interface{} {\n", ir.Name))
			buf.WriteString(fmt.Sprintf("\tc.set%s(c.%s - 1)\n",
				capitalizeFirst(state.Name), state.Name))
			buf.WriteString("\treturn nil\n")
			buf.WriteString("}\n\n")
		}
	}

	return buf.String()
}

// generateWASMExports generates the main function and exports for WASM
func (t *Transpiler) generateWASMExports(ir *analyzer.ComponentIR) string {
	var buf bytes.Buffer

	buf.WriteString("// WASM exports\n")
	buf.WriteString("func main() {\n")
	buf.WriteString("\t// Register component factory\n")
	buf.WriteString(fmt.Sprintf("\tjs.Global().Set(\"%s\", js.FuncOf(func(this js.Value, args []js.Value) interface{} {\n", ir.Name))

	if len(ir.Props) > 0 {
		buf.WriteString("\t\t// Parse props from arguments\n")
		// For simplicity, assume first prop is passed
		buf.WriteString(fmt.Sprintf("\t\tcomp := New%s(", ir.Name))
		for i, prop := range ir.Props {
			if i > 0 {
				buf.WriteString(", ")
			}
			// Simple type conversion based on prop type
			// For now, assume int for numeric props, string otherwise
			if typeStr := typeToString(prop.Type); strings.Contains(typeStr, "int") {
				buf.WriteString("args[0].Int()")
			} else {
				buf.WriteString("args[0].String()")
			}
		}
		buf.WriteString(")\n")
	} else {
		buf.WriteString(fmt.Sprintf("\t\tcomp := New%s()\n", ir.Name))
	}

	buf.WriteString("\t\treturn comp\n")
	buf.WriteString("\t}))\n\n")

	buf.WriteString("\t// Keep the Go program running\n")
	buf.WriteString("\tselect {}\n")
	buf.WriteString("}")

	return buf.String()
}

// Helper functions

func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func typeToString(expr ast.Expr) string {
	// Extract type name from ast.Expr
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			return x.Name + "." + t.Sel.Name
		}
	case *ast.ArrayType:
		return "[]" + typeToString(t.Elt)
	case *ast.StarExpr:
		return "*" + typeToString(t.X)
	}
	return "interface{}"
}