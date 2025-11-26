// Package optimizer provides optimization passes for GoX compilation
package optimizer

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/user/gox/pkg/analyzer"
)

// Options controls optimization behavior
type Options struct {
	MinifyHTML        bool
	MinifyCSS         bool
	RemoveComments    bool
	InlineSmallAssets bool
	TreeShaking       bool
	DeadCodeElim      bool
	BundleStyles      bool
	Production        bool
}

// DefaultOptions returns default optimization options
func DefaultOptions() *Options {
	return &Options{
		MinifyHTML:        false,
		MinifyCSS:         false,
		RemoveComments:    false,
		InlineSmallAssets: false,
		TreeShaking:       false,
		DeadCodeElim:      false,
		BundleStyles:      false,
		Production:        false,
	}
}

// ProductionOptions returns production optimization options
func ProductionOptions() *Options {
	return &Options{
		MinifyHTML:        true,
		MinifyCSS:         true,
		RemoveComments:    true,
		InlineSmallAssets: true,
		TreeShaking:       true,
		DeadCodeElim:      true,
		BundleStyles:      true,
		Production:        true,
	}
}

// Optimizer performs optimization passes
type Optimizer struct {
	options *Options
	styles  []string
}

// New creates a new optimizer
func New(opts *Options) *Optimizer {
	if opts == nil {
		opts = DefaultOptions()
	}
	return &Optimizer{
		options: opts,
		styles:  []string{},
	}
}

// OptimizeComponent optimizes a component IR
func (o *Optimizer) OptimizeComponent(ir *analyzer.ComponentIR) *analyzer.ComponentIR {
	if !o.options.Production {
		return ir
	}

	// Dead code elimination
	if o.options.DeadCodeElim {
		ir = o.eliminateDeadCode(ir)
	}

	// Tree shaking - remove unused hooks
	if o.options.TreeShaking {
		ir = o.treeShakeHooks(ir)
	}

	// Optimize render logic
	if ir.RenderLogic != nil {
		ir.RenderLogic = o.optimizeRenderLogic(ir.RenderLogic)
	}

	// Optimize styles
	if ir.Styles != nil && o.options.MinifyCSS {
		ir.Styles = o.optimizeStyles(ir.Styles)
	}

	return ir
}

// OptimizeHTML optimizes HTML output
func (o *Optimizer) OptimizeHTML(html string) string {
	if !o.options.MinifyHTML {
		return html
	}

	// Remove unnecessary whitespace
	html = regexp.MustCompile(`\s+`).ReplaceAllString(html, " ")

	// Remove whitespace between tags
	html = regexp.MustCompile(`>\s+<`).ReplaceAllString(html, "><")

	// Remove comments
	if o.options.RemoveComments {
		html = regexp.MustCompile(`<!--.*?-->`).ReplaceAllString(html, "")
	}

	// Trim leading/trailing whitespace
	html = strings.TrimSpace(html)

	return html
}

// OptimizeCSS optimizes CSS output
func (o *Optimizer) OptimizeCSS(css string) string {
	if !o.options.MinifyCSS {
		return css
	}

	// Remove comments
	if o.options.RemoveComments {
		css = regexp.MustCompile(`/\*.*?\*/`).ReplaceAllString(css, "")
	}

	// Remove unnecessary whitespace
	css = regexp.MustCompile(`\s+`).ReplaceAllString(css, " ")

	// Remove whitespace around special characters
	css = regexp.MustCompile(`\s*([{}:;,])\s*`).ReplaceAllString(css, "$1")

	// Remove trailing semicolon before closing brace
	css = regexp.MustCompile(`;}`).ReplaceAllString(css, "}")

	// Remove units from zero values
	css = regexp.MustCompile(`:\s*0(px|em|%|rem)`).ReplaceAllString(css, ":0")

	// Shorten hex colors
	css = regexp.MustCompile(`#([0-9a-fA-F])\1([0-9a-fA-F])\2([0-9a-fA-F])\3`).
		ReplaceAllString(css, "#$1$2$3")

	return strings.TrimSpace(css)
}

// OptimizeGo optimizes generated Go code
func (o *Optimizer) OptimizeGo(code []byte) []byte {
	if !o.options.Production {
		return code
	}

	codeStr := string(code)

	// Remove debug statements
	if o.options.RemoveComments {
		codeStr = regexp.MustCompile(`fmt\.Printf\([^)]*\)`).ReplaceAllString(codeStr, "")
		codeStr = regexp.MustCompile(`fmt\.Println\([^)]*\)`).ReplaceAllString(codeStr, "")
		codeStr = regexp.MustCompile(`log\.Printf\([^)]*\)`).ReplaceAllString(codeStr, "")
		codeStr = regexp.MustCompile(`log\.Println\([^)]*\)`).ReplaceAllString(codeStr, "")
	}

	// Remove empty lines
	lines := strings.Split(codeStr, "\n")
	var optimized []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" || !o.options.Production {
			optimized = append(optimized, line)
		}
	}

	return []byte(strings.Join(optimized, "\n"))
}

