package analyzer

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/user/gox/pkg/parser"
)

// Analyzer performs semantic analysis on GoX components
type Analyzer struct {
	components map[string]*ComponentIR
	errors     []error
	warnings   []string
	current    *ComponentIR // current component being analyzed
}

// New creates a new Analyzer instance
func New() *Analyzer {
	return &Analyzer{
		components: make(map[string]*ComponentIR),
		errors:     []error{},
		warnings:   []string{},
	}
}

// Analyze performs semantic analysis on parsed GoX components
func (a *Analyzer) Analyze(file *parser.File) (*AnalysisResult, error) {
	result := &AnalysisResult{
		Components: make(map[string]*ComponentIR),
		Errors:     []error{},
		Warnings:   []string{},
	}

	// Analyze each component
	for _, comp := range file.Components {
		ir, err := a.analyzeComponent(comp)
		if err != nil {
			result.Errors = append(result.Errors, err)
			continue
		}

		result.Components[ir.Name] = ir
		a.components[ir.Name] = ir
	}

	result.Errors = append(result.Errors, a.errors...)
	result.Warnings = append(result.Warnings, a.warnings...)

	if len(result.Errors) > 0 {
		return result, fmt.Errorf("analysis errors: %v", result.Errors)
	}

	return result, nil
}

// analyzeComponent analyzes a single component
func (a *Analyzer) analyzeComponent(comp *parser.ComponentDecl) (*ComponentIR, error) {
	ir := NewComponentIR(comp.Name.Name)
	a.current = ir

	// Extract props from parameters
	if comp.Params != nil {
		ir.Props = a.extractProps(comp.Params)
	}

	// Analyze component body
	if comp.Body != nil {
		// Analyze hooks
		for _, hook := range comp.Body.Hooks {
			a.analyzeHook(hook, ir)
		}

		// Analyze render block
		if comp.Body.Render != nil {
			ir.RenderLogic = a.analyzeRender(comp.Body.Render)
		}

		// Analyze styles
		if comp.Body.Style != nil {
			ir.Styles = a.analyzeStyle(comp.Body.Style, ir.Name)
		}
	}

	// Validate component
	a.validateComponent(ir)

	return ir, nil
}

// extractProps extracts prop fields from component parameters
func (a *Analyzer) extractProps(params *ast.FieldList) []PropField {
	var props []PropField

	for _, field := range params.List {
		for _, name := range field.Names {
			prop := PropField{
				Name: name.Name,
				Type: field.Type,
			}

			// Check for optional props (pointer types)
			if _, isPointer := field.Type.(*ast.StarExpr); isPointer {
				prop.Optional = true
			}

			props = append(props, prop)
		}
	}

	return props
}

// analyzeHook analyzes a hook call
func (a *Analyzer) analyzeHook(hook *parser.HookCall, ir *ComponentIR) {
	switch hook.Name {
	case "UseState", "useState":
		state := a.analyzeUseState(hook)
		if state != nil {
			ir.State = append(ir.State, *state)
		}

	case "UseEffect", "useEffect":
		effect := a.analyzeUseEffect(hook)
		if effect != nil {
			ir.Effects = append(ir.Effects, *effect)
		}

	case "UseMemo", "useMemo":
		memo := a.analyzeUseMemo(hook)
		if memo != nil {
			ir.Memos = append(ir.Memos, *memo)
		}

	case "UseRef", "useRef":
		ref := a.analyzeUseRef(hook)
		if ref != nil {
			ir.Refs = append(ir.Refs, *ref)
		}

	case "UseCallback", "useCallback":
		callback := a.analyzeUseCallback(hook)
		if callback != nil {
			ir.Callbacks = append(ir.Callbacks, *callback)
		}

	case "UseContext", "useContext":
		ctx := a.analyzeUseContext(hook)
		if ctx != nil {
			ir.Context = append(ir.Context, *ctx)
		}

	default:
		// Custom hook
		custom := CustomHookCall{
			Name:    hook.Name,
			Args:    hook.Args,
			Results: hook.Results,
		}
		ir.CustomHooks = append(ir.CustomHooks, custom)
	}
}

// analyzeUseState analyzes a useState hook call
func (a *Analyzer) analyzeUseState(hook *parser.HookCall) *StateVar {
	state := &StateVar{}

	// Get return values (value, setValue)
	if len(hook.Results) >= 2 {
		state.Name = hook.Results[0]
		state.Setter = hook.Results[1]
	} else {
		a.addError("useState must have two return values")
		return nil
	}

	// Get type argument
	if len(hook.TypeArgs) > 0 {
		state.Type = hook.TypeArgs[0]
	}

	// Get initial value
	if len(hook.Args) > 0 {
		state.InitValue = hook.Args[0]
	}

	return state
}

