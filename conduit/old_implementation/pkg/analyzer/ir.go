// Package analyzer provides semantic analysis for GoX components
package analyzer

import (
	"go/ast"

	"github.com/user/gox/pkg/parser"
)

// ComponentIR represents the analyzed and normalized component structure
type ComponentIR struct {
	Name         string
	Props        []PropField
	State        []StateVar
	Effects      []EffectHook
	Memos        []MemoHook
	Refs         []RefHook
	Callbacks    []CallbackHook
	Context      []ContextHook
	CustomHooks  []CustomHookCall
	RenderLogic  *RenderIR
	Styles       *StyleIR
	Dependencies []string // Other components used
}

// PropField represents a component prop
type PropField struct {
	Name     string
	Type     ast.Expr
	Optional bool
	Default  ast.Expr
}

// StateVar represents a state variable created by useState
type StateVar struct {
	Name      string
	Type      ast.Expr
	InitValue ast.Expr
	Setter    string // setState function name
}

// EffectHook represents a useEffect hook
type EffectHook struct {
	Setup   *ast.BlockStmt
	Cleanup *ast.BlockStmt
	Deps    []string // Dependency list
}

// MemoHook represents a useMemo hook
type MemoHook struct {
	Name    string
	Type    ast.Expr
	Compute ast.Expr
	Deps    []string
}

// RefHook represents a useRef hook
type RefHook struct {
	Name      string
	Type      ast.Expr
	InitValue ast.Expr
}

// CallbackHook represents a useCallback hook
type CallbackHook struct {
	Name     string
	Callback *ast.FuncLit
	Deps     []string
}

// ContextHook represents a useContext hook
type ContextHook struct {
	Name        string
	Type        ast.Expr
	ContextName string
}

// CustomHookCall represents a call to a custom hook
type CustomHookCall struct {
	Name    string
	Args    []ast.Expr
	Results []string
}

// RenderIR represents the analyzed render logic
type RenderIR struct {
	VNode *VNodeIR
}

// VNodeIR represents a virtual node in the IR
type VNodeIR struct {
	Type      VNodeType
	Tag       string // HTML tag or component name
	Props     map[string]IRExpr
	Children  []*VNodeIR
	Key       IRExpr
	Ref       IRExpr
	Condition IRExpr  // for conditional rendering
	Loop      *LoopIR // for list rendering
}

// VNodeType represents the type of virtual node
type VNodeType int

const (
	VNodeElement VNodeType = iota
	VNodeComponent
	VNodeText
	VNodeExpression
	VNodeFragment
	VNodeConditional
	VNodeList
)

// LoopIR represents list rendering logic
type LoopIR struct {
	Item       string
	Index      string
	Collection IRExpr
	Body       *VNodeIR
}

// IRExpr represents an expression in the IR
type IRExpr struct {
	Type     IRExprType
	Value    interface{}
	Original ast.Expr
}

// IRExprType represents the type of IR expression
type IRExprType int

const (
	IRExprLiteral IRExprType = iota
	IRExprIdentifier
	IRExprBinary
	IRExprCall
	IRExprMember
	IRExprFunction
	IRExprArray
	IRExprObject
)

// StyleIR represents analyzed styles
type StyleIR struct {
	Scoped      bool
	Rules       []CSSRuleIR
	ComponentID string // For scoped styles
}

// CSSRuleIR represents a CSS rule in the IR
type CSSRuleIR struct {
	Selector   string
	Properties map[string]string
	Scoped     bool
}

// AnalysisResult contains the complete analysis output
type AnalysisResult struct {
	Components map[string]*ComponentIR
	Errors     []error
	Warnings   []string
}

// NewComponentIR creates a new ComponentIR instance
func NewComponentIR(name string) *ComponentIR {
	return &ComponentIR{
		Name:         name,
		Props:        []PropField{},
		State:        []StateVar{},
		Effects:      []EffectHook{},
		Memos:        []MemoHook{},
		Refs:         []RefHook{},
		Callbacks:    []CallbackHook{},
		Context:      []ContextHook{},
		CustomHooks:  []CustomHookCall{},
		Dependencies: []string{},
	}
}

// JSXNodeToVNodeIR converts a parser JSX node to a VNode IR
func JSXNodeToVNodeIR(node parser.JSXNode) *VNodeIR {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *parser.JSXElement:
		vnode := &VNodeIR{
			Tag:      n.Tag,
			Props:    make(map[string]IRExpr),
			Children: []*VNodeIR{},
		}

		// Determine if it's an HTML element or component
		if len(n.Tag) > 0 && n.Tag[0] >= 'A' && n.Tag[0] <= 'Z' {
			vnode.Type = VNodeComponent
		} else {
			vnode.Type = VNodeElement
		}

		// Convert attributes
		for _, attr := range n.Attrs {
			vnode.Props[attr.Name] = JSXAttrValueToIRExpr(attr.Value)
		}

		// Convert children
		for _, child := range n.Children {
			childVNode := JSXNodeToVNodeIR(child)
			if childVNode != nil {
				vnode.Children = append(vnode.Children, childVNode)
			}
		}

		return vnode

	case *parser.JSXText:
		return &VNodeIR{
			Type: VNodeText,
			Tag:  n.Value,
		}

	case *parser.JSXExpression:
		return &VNodeIR{
			Type: VNodeExpression,
			Props: map[string]IRExpr{
				"expr": {
					Type:     IRExprIdentifier,
					Original: n.Expr,
				},
			},
		}

	case *parser.JSXFragment:
		vnode := &VNodeIR{
			Type:     VNodeFragment,
			Children: []*VNodeIR{},
		}
		for _, child := range n.Children {
			childVNode := JSXNodeToVNodeIR(child)
			if childVNode != nil {
				vnode.Children = append(vnode.Children, childVNode)
			}
		}
		return vnode

	case *parser.JSXConditional:
		vnode := &VNodeIR{
			Type: VNodeConditional,
			Condition: IRExpr{
				Type:     IRExprIdentifier,
				Original: n.Condition,
			},
		}
		if n.Then != nil {
			vnode.Children = append(vnode.Children, JSXNodeToVNodeIR(n.Then))
		}
		if n.Else != nil {
			vnode.Children = append(vnode.Children, JSXNodeToVNodeIR(n.Else))
		}
		return vnode

	case *parser.JSXList:
		return &VNodeIR{
			Type: VNodeList,
			Loop: &LoopIR{
				Item:  n.Iterator,
				Index: n.Index,
				Collection: IRExpr{
					Type:     IRExprIdentifier,
					Original: n.Items,
				},
				Body: JSXNodeToVNodeIR(n.Body),
			},
		}

	default:
		return nil
	}
}

// JSXAttrValueToIRExpr converts a JSX attribute value to an IR expression
func JSXAttrValueToIRExpr(value parser.JSXAttrValue) IRExpr {
	if value == nil {
		return IRExpr{Type: IRExprLiteral, Value: true} // boolean attribute
	}

	switch v := value.(type) {
	case parser.JSXText:
		return IRExpr{
			Type:  IRExprLiteral,
			Value: v.Value,
		}
	case parser.JSXExpression:
		return IRExpr{
			Type:     IRExprIdentifier,
			Original: v.Expr,
		}
	default:
		return IRExpr{Type: IRExprLiteral, Value: nil}
	}
}