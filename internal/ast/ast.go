package ast

// Node is any top-level stylesheet rule.
type Node interface {
	// Location provides a mutable reference to the node's location.
	Location() *Span
}

// Span is a location in the source.
type Span struct {
	Start int
}

// Stylesheet is a CSS stylesheet.
type Stylesheet struct {
	Nodes []Node

	Imports []ImportSpecifier
}

// ImportSpecifier is a pointer to an import at rule.
type ImportSpecifier struct {
	Value string

	// AtRule is a pointer to the at rule that specified this import.
	AtRule *AtRule
}

// Location implements Node.
func (l Stylesheet) Location() *Span { return &Span{} }

// Comment represents a comment.
type Comment struct {
	Span

	Text string
}

// Location implements Node.
func (n *Comment) Location() *Span { return &n.Span }

// Block can either be a block of rules or declarations.
// See https://www.w3.org/TR/css-syntax-3/#declaration-rule-list.
type Block interface {
	Node

	isBlock()
}

// DeclarationBlock is a block containing a set of declarations.
type DeclarationBlock struct {
	Span

	Declarations []*Declaration
}

// Location implements Node.
func (n *DeclarationBlock) Location() *Span { return &n.Span }

// QualifiedRuleBlock is a block containing a set of rules.
type QualifiedRuleBlock struct {
	Span

	Rules []*QualifiedRule
}

// Location implements Node.
func (n *QualifiedRuleBlock) Location() *Span { return &n.Span }

func (DeclarationBlock) isBlock()   {}
func (QualifiedRuleBlock) isBlock() {}

var _ Block = &DeclarationBlock{}
var _ Block = &QualifiedRuleBlock{}

// Declaration is a property assignment, e.g. width: 2px.
type Declaration struct {
	Span

	// Property is the property being assigned.
	Property string

	// Values is the list of values assigned to the declaration.
	Values []Value

	// Important is whether or not the declaration was marked !important.
	Important bool
}

// Location implements Node.
func (n *Declaration) Location() *Span { return &n.Span }

var _ Node = &Declaration{}