// analyzeUseEffect analyzes a useEffect hook call
func (a *Analyzer) analyzeUseEffect(hook *parser.HookCall) *EffectHook {
	effect := &EffectHook{}

	// First argument should be a function
	if len(hook.Args) > 0 {
		if fn, ok := hook.Args[0].(*ast.FuncLit); ok {
			effect.Setup = fn.Body

			// Check if function returns cleanup function
			// This is a simplified check
			// TODO: Implement proper return value analysis
		}
	}

	// Second argument is dependency array
	if len(hook.Args) > 1 {
		effect.Deps = a.parseDependencyArray(hook.Args[1])
	}

	return effect
}

// analyzeUseMemo analyzes a useMemo hook call
func (a *Analyzer) analyzeUseMemo(hook *parser.HookCall) *MemoHook {
	memo := &MemoHook{}

	// Get return value
	if len(hook.Results) > 0 {
		memo.Name = hook.Results[0]
	}

	// Get type argument
	if len(hook.TypeArgs) > 0 {
		memo.Type = hook.TypeArgs[0]
	}

	// Get compute function
	if len(hook.Args) > 0 {
		memo.Compute = hook.Args[0]
	}

	// Get dependencies
	if len(hook.Args) > 1 {
		memo.Deps = a.parseDependencyArray(hook.Args[1])
	}

	return memo
}

// analyzeUseRef analyzes a useRef hook call
func (a *Analyzer) analyzeUseRef(hook *parser.HookCall) *RefHook {
	ref := &RefHook{}

	// Get return value
	if len(hook.Results) > 0 {
		ref.Name = hook.Results[0]
	}

	// Get type argument
	if len(hook.TypeArgs) > 0 {
		ref.Type = hook.TypeArgs[0]
	}

	// Get initial value
	if len(hook.Args) > 0 {
		ref.InitValue = hook.Args[0]
	}

	return ref
}

// analyzeUseCallback analyzes a useCallback hook call
func (a *Analyzer) analyzeUseCallback(hook *parser.HookCall) *CallbackHook {
	callback := &CallbackHook{}

	// Get return value
	if len(hook.Results) > 0 {
		callback.Name = hook.Results[0]
	}

	// Get callback function
	if len(hook.Args) > 0 {
		if fn, ok := hook.Args[0].(*ast.FuncLit); ok {
			callback.Callback = fn
		}
	}

	// Get dependencies
	if len(hook.Args) > 1 {
		callback.Deps = a.parseDependencyArray(hook.Args[1])
	}

	return callback
}

// analyzeUseContext analyzes a useContext hook call
func (a *Analyzer) analyzeUseContext(hook *parser.HookCall) *ContextHook {
	ctx := &ContextHook{}

	// Get return value
	if len(hook.Results) > 0 {
		ctx.Name = hook.Results[0]
	}

	// Get type argument
	if len(hook.TypeArgs) > 0 {
		ctx.Type = hook.TypeArgs[0]
	}

	// Get context name
	if len(hook.Args) > 0 {
		if ident, ok := hook.Args[0].(*ast.Ident); ok {
			ctx.ContextName = ident.Name
		}
	}

	return ctx
}

// analyzeRender analyzes the render block
func (a *Analyzer) analyzeRender(render *parser.RenderBlock) *RenderIR {
	renderIR := &RenderIR{}

	if render.Root != nil {
		renderIR.VNode = JSXNodeToVNodeIR(render.Root)

		// Validate JSX usage
		a.validateJSX(renderIR.VNode)
	}

	return renderIR
}

// analyzeStyle analyzes the style block
func (a *Analyzer) analyzeStyle(style *parser.StyleBlock, componentName string) *StyleIR {
	styleIR := &StyleIR{
		Scoped:      !style.Global,
		ComponentID: componentName,
		Rules:       []CSSRuleIR{},
	}

	for _, rule := range style.Rules {
		ruleIR := CSSRuleIR{
			Selector:   rule.Selector,
			Properties: make(map[string]string),
			Scoped:     !style.Global,
		}

		for _, prop := range rule.Properties {
			ruleIR.Properties[prop.Name] = prop.Value
		}

		styleIR.Rules = append(styleIR.Rules, ruleIR)
	}

	return styleIR
}

// parseDependencyArray parses a dependency array expression
func (a *Analyzer) parseDependencyArray(expr ast.Expr) []string {
	var deps []string

	// Handle []interface{}{dep1, dep2, ...}
	if composite, ok := expr.(*ast.CompositeLit); ok {
		for _, elt := range composite.Elts {
			if ident, ok := elt.(*ast.Ident); ok {
				deps = append(deps, ident.Name)
			}
		}
	}

	return deps
}