// BundleStyles bundles all styles into a single CSS file
func (o *Optimizer) BundleStyles(components []*analyzer.ComponentIR) string {
	if !o.options.BundleStyles {
		return ""
	}

	var buf bytes.Buffer
	seen := make(map[string]bool)

	for _, comp := range components {
		if comp.Styles == nil {
			continue
		}

		for _, rule := range comp.Styles.Rules {
			// Generate CSS rule
			selector := rule.Selector
			if comp.Styles.Scoped {
				// Add component scope
				selector = fmt.Sprintf("[data-gox-%s] %s", strings.ToLower(comp.Name), selector)
			}

			// Check for duplicates
			ruleKey := fmt.Sprintf("%s{%v}", selector, rule.Properties)
			if seen[ruleKey] {
				continue
			}
			seen[ruleKey] = true

			// Write CSS rule
			buf.WriteString(selector)
			buf.WriteString(" {\n")

			for prop, value := range rule.Properties {
				buf.WriteString(fmt.Sprintf("  %s: %s;\n", prop, value))
			}

			buf.WriteString("}\n\n")
		}
	}

	css := buf.String()
	if o.options.MinifyCSS {
		css = o.OptimizeCSS(css)
	}

	return css
}

// eliminateDeadCode removes unused code
func (o *Optimizer) eliminateDeadCode(ir *analyzer.ComponentIR) *analyzer.ComponentIR {
	// Remove unused state that's never referenced in render
	usedState := make(map[string]bool)

	// Scan render logic for state references
	if ir.RenderLogic != nil {
		o.scanForStateUsage(ir.RenderLogic.VNode, usedState)
	}

	// Filter state
	var optimizedState []analyzer.StateVar
	for _, state := range ir.State {
		if usedState[state.Name] || !o.options.Production {
			optimizedState = append(optimizedState, state)
		}
	}
	ir.State = optimizedState

	// Remove unused effects
	var optimizedEffects []analyzer.EffectHook
	for _, effect := range ir.Effects {
		// Keep effects that have dependencies on used state
		keep := false
		for _, dep := range effect.Deps {
			if usedState[dep] {
				keep = true
				break
			}
		}
		if keep || !o.options.Production {
			optimizedEffects = append(optimizedEffects, effect)
		}
	}
	ir.Effects = optimizedEffects

	return ir
}

// scanForStateUsage scans VNode tree for state references
func (o *Optimizer) scanForStateUsage(vnode *analyzer.VNodeIR, used map[string]bool) {
	if vnode == nil {
		return
	}

	// Check props for state references
	for _, expr := range vnode.Props {
		if expr.Type == analyzer.IRExprIdentifier {
			if name, ok := expr.Value.(string); ok {
				used[name] = true
			}
		}
	}

	// Scan children
	for _, child := range vnode.Children {
		o.scanForStateUsage(child, used)
	}
}

// treeShakeHooks removes unused hook imports
func (o *Optimizer) treeShakeHooks(ir *analyzer.ComponentIR) *analyzer.ComponentIR {
	// Track which hooks are actually used
	usedHooks := make(map[string]bool)

	if len(ir.State) > 0 {
		usedHooks["UseState"] = true
	}
	if len(ir.Effects) > 0 {
		usedHooks["UseEffect"] = true
	}
	if len(ir.Memos) > 0 {
		usedHooks["UseMemo"] = true
	}
	if len(ir.Callbacks) > 0 {
		usedHooks["UseCallback"] = true
	}
	if len(ir.Refs) > 0 {
		usedHooks["UseRef"] = true
	}

	// Used hooks info could be returned separately if needed by transpiler
	// For now just return the IR as-is
	return ir
}

// optimizeRenderLogic optimizes the render tree
func (o *Optimizer) optimizeRenderLogic(render *analyzer.RenderIR) *analyzer.RenderIR {
	if render.VNode != nil {
		render.VNode = o.optimizeVNode(render.VNode)
	}
	return render
}

// optimizeVNode optimizes a VNode tree
func (o *Optimizer) optimizeVNode(vnode *analyzer.VNodeIR) *analyzer.VNodeIR {
	if vnode == nil {
		return nil
	}

	// Collapse consecutive text nodes
	if len(vnode.Children) > 1 {
		var optimizedChildren []*analyzer.VNodeIR
		var currentText *analyzer.VNodeIR

		for _, child := range vnode.Children {
			if child.Type == analyzer.VNodeText {
				if currentText == nil {
					currentText = child
					optimizedChildren = append(optimizedChildren, currentText)
				} else {
					// Merge text nodes
					currentText.Tag += child.Tag
				}
			} else {
				currentText = nil
				optimizedChildren = append(optimizedChildren, o.optimizeVNode(child))
			}
		}

		vnode.Children = optimizedChildren
	}

	// Optimize children recursively
	for i, child := range vnode.Children {
		vnode.Children[i] = o.optimizeVNode(child)
	}

	return vnode
}

// optimizeStyles optimizes component styles
func (o *Optimizer) optimizeStyles(styles *analyzer.StyleIR) *analyzer.StyleIR {
	// Remove duplicate properties
	for i, rule := range styles.Rules {
		seen := make(map[string]bool)
		optimizedProps := make(map[string]string)

		// Keep only last occurrence of each property
		for prop, value := range rule.Properties {
			optimizedProps[prop] = value
			seen[prop] = true
		}

		styles.Rules[i].Properties = optimizedProps
	}

	// Merge rules with same selector
	ruleMap := make(map[string]*analyzer.CSSRuleIR)
	for _, rule := range styles.Rules {
		if existing, ok := ruleMap[rule.Selector]; ok {
			// Merge properties
			for prop, value := range rule.Properties {
				existing.Properties[prop] = value
			}
		} else {
			ruleMap[rule.Selector] = &rule
		}
	}

	// Convert back to array
	var optimizedRules []analyzer.CSSRuleIR
	for _, rule := range ruleMap {
		optimizedRules = append(optimizedRules, *rule)
	}
	styles.Rules = optimizedRules

	return styles
}