// validateComponent validates a component's structure
func (a *Analyzer) validateComponent(ir *ComponentIR) {
	// Check for render block
	if ir.RenderLogic == nil {
		a.addWarning(fmt.Sprintf("Component %s has no render block", ir.Name))
	}

	// Validate hooks usage
	a.validateHooks(ir)

	// Check for unused props
	a.checkUnusedProps(ir)

	// Check for unused state
	a.checkUnusedState(ir)
}

// validateHooks validates hook usage rules
func (a *Analyzer) validateHooks(ir *ComponentIR) {
	// Check for conditional hooks (simplified check)
	// In a real implementation, we'd need to analyze the control flow

	// Check for hooks in loops
	// This requires more sophisticated analysis

	// Check for duplicate state names
	stateNames := make(map[string]bool)
	for _, state := range ir.State {
		if stateNames[state.Name] {
			a.addError(fmt.Sprintf("Duplicate state variable: %s", state.Name))
		}
		stateNames[state.Name] = true

		if stateNames[state.Setter] {
			a.addError(fmt.Sprintf("State setter conflicts with existing name: %s", state.Setter))
		}
		stateNames[state.Setter] = true
	}
}

// validateJSX validates JSX usage
func (a *Analyzer) validateJSX(vnode *VNodeIR) {
	if vnode == nil {
		return
	}

	// Check for component references
	if vnode.Type == VNodeComponent {
		// Check if component exists
		if !a.isHTMLTag(vnode.Tag) && !a.componentExists(vnode.Tag) {
			dependencies = append(dependencies, vnode.Tag)
		}
	}

	// Validate children
	for _, child := range vnode.Children {
		a.validateJSX(child)
	}

	// Validate loop
	if vnode.Loop != nil {
		a.validateJSX(vnode.Loop.Body)
	}
}

// checkUnusedProps checks for unused props
func (a *Analyzer) checkUnusedProps(ir *ComponentIR) {
	// This requires analyzing the render logic and all functions
	// For now, this is a placeholder
	// TODO: Implement proper usage analysis
}

// checkUnusedState checks for unused state variables
func (a *Analyzer) checkUnusedState(ir *ComponentIR) {
	// This requires analyzing the render logic and all functions
	// For now, this is a placeholder
	// TODO: Implement proper usage analysis
}

// Helper methods

func (a *Analyzer) addError(msg string) {
	a.errors = append(a.errors, fmt.Errorf("%s", msg))
}

func (a *Analyzer) addWarning(msg string) {
	a.warnings = append(a.warnings, msg)
}

var dependencies []string

func (a *Analyzer) componentExists(name string) bool {
	_, exists := a.components[name]
	return exists
}

func (a *Analyzer) isHTMLTag(tag string) bool {
	// Common HTML tags
	htmlTags := map[string]bool{
		"a": true, "abbr": true, "address": true, "area": true, "article": true,
		"aside": true, "audio": true, "b": true, "base": true, "bdi": true,
		"bdo": true, "blockquote": true, "body": true, "br": true, "button": true,
		"canvas": true, "caption": true, "cite": true, "code": true, "col": true,
		"colgroup": true, "data": true, "datalist": true, "dd": true, "del": true,
		"details": true, "dfn": true, "dialog": true, "div": true, "dl": true,
		"dt": true, "em": true, "embed": true, "fieldset": true, "figcaption": true,
		"figure": true, "footer": true, "form": true, "h1": true, "h2": true,
		"h3": true, "h4": true, "h5": true, "h6": true, "head": true, "header": true,
		"hr": true, "html": true, "i": true, "iframe": true, "img": true,
		"input": true, "ins": true, "kbd": true, "label": true, "legend": true,
		"li": true, "link": true, "main": true, "map": true, "mark": true,
		"meta": true, "meter": true, "nav": true, "noscript": true, "object": true,
		"ol": true, "optgroup": true, "option": true, "output": true, "p": true,
		"param": true, "picture": true, "pre": true, "progress": true, "q": true,
		"rp": true, "rt": true, "ruby": true, "s": true, "samp": true,
		"script": true, "section": true, "select": true, "small": true, "source": true,
		"span": true, "strong": true, "style": true, "sub": true, "summary": true,
		"sup": true, "svg": true, "table": true, "tbody": true, "td": true,
		"template": true, "textarea": true, "tfoot": true, "th": true, "thead": true,
		"time": true, "title": true, "tr": true, "track": true, "u": true,
		"ul": true, "var": true, "video": true, "wbr": true,
	}

	return htmlTags[strings.ToLower(tag)]
}