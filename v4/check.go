// Copyright 2022 The CC Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cc // import "modernc.org/cc/v4"

import (
	"math"
	"math/big"
	"math/bits"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"

	"modernc.org/mathutil"
)

const (
	// type check
	decay flags = 1 << iota
	asmArgList
	implicitFuncDef
	ignoreUndefined

	// eval
	addrOf
)

type flags int

func (f flags) add(g flags) flags { return f | g }
func (f flags) del(g flags) flags { return f &^ g }
func (f flags) has(g flags) bool  { return f&g != 0 }

type ExpressionNode interface {
	Node
	Type() Type
	Value() Value
	check(*ctx, flags) Type
	eval(*ctx, flags) Value
}

const longDoublePrec = 256 // mantissa bits

type ctx struct {
	ast          *AST
	builtinTypes map[string]Type
	errors       errors
	fnScope      *Scope
	implicitFunc Type
	int64T       Type
	intT         Type
	pcharT       Type
	ptrDiffT0    Type
	pvoidT       Type
	sizeT0       Type
	voidT        Type
	wcharT0      Type

	inLoop      int
	inSwitch    int
	switchCases int
}

func newCtx(ast *AST) *ctx {
	c := &ctx{ast: ast}
	complexdouble := c.newPredefinedType(ComplexDouble)
	complexlong := c.newPredefinedType(ComplexLong)
	int := c.newPredefinedType(Int)
	long := c.newPredefinedType(Long)
	longlong := c.newPredefinedType(LongLong)
	short := c.newPredefinedType(Short)
	uint := c.newPredefinedType(UInt)
	uint128 := c.newPredefinedType(UInt128)
	ulong := c.newPredefinedType(ULong)
	ulonglog := c.newPredefinedType(ULongLong)
	ushort := c.newPredefinedType(UShort)
	c.builtinTypes = map[string]Type{
		ts2String([]TypeSpecifierCase{TypeSpecifierBool}):                                                             c.newPredefinedType(Bool),
		ts2String([]TypeSpecifierCase{TypeSpecifierChar, TypeSpecifierSigned}):                                        c.newPredefinedType(SChar),
		ts2String([]TypeSpecifierCase{TypeSpecifierChar, TypeSpecifierUnsigned}):                                      c.newPredefinedType(UChar),
		ts2String([]TypeSpecifierCase{TypeSpecifierChar}):                                                             c.newPredefinedType(Char),
		ts2String([]TypeSpecifierCase{TypeSpecifierComplex, TypeSpecifierChar}):                                       c.newPredefinedType(ComplexChar),
		ts2String([]TypeSpecifierCase{TypeSpecifierComplex, TypeSpecifierDouble}):                                     complexdouble,
		ts2String([]TypeSpecifierCase{TypeSpecifierComplex, TypeSpecifierFloat}):                                      c.newPredefinedType(ComplexFloat),
		ts2String([]TypeSpecifierCase{TypeSpecifierComplex, TypeSpecifierInt, TypeSpecifierLong}):                     complexlong,
		ts2String([]TypeSpecifierCase{TypeSpecifierComplex, TypeSpecifierInt}):                                        c.newPredefinedType(ComplexInt),
		ts2String([]TypeSpecifierCase{TypeSpecifierComplex, TypeSpecifierLong, TypeSpecifierDouble}):                  c.newPredefinedType(ComplexLongDouble),
		ts2String([]TypeSpecifierCase{TypeSpecifierComplex, TypeSpecifierLong, TypeSpecifierLong}):                    c.newPredefinedType(ComplexLongLong),
		ts2String([]TypeSpecifierCase{TypeSpecifierComplex, TypeSpecifierLong}):                                       complexlong,
		ts2String([]TypeSpecifierCase{TypeSpecifierComplex, TypeSpecifierShort, TypeSpecifierUnsigned}):               c.newPredefinedType(ComplexUShort),
		ts2String([]TypeSpecifierCase{TypeSpecifierComplex, TypeSpecifierShort}):                                      c.newPredefinedType(ComplexShort),
		ts2String([]TypeSpecifierCase{TypeSpecifierComplex, TypeSpecifierUnsigned}):                                   c.newPredefinedType(ComplexUInt),
		ts2String([]TypeSpecifierCase{TypeSpecifierComplex}):                                                          complexdouble,
		ts2String([]TypeSpecifierCase{TypeSpecifierDecimal128}):                                                       c.newPredefinedType(Decimal128),
		ts2String([]TypeSpecifierCase{TypeSpecifierDecimal32}):                                                        c.newPredefinedType(Decimal32),
		ts2String([]TypeSpecifierCase{TypeSpecifierDecimal64}):                                                        c.newPredefinedType(Decimal64),
		ts2String([]TypeSpecifierCase{TypeSpecifierDouble, TypeSpecifierLong}):                                        c.newPredefinedType(LongDouble),
		ts2String([]TypeSpecifierCase{TypeSpecifierDouble}):                                                           c.newPredefinedType(Double),
		ts2String([]TypeSpecifierCase{TypeSpecifierFloat128}):                                                         c.newPredefinedType(Float128),
		ts2String([]TypeSpecifierCase{TypeSpecifierFloat128x}):                                                        c.newPredefinedType(Float128x),
		ts2String([]TypeSpecifierCase{TypeSpecifierFloat16}):                                                          c.newPredefinedType(Float16),
		ts2String([]TypeSpecifierCase{TypeSpecifierFloat32x}):                                                         c.newPredefinedType(Float32x),
		ts2String([]TypeSpecifierCase{TypeSpecifierFloat32}):                                                          c.newPredefinedType(Float32),
		ts2String([]TypeSpecifierCase{TypeSpecifierFloat64x}):                                                         c.newPredefinedType(Float64x),
		ts2String([]TypeSpecifierCase{TypeSpecifierFloat64}):                                                          c.newPredefinedType(Float64),
		ts2String([]TypeSpecifierCase{TypeSpecifierFloat}):                                                            c.newPredefinedType(Float),
		ts2String([]TypeSpecifierCase{TypeSpecifierInt, TypeSpecifierLong, TypeSpecifierLong, TypeSpecifierSigned}):   longlong,
		ts2String([]TypeSpecifierCase{TypeSpecifierInt, TypeSpecifierLong, TypeSpecifierLong, TypeSpecifierUnsigned}): ulonglog,
		ts2String([]TypeSpecifierCase{TypeSpecifierInt, TypeSpecifierLong, TypeSpecifierLong}):                        longlong,
		ts2String([]TypeSpecifierCase{TypeSpecifierInt, TypeSpecifierLong, TypeSpecifierSigned}):                      long,
		ts2String([]TypeSpecifierCase{TypeSpecifierInt, TypeSpecifierLong, TypeSpecifierUnsigned}):                    ulong,
		ts2String([]TypeSpecifierCase{TypeSpecifierInt, TypeSpecifierLong}):                                           long,
		ts2String([]TypeSpecifierCase{TypeSpecifierInt, TypeSpecifierShort, TypeSpecifierSigned}):                     short,
		ts2String([]TypeSpecifierCase{TypeSpecifierInt, TypeSpecifierShort, TypeSpecifierUnsigned}):                   ushort,
		ts2String([]TypeSpecifierCase{TypeSpecifierInt, TypeSpecifierShort}):                                          short,
		ts2String([]TypeSpecifierCase{TypeSpecifierInt, TypeSpecifierSigned}):                                         int,
		ts2String([]TypeSpecifierCase{TypeSpecifierInt, TypeSpecifierUnsigned}):                                       uint,
		ts2String([]TypeSpecifierCase{TypeSpecifierInt128, TypeSpecifierUnsigned}):                                    uint128,
		ts2String([]TypeSpecifierCase{TypeSpecifierInt128}):                                                           c.newPredefinedType(Int128),
		ts2String([]TypeSpecifierCase{TypeSpecifierInt}):                                                              int,
		ts2String([]TypeSpecifierCase{TypeSpecifierLong, TypeSpecifierLong, TypeSpecifierSigned}):                     longlong,
		ts2String([]TypeSpecifierCase{TypeSpecifierLong, TypeSpecifierLong, TypeSpecifierUnsigned}):                   ulonglog,
		ts2String([]TypeSpecifierCase{TypeSpecifierLong, TypeSpecifierLong}):                                          longlong,
		ts2String([]TypeSpecifierCase{TypeSpecifierLong, TypeSpecifierSigned}):                                        long,
		ts2String([]TypeSpecifierCase{TypeSpecifierLong, TypeSpecifierUnsigned}):                                      ulong,
		ts2String([]TypeSpecifierCase{TypeSpecifierLong}):                                                             long,
		ts2String([]TypeSpecifierCase{TypeSpecifierShort, TypeSpecifierSigned}):                                       short,
		ts2String([]TypeSpecifierCase{TypeSpecifierShort, TypeSpecifierUnsigned}):                                     ushort,
		ts2String([]TypeSpecifierCase{TypeSpecifierShort}):                                                            short,
		ts2String([]TypeSpecifierCase{TypeSpecifierSigned}):                                                           int,
		ts2String([]TypeSpecifierCase{TypeSpecifierUint128}):                                                          uint128,
		ts2String([]TypeSpecifierCase{TypeSpecifierUnsigned}):                                                         uint,
		ts2String([]TypeSpecifierCase{TypeSpecifierVoid}):                                                             c.newPredefinedType(Void),
	}
	c.ast.kinds = map[Kind]Type{}
	for _, v := range c.builtinTypes {
		c.ast.kinds[v.Kind()] = v
	}
	c.intT = c.ast.kinds[Int]
	c.int64T = c.ast.kinds[LongLong]
	c.pcharT = c.newPointerType(c.ast.kinds[Char])
	c.voidT = c.ast.kinds[Void]
	c.pvoidT = c.newPointerType(c.voidT)
	c.implicitFunc = c.newPointerType(c.newFunctionType(c.intT, nil, false))

	ast.Void = c.voidT
	return c
}

func (c *ctx) checkScope(s *Scope) {
	for _, ns := range s.Nodes {
		var (
			ds  []*Declarator
			es  []*Enumerator
			ess []*EnumSpecifier
			lds []*LabelDeclaration
			lss []*LabeledStatement
			ps  []*Parameter
			sus []*StructOrUnionSpecifier
		)
		for _, n := range ns {
			switch x := n.(type) {
			case *Declarator:
				ds = append(ds, x)
			case *Enumerator:
				es = append(es, x)
			case *EnumSpecifier:
				ess = append(ess, x)
			case *LabelDeclaration:
				lds = append(lds, x)
			case *LabeledStatement:
				lss = append(lss, x)
			case *Parameter:
				ps = append(ps, x)
			case *StructOrUnionSpecifier:
				sus = append(sus, x)
			default:
				c.errors.add(errorf("TODO %T", x))
			}
		}
		if len(ds) > 1 {
			a := ds[0]
			t := a.Type()
			switch {
			case s.Parent != nil:
				if t.Kind() != Function {
					c.errors.add(errorf("%v: redeclaration of '%s' at %v:", a.Position(), a.Name(), ds[1].Position()))
					break
				}

				fallthrough
			case s.Parent == nil:
				for _, b := range ds[1:] {
					u := b.Type()
					if !t.isCompatible(u) {
						c.errors.add(errorf("%v: conflicting types for '%s', previous declaration at %v, %s and %s", b.Position(), a.Name(), a.Position(), t, u))
						return
					}
				}

				switch {
				case t.Kind() == Function:
					f := t.(*FunctionType)
					if len(f.fp) != 0 {
						break
					}

					for _, b := range ds[1:] {
						if g := b.Type().(*FunctionType); len(g.fp) != 0 {
							t = g
							break
						}
					}
				case t.IsIncomplete():
					for _, b := range ds[1:] {
						if u := b.Type(); !u.IsIncomplete() {
							t = u
							break
						}
					}
				}
				for i := range ds {
					ds[i].typ = t
				}
			}
		}
		if len(ess) > 1 {
			a := ess[1]
			c.errors.add(errorf("%v: redeclaration of 'enum %s' at %v:", a.Position(), a.Token2.Src(), ess[0].Position()))
		}
		if len(es) > 1 {
			a := es[1]
			c.errors.add(errorf("%v: redeclaration of enumerator '%s' at %v:", a.Position(), a.Token.Src(), es[0].Position()))
		}
		if len(lds) > 1 {
			c.errors.add(errorf("TODO %T", lds[0]))
		}
		if len(lss) > 1 {
			c.errors.add(errorf("TODO %T", lss[0]))
		}
		if len(ps) > 1 {
			c.errors.add(errorf("TODO %T", lss[0]))
		}
		if len(sus) > 1 {
			c.errors.add(errorf("TODO %T", sus[0]))
		}
	}
}

func (c *ctx) convert(v Value, t Type) (r Value) {
	if v == nil || v == Unknown {
		return Unknown
	}

	if t.Kind() == Enum {
		t = t.(*EnumType).UnderlyingType()
	}
	if IsIntegerType(t) {
		switch {
		case t.Size() > 8:
			return Unknown
		case IsSignedInteger(t):
			m := Int64Value(-1)
			if t.Size() < 8 {
				m = Int64Value(1)<<(8*t.Size()) - 1
			}
			switch x := v.(type) {
			case Int64Value:
				switch {
				case x < 0:
					return x | ^m
				default:
					return x & m
				}
			case UInt64Value:
				switch y := Int64Value(x); {
				case y < 0:
					return y | ^m
				default:
					return y & m
				}
			}
		default:
			m := ^UInt64Value(0)
			if t.Size() < 8 {
				m = UInt64Value(1)<<(8*t.Size()) - 1
			}
			switch x := v.(type) {
			case Int64Value:
				return UInt64Value(x) & m
			case UInt64Value:
				return x & m
			}
		}
	}

	switch t.Kind() {
	case Bool:
		switch x := v.(type) {
		case Int64Value:
			if x != 0 {
				return int1
			}

			return int0
		case UInt64Value:
			if x != 0 {
				return int1
			}

			return int0
		}
	case Ptr:
		switch x := v.(type) {
		case Int64Value:
			return UInt64Value(x)
		case UInt64Value:
			return x
		}
	case Void:
		return VoidValue{}
	}
	return Unknown
}

func (c *ctx) decay(t Type, mode flags) Type {
	if !mode.has(decay) || t == nil {
		return t
	}

	return t.Decay()
}

func (c *ctx) wcharT(n Node) Type {
	if c.wcharT0 == nil {
		if s := c.ast.Scope.Nodes["wchar_t"]; len(s) != 0 {
			if d, ok := s[0].(*Declarator); ok && d.isTypename && d.Type() != Invalid {
				c.wcharT0 = d.Type()
			}
		}
		if c.wcharT0 == nil {
			if s := c.ast.Scope.Nodes["__predefined_wchar_t"]; len(s) != 0 {
				if d, ok := s[0].(*Declarator); ok && d.isTypename && d.Type() != Invalid {
					c.wcharT0 = d.Type()
				}
			}
		}
		if c.wcharT0 == nil {
			c.errors.add(errorf("%v: undefined type: wchar_t", n.Position()))
			c.wcharT0 = c.intT
		}
	}
	return c.wcharT0
}

func (c *ctx) ptrDiffT(n Node) Type {
	if c.ptrDiffT0 == nil {
		if s := c.ast.Scope.Nodes["ptrdiff_t"]; len(s) != 0 {
			if d, ok := s[0].(*Declarator); ok && d.isTypename && d.Type() != Invalid {
				c.ptrDiffT0 = d.Type()
			}
		}
		if c.ptrDiffT0 == nil {
			if s := c.ast.Scope.Nodes["__predefined_ptrdiff_t"]; len(s) != 0 {
				if d, ok := s[0].(*Declarator); ok && d.isTypename && d.Type() != Invalid {
					c.ptrDiffT0 = d.Type()
				}
			}
		}
		if c.ptrDiffT0 == nil {
			c.errors.add(errorf("%v: undefined type: ptrdiff_t", n.Position()))
			c.ptrDiffT0 = c.intT
		}
	}
	return c.ptrDiffT0
}

func (c *ctx) sizeT(n Node) Type {
	if c.sizeT0 == nil {
		if s := c.ast.Scope.Nodes["size_t"]; len(s) != 0 {
			if d, ok := s[0].(*Declarator); ok && d.isTypename && d.Type() != Invalid {
				c.sizeT0 = d.Type()
			}
		}
		if c.sizeT0 == nil {
			if s := c.ast.Scope.Nodes["__predefined_size_t"]; len(s) != 0 {
				if d, ok := s[0].(*Declarator); ok && d.isTypename && d.Type() != Invalid {
					c.sizeT0 = d.Type()
				}
			}
		}
		if c.sizeT0 == nil {
			c.errors.add(errorf("%v: undefined type: size_t", n.Position()))
			c.sizeT0 = c.intT
		}
	}
	return c.sizeT0
}

type resolver struct{ resolved *Scope }

// ResolvedIn returns the scope an identifier was resolved in, if any.
func (n resolver) ResolvedIn() *Scope { return n.resolved }

type AST struct {
	ABI             *ABI
	EOF             Token
	Macros          map[string]*Macro
	Scope           *Scope // File scope.
	TranslationUnit *TranslationUnit
	Void            Type // Valid only after Translate
	kinds           map[Kind]Type
}

func (n *AST) check() error {
	c := newCtx(n)
	for l := n.TranslationUnit; l != nil; l = l.TranslationUnit {
		l.ExternalDeclaration.check(c)
	}
	c.checkScope(n.Scope)
	return c.errors.err()
}

//  ExternalDeclaration:
//          FunctionDefinition  // Case ExternalDeclarationFuncDef
//  |       Declaration         // Case ExternalDeclarationDecl
//  |       AsmStatement        // Case ExternalDeclarationAsmStmt
//  |       ';'                 // Case ExternalDeclarationEmpty
func (n *ExternalDeclaration) check(c *ctx) {
	switch n.Case {
	case ExternalDeclarationFuncDef: // FunctionDefinition
		n.FunctionDefinition.check(c)
	case ExternalDeclarationDecl: // Declaration
		n.Declaration.check(c)
	case ExternalDeclarationAsmStmt: // AsmStatement
		n.AsmStatement.check(c)
	case ExternalDeclarationEmpty: // ';'
		c.errors.add(errorf("TODO %v", n.Case))
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
}

//  AsmStatement:
//          Asm ';'
func (n *AsmStatement) check(c *ctx) {
	n.Asm.check(c)
}

//  Asm:
//          "asm" AsmQualifierList '(' STRINGLITERAL AsmArgList ')'
func (n *Asm) check(c *ctx) {
	n.AsmQualifierList.check(c)
	n.AsmArgList.check(c)
}

//  AsmArgList:
//          ':' AsmExpressionList
//  |       AsmArgList ':' AsmExpressionList
func (n *AsmArgList) check(c *ctx) {
	for ; n != nil; n = n.AsmArgList {
		n.AsmExpressionList.check(c)
	}
}

//  AsmExpressionList:
//          AsmIndex AssignmentExpression
//  |       AsmExpressionList ',' AsmIndex AssignmentExpression
func (n *AsmExpressionList) check(c *ctx) {
	for ; n != nil; n = n.AsmExpressionList {
		n.AsmIndex.check(c)
		n.AssignmentExpression.check(c, decay|asmArgList|ignoreUndefined)
		n.AssignmentExpression.eval(c, decay|ignoreUndefined)
	}
}

//  AsmIndex:
//          '[' ExpressionList ']'
func (n *AsmIndex) check(c *ctx) {
	if n == nil {
		return
	}

	n.ExpressionList.check(c, decay|asmArgList|ignoreUndefined)
	n.ExpressionList.eval(c, decay|ignoreUndefined)
}

func (n *AsmQualifierList) check(c *ctx) {
	for ; n != nil; n = n.AsmQualifierList {
		n.AsmQualifier.check(c)
	}
}

//  AsmQualifierList:
//          AsmQualifier
//  |       AsmQualifierList AsmQualifier
func (n *AsmQualifier) check(c *ctx) {
	switch n.Case {
	case AsmQualifierVolatile: // "volatile"
		//TODO c.errors.add(errorf("TODO %v", n.Case))
	case AsmQualifierInline: // "inline"
		c.errors.add(errorf("TODO %v", n.Case))
	case AsmQualifierGoto: // "goto"
		//TODO c.errors.add(errorf("TODO %v", n.Case))
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
}

//  FunctionDefinition:
//          DeclarationSpecifiers Declarator DeclarationList CompoundStatement
func (n *FunctionDefinition) check(c *ctx) {
	d := n.Declarator
	d.check(c, n.DeclarationSpecifiers.check(c, &d.isExtern, &d.isStatic, &d.isAtomic, &d.isThreadLocal, &d.isConst, &d.isVolatile, &d.isInline, &d.isRegister, &d.isAuto, &d.isNoreturn, &d.isRestrict, &d.alignas))
	if x, ok := d.Type().(*FunctionType); ok {
		x.hasImplicitResult = true
	}
	switch d.DirectDeclarator.Case {
	case DirectDeclaratorFuncIdent:
		ft, ok := d.Type().(*FunctionType)
		if !ok {
			break
		}

		m := n.DeclarationList.check(c)
		for _, param := range d.DirectDeclarator.IdentifierList.parameters {
			switch d := m[param.name.SrcStr()]; {
			case d == nil:
				param.typ = c.intT
			default:
				param.Declarator = d
				param.typ = d.Type()
			}
			ft.fp = append(ft.fp, param)
		}
		ft2 := c.newFunctionType2(ft.Result(), ft.fp)
		ft2.hasImplicitResult = ft.hasImplicitResult
		d.typ = ft2
	}
	c.fnScope = n.scope
	defer func() { c.fnScope = nil }()
	n.CompoundStatement.check(c)
}

//  DeclarationList:
//          Declaration
//  |       DeclarationList Declaration
func (n *DeclarationList) check(c *ctx) (m map[string]*Declarator) {
	for ; n != nil; n = n.DeclarationList {
		n.Declaration.check(c)
		if m == nil {
			m = map[string]*Declarator{}
		}
		for l := n.Declaration.InitDeclaratorList; l != nil; l = l.InitDeclaratorList {
			d := l.InitDeclarator.Declarator
			nm := d.Name()
			if x := m[nm]; x != nil {
				c.errors.add(errorf("%v: %s redeclared, previous declaration at %v:", d.Position(), nm, x.Position()))
				continue
			}

			m[nm] = d
		}
	}
	return m
}

//  CompoundStatement:
//          '{' LabelDeclarationList BlockItemList '}'
func (n *CompoundStatement) check(c *ctx) (r Type) {
	if n == nil {
		return
	}

	r = c.voidT
	for l := n.BlockItemList; l != nil; l = l.BlockItemList {
		r = l.BlockItem.check(c)
	}
	c.checkScope(n.LexicalScope())
	return r
}

func (n *BlockItem) check(c *ctx) (r Type) {
	if n == nil {
		return Invalid
	}

	switch n.Case {
	case BlockItemDecl: // Declaration
		n.Declaration.check(c)
	case BlockItemLabel:
		n.LabelDeclaration.check(c)
	case BlockItemStmt: // Statement
		return n.Statement.check(c)
	case BlockItemFuncDef: // DeclarationSpecifiers Declarator CompoundStatement
		var isExtern, isStatic, isAtomic, isThreadLocal, isConst, isVolatile, isInline, isRegister, isAuto, isNoreturn, isRestrict bool
		var alignas int
		n.Declarator.check(c, n.DeclarationSpecifiers.check(c, &isExtern, &isStatic, &isAtomic, &isThreadLocal, &isConst, &isVolatile, &isInline, &isRegister, &isAuto, &isNoreturn, &isRestrict, &alignas))
		if isExtern || isStatic || isAtomic || isThreadLocal || isConst || isVolatile || isRegister || isAuto || isRestrict || alignas != 0 {
			c.errors.add(errorf("%v: invalid specifier/qualifer combination", n.Position()))
		}
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return nil
}

func (n *LabelDeclaration) check(c *ctx) {
	//TODO
}

func (n *Statement) check(c *ctx) (r Type) {
	if n == nil {
		return Invalid
	}

	switch n.Case {
	case StatementLabeled: // LabeledStatement
		return n.LabeledStatement.check(c)
	case StatementCompound: // CompoundStatement
		return n.CompoundStatement.check(c)
	case StatementExpr: // ExpressionStatement
		return n.ExpressionStatement.check(c)
	case StatementSelection: // SelectionStatement
		return n.SelectionStatement.check(c)
	case StatementIteration: // IterationStatement
		return n.IterationStatement.check(c)
	case StatementJump: // JumpStatement
		return n.JumpStatement.check(c)
	case StatementAsm: // AsmStatement
		n.AsmStatement.check(c)
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return Invalid
}

func (n *LabeledStatement) check(c *ctx) (r Type) {
	if n == nil {
		return nil
	}

	switch n.Case {
	case LabeledStatementLabel: // IDENTIFIER ':' Statement
		return n.Statement.check(c)
	case LabeledStatementCaseLabel: // "case" ConstantExpression ':' Statement
		if c.inSwitch == 0 {
			c.errors.add(errorf("%v: case label not within a switch statement", n.Position()))
		}
		n.caseOrdinal = c.switchCases
		c.switchCases++
		n.ConstantExpression.check(c, decay)
		return n.Statement.check(c)
	case LabeledStatementRange: // "case" ConstantExpression "..." ConstantExpression ':' Statement
		if c.inSwitch == 0 {
			c.errors.add(errorf("%v: case label not within a switch statement", n.Position()))
		}
		n.caseOrdinal = c.switchCases
		c.switchCases++
		n.ConstantExpression.check(c, decay)
		n.ConstantExpression2.check(c, decay)
		return n.Statement.check(c)
	case LabeledStatementDefault: // "default" ':' Statement
		if c.inSwitch == 0 {
			c.errors.add(errorf("%v: default label not within a switch statement", n.Position()))
		}
		n.caseOrdinal = c.switchCases
		c.switchCases++
		return n.Statement.check(c)
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return nil
}

func (n *IterationStatement) check(c *ctx) (r Type) {
	if n == nil {
		return nil
	}

	switch n.Case {
	case IterationStatementWhile: // "while" '(' ExpressionList ')' Statement
		n.ExpressionList.check(c, decay)
		n.ExpressionList.eval(c, decay)
		c.inLoop++
		defer func() { c.inLoop-- }()
		return n.Statement.check(c)
	case IterationStatementDo: // "do" Statement "while" '(' ExpressionList ')' ';'
		c.inLoop++
		defer func() { c.inLoop-- }()
		r = n.Statement.check(c)
		n.ExpressionList.check(c, decay)
		n.ExpressionList.eval(c, decay)
		return r
	case IterationStatementFor: // "for" '(' ExpressionList ';' ExpressionList ';' ExpressionList ')' Statement
		n.ExpressionList.check(c, decay)
		n.ExpressionList.eval(c, decay)
		n.ExpressionList2.check(c, decay)
		n.ExpressionList2.eval(c, decay)
		n.ExpressionList3.check(c, decay)
		n.ExpressionList3.eval(c, decay)
		c.inLoop++
		defer func() { c.inLoop-- }()
		return n.Statement.check(c)
	case IterationStatementForDecl: // "for" '(' Declaration ExpressionList ';' ExpressionList ')' Statement
		n.Declaration.check(c)
		n.ExpressionList.check(c, decay)
		n.ExpressionList.eval(c, decay)
		n.ExpressionList2.check(c, decay)
		n.ExpressionList2.eval(c, decay)
		c.inLoop++
		defer func() { c.inLoop-- }()
		return n.Statement.check(c)
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return nil
}

func (n *JumpStatement) check(c *ctx) (r Type) {
	if n == nil {
		return
	}

out:
	switch n.Case {
	case JumpStatementGoto: // "goto" IDENTIFIER ';'
		for _, nd := range c.fnScope.Nodes[string(n.Token2.Src())] {
			switch nd.(type) {
			case *LabeledStatement:
				break out
			}
		}

		c.errors.add(errorf("%v: undefined label: %s", n.Token2.Position(), n.Token2.Src()))
	case JumpStatementGotoExpr: // "goto" '*' ExpressionList ';'
		n.ExpressionList.check(c, decay)
		n.ExpressionList.eval(c, decay)
	case JumpStatementContinue: // "continue" ';'
		if c.inLoop == 0 {
			c.errors.add(errorf("%v: continue statement not within a loop", n.Position()))
		}
	case JumpStatementBreak: // "break" ';'
		if c.inLoop+c.inSwitch == 0 {
			c.errors.add(errorf("%v: break statement not within loop or switch", n.Position()))
		}
	case JumpStatementReturn: // "return" ExpressionList ';'
		r = n.ExpressionList.check(c, decay)
		n.ExpressionList.eval(c, decay)
		//TODO check assignable to fn result
		return r
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return c.voidT
}

func (n *SelectionStatement) check(c *ctx) (r Type) {
	if n == nil {
		return nil
	}

	switch n.Case {
	case SelectionStatementIf: // "if" '(' ExpressionList ')' Statement
		n.ExpressionList.check(c, decay)
		n.ExpressionList.eval(c, decay)
		return n.Statement.check(c)
	case SelectionStatementIfElse: // "if" '(' ExpressionList ')' Statement "else" Statement
		n.ExpressionList.check(c, decay)
		n.ExpressionList.eval(c, decay)
		r1 := n.Statement.check(c)
		r2 := n.Statement2.check(c)
		if r1 != nil && r1 != Invalid {
			return r1
		}

		return r2
	case SelectionStatementSwitch: // "switch" '(' ExpressionList ')' Statement
		n.ExpressionList.check(c, decay)
		n.ExpressionList.eval(c, decay)
		c.inSwitch++
		switchCases := c.switchCases
		c.switchCases = 0
		defer func() {
			c.inSwitch--
			n.switchCases = c.switchCases
			c.switchCases = switchCases
		}()
		return n.Statement.check(c)
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return nil
}

func (n *ExpressionStatement) check(c *ctx) (r Type) {
	if n == nil {
		return nil
	}

	if n.ExpressionList == nil {
		return c.voidT
	}

	r = n.ExpressionList.check(c, decay)
	n.ExpressionList.eval(c, decay)
	return r
}

func (n *Declaration) check(c *ctx) {
	switch n.Case {
	case DeclarationDecl: // DeclarationSpecifiers InitDeclaratorList AttributeSpecifierList ';'
		var isExtern, isStatic, isAtomic, isThreadLocal, isConst, isVolatile, isInline, isRegister, isAuto, isNoreturn, isRestrict bool
		var alignas int
		t := n.DeclarationSpecifiers.check(c, &isExtern, &isStatic, &isAtomic, &isThreadLocal, &isConst, &isVolatile, &isInline, &isRegister, &isAuto, &isNoreturn, &isRestrict, &alignas)
		var attr *Attributes
		if n.InitDeclaratorList != nil && n.InitDeclaratorList.InitDeclaratorList == nil {
			if attr = n.InitDeclaratorList.InitDeclarator.AttributeSpecifierList.check(c); attr != nil {
				t = t.setAttr(attr)
			}
		}
		for l := n.InitDeclaratorList; l != nil; l = l.InitDeclaratorList {
			l.InitDeclarator.check(c, t, isExtern, isStatic, isAtomic, isThreadLocal, isConst, isVolatile, isInline, isRegister, isAuto, alignas)
		}
	case DeclarationAssert: // StaticAssertDeclaration
		n.StaticAssertDeclaration.check(c)
	case DeclarationAuto: // "__auto_type" Declarator '=' Initializer ';'
		if n.Initializer.Case != InitializerExpr {
			c.errors.add(errorf("%v: expected assignment expression", n.Initializer.Position()))
			break
		}

		n.Declarator.typ = n.Initializer.AssignmentExpression.check(c, decay)
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
}

//  StaticAssertDeclaration:
//          "_Static_assert" '(' ConstantExpression ',' STRINGLITERAL ')'
func (n *StaticAssertDeclaration) check(c *ctx) {
	n.ConstantExpression.check(c, decay)
	if !isNonzero(n.ConstantExpression.Value()) {
		s := stringConst(func(msg string, args ...interface{}) {
			c.errors.add(errorf(msg, args...))
		}, n.Token4)
		c.errors.add(errorf("%v: assertion failed: %s", n.ConstantExpression.Position(), s[:len(s)-1]))
	}
}

//  InitDeclarator:
//          Declarator Asm                  // Case InitDeclaratorDecl
//  |       Declarator Asm '=' Initializer  // Case InitDeclaratorInit
func (n *InitDeclarator) check(c *ctx, t Type, isExtern, isStatic, isAtomic, isThreadLocal, isConst, isVolatile, isInline, isRegister, isAuto bool, alignas int) {
	if t == nil {
		c.errors.add(errorf("TODO %T internal error", n))
		return
	}

	if n == nil {
		return
	}

	n.Declarator.isAtomic = isAtomic
	n.Declarator.isAuto = isAuto
	n.Declarator.isConst = isConst
	n.Declarator.isExtern = isExtern
	n.Declarator.isInline = isInline
	n.Declarator.isRegister = isRegister
	n.Declarator.isStatic = isStatic
	n.Declarator.isThreadLocal = isThreadLocal
	n.Declarator.isVolatile = isVolatile
	n.Declarator.alignas = alignas
	t = n.Declarator.check(c, t)
	if n.Asm != nil {
		n.Asm.check(c)
		return
	}

	if n.Case == InitDeclaratorInit {
		n.Initializer.check(c, t, 0)
	}
}

//  Initializer:
//          AssignmentExpression         // Case InitializerExpr
//  |       '{' InitializerList ',' '}'  // Case InitializerInitList
func (n *Initializer) check(c *ctx, t Type, off int64) {
	if n == nil || t == nil {
		c.errors.add(errorf("internal error %T(%v) %T(%v)", n, n == nil, t, t == nil))
		return
	}

	// The type of the entity to be initialized shall be an array of unknown size
	// or an object type that is not a variable length array type.
	n.typ = t
	n.off = off
	t = n.Type()
	if x, ok := t.(*ArrayType); ok && x.IsVLA() {
		c.errors.add(errorf("%v: cannot initalize a variable length array", n.Position()))
		return
	}

	switch n.Case {
	case InitializerExpr: // AssignmentExpression
		exprT := n.AssignmentExpression.check(c, decay)
		if exprT == Invalid {
			n.val = Unknown
			c.errors.add(errorf("TODO %T <- %T", t, exprT))
			return
		}

		n.val = n.AssignmentExpression.eval(c, decay)
		switch x := t.(type) {
		case *ArrayType:
			n.checkExprArray(c, x, exprT, off)
		case *EnumType:
			n.val = c.convert(n.Value(), t)
		case *PointerType:
			if isPointerType(exprT) {
				return
			}

			if IsIntegerType(exprT) {
				n.val = c.convert(n.Value(), t)
				return
			}

			c.errors.add(errorf("TODO %T <- %T %T", x, n.Value(), exprT))
		case *PredefinedType:
			n.val = c.convert(n.Value(), t)
		case *StructType:
			n.checkExprStruct(c, x, exprT, off)
		case *UnionType:
			n.checkExprUnion(c, x, exprT, off)
		default:
			c.errors.add(errorf("TODO %T <- %T %T", x, n.Value(), exprT))
		}
	case InitializerInitList: // '{' InitializerList ',' '}'
		if n.InitializerList == nil {
			n.val = Zero
			return
		}

		n.InitializerList.check(c, t, off, true)
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
}

func (n *Initializer) checkExprArray(c *ctx, t *ArrayType, exprT Type, off int64) {
	if n.Case != InitializerExpr {
		c.errors.add(errorf("internal error: %T %v", n, n.Case))
		return
	}

	v := n.Value()
	switch x := t.Elem().(type) {
	case *ArrayType:
		n.check(c, x.Elem(), off)
		if t.IsIncomplete() {
			t.elems = 1
		}
	case
		*EnumType,
		*PointerType,
		*StructType,
		*UnionType:

		n.check(c, x, off)
		if t.IsIncomplete() {
			t.elems = 1
		}
	case *PredefinedType:
		// [0]6.7.8/14 An array of character type may be initialized by a character
		// string literal, optionally enclosed in braces. Successive characters of the
		// character string literal (including the terminating null character if there
		// is room or if the array is of unknown size) initialize the elements of the
		// array.
		switch x.Kind() {
		case Char, SChar, UChar:
			if x, ok := v.(StringValue); ok {
				if t.IsIncomplete() {
					t.elems = int64(len(x))
				}
				if max, sz := t.Len(), int64(len(x)); sz > max {
					n.val = x[:max]
				}
				return
			}
		}

		// [0]6.7.8/15 An array with element type compatible with wchar_t may be
		// initialized by a wide string literal, optionally enclosed in braces.
		// Successive wide characters of the wide string literal (including the
		// terminating null wide character if there is room or if the array is of
		// unknown size) initialize the elements of the array.
		if IsIntegerType(x) && x.Size() == c.wcharT(n).Size() {
			switch x := v.(type) {
			case UTF16StringValue:
				if t.IsIncomplete() {
					t.elems = int64(len(x))
				}
				if max, sz := t.Len(), int64(len(x)); sz > max {
					n.val = x[:max]
				}
				return
			case UTF32StringValue:
				if t.IsIncomplete() {
					t.elems = int64(len(x))
				}
				if max, sz := t.Len(), int64(len(x)); sz > max {
					n.val = x[:max]
				}
				return
			}
		}

		n.check(c, x, off)
		if t.IsIncomplete() {
			t.elems = 1
		}
	default:
		c.errors.add(errorf("TODO []%T <- %T %T", x, v, exprT))
	}
}

func (n *Initializer) checkExprStruct(c *ctx, t *StructType, exprT Type, off int64) {
	if n.Case != InitializerExpr {
		c.errors.add(errorf("internal error: %T %v", n, n.Case))
		return
	}

	switch x := exprT.(type) {
	case
		*EnumType,
		*PointerType,
		*PredefinedType:

		f := t.NamedFieldByIndex(0)
		if f == nil {
			c.errors.add(errorf("TODO %T <- %T %T", t, n.Value(), exprT))
			return
		}

		n.check(c, f.Type(), off)
	case *StructType:
		if !t.isCompatible(x) {
			c.errors.add(errorf("%v: incompatible types: %s and %s", n.AssignmentExpression.Position(), t, x))
		}
	default:
		c.errors.add(errorf("TODO %T <- %T %T", t, n.Value(), x))
	}

}

func (n *Initializer) checkExprUnion(c *ctx, t *UnionType, exprT Type, off int64) {
	if n.Case != InitializerExpr {
		c.errors.add(errorf("internal error: %T %v", n, n.Case))
		return
	}

	switch x := exprT.(type) {
	case
		*PointerType,
		*PredefinedType:

		f := t.NamedFieldByIndex(0)
		if f == nil {
			c.errors.add(errorf("TODO %T <- %T %T", t, n.Value(), exprT))
			return
		}

		n.check(c, f.Type(), off)
	case *UnionType:
		if !t.isCompatible(x) {
			c.errors.add(errorf("%v: incompatible types: %s and %s", n.AssignmentExpression.Position(), t, x))
		}
	default:
		c.errors.add(errorf("TODO %T <- %T %T", t, n.Value(), x))
	}
}

func (n *InitializerList) check(c *ctx, currObj Type, off int64, outer bool) *InitializerList {
	if n == nil || currObj == nil {
		c.errors.add(errorf("internal error: %T %T", n, currObj))
		return nil
	}

	switch x := currObj.(type) {
	case *ArrayType:
		return n.checkArray(c, x, off, outer)
	case *EnumType:
		return n.checkEnum(c, x, off, outer)
	case *PointerType:
		return n.checkPointer(c, x, off, outer)
	case *PredefinedType:
		if x.VectorSize() > 0 {
			return n.checkArray(c, x.vector, off, outer)
		}

		return n.checkPredefined(c, x, off, outer)
	case *StructType:
		return n.checkStruct(c, x, off, outer)
	case *UnionType:
		return n.checkUnion(c, x, off, outer)
	default:
		c.errors.add(errorf("TODO %T <- ...", x))
	}
	return nil
}

func (n *InitializerList) checkArray(c *ctx, t *ArrayType, off int64, outer bool) *InitializerList {
	elemT := t.Elem()
	switch {
	case t.IsIncomplete():
		var lo, hi int64
		for ; n != nil; n = n.InitializerList {
			if n.Designation != nil {
				if !outer {
					return n
				}

				dl := n.Designation.DesignatorList
				lo, hi = dl.Designator.index(c)
				if lo < 0 {
					return nil
				}

				if dl := dl.DesignatorList; dl != nil {
					//TODO if n = n.checkDesignatorList(dl, c, f.Type(), off+f.Offset()); n == nil { return nil }
					c.errors.add(errorf("TODO %T", n))
					return nil
				}

				lo = mathutil.MaxInt64(lo, hi)
			}
			n.Initializer.check(c, elemT, off+lo*elemT.Size())
			lo++
			t.elems = mathutil.MaxInt64(t.elems, lo)
		}
		return nil
	default:
		var lo, hi int64
		for ; n != nil; n = n.InitializerList {
			if n.Designation != nil {
				if !outer {
					return n
				}

				dl := n.Designation.DesignatorList
				lo, hi = dl.Designator.index(c)
				if lo < 0 {
					return nil
				}

				if lo >= t.elems {
					c.errors.add(errorf("%v: index %v out of range for array of %v elements", dl.Designator.Position(), lo, t.elems))
					return nil
				}

				if hi >= t.elems {
					c.errors.add(errorf("%v: index %v out of range for array of %v elements", dl.Designator.Position(), hi, t.elems))
					return nil
				}

				switch {
				case hi < 0:
					n.Initializer.nelems = 1
				default:
					n.Initializer.nelems = hi - lo + 1
				}

				if dl := dl.DesignatorList; dl != nil {
					if n = n.checkDesignatorList(dl, c, elemT, off+lo*elemT.Size(), hi >= 0); n == nil {
						return nil
					}
				}

				lo = mathutil.MaxInt64(lo, hi)
			}
			if lo >= t.elems {
				c.errors.add(errorf("%v: index %v out of range for array of %v elements", lo, t.elems))
				return nil
			}

			n.Initializer.check(c, elemT, off+lo*elemT.Size())
			lo++
			if lo == t.elems {
				return n.InitializerList
			}
		}
		return n
	}
}

func (n *Designator) index(c *ctx) (lo, hi int64) {
	switch n.Case {
	case DesignatorIndex: // '[' ConstantExpression ']'
		lo = arrayIndex(c, n.ConstantExpression)
		hi = arrayIndex(c, n.ConstantExpression2)
		if lo < 0 {
			c.errors.add(errorf("%v: array indices cannot be negative"))
			return -1, -1
		}

		return lo, hi
	case DesignatorIndex2: // '[' ConstantExpression "..." ConstantExpression ']'
		lo = arrayIndex(c, n.ConstantExpression)
		hi = arrayIndex(c, n.ConstantExpression2)
		if lo < 0 || hi < 0 {
			c.errors.add(errorf("%v: array indices cannot be negative"))
			return -1, -1
		}

		if lo >= 0 && hi >= 0 && lo >= hi {
			c.errors.add(errorf("%v: first index shall be smaller than second index"))
			return -1, -1
		}

		return lo, hi
	case
		DesignatorField,  // '.' IDENTIFIER
		DesignatorField2: // IDENTIFIER ':'
		c.errors.add(errorf("%v: expected bracketed array index"))
	}
	return -1, -1
}

func arrayIndex(c *ctx, n ExpressionNode) int64 {
	if n == nil {
		return -1
	}

	v, ok := int64Value(c, n)
	if !ok || v < 0 {
		c.errors.add(errorf("%v: invalid array index", n.Position()))
		return -1
	}

	return v
}

func int64Value(c *ctx, n ExpressionNode) (int64, bool) {
	if n == nil {
		return 0, false
	}

	switch t := n.check(c, decay); {
	case IsIntegerType(t):
		switch x := n.eval(c, 0).(type) {
		case Int64Value:
			return int64(x), true
		case UInt64Value:
			if x <= math.MaxInt64 {
				return int64(x), true
			}
		}
	}
	return 0, false
}

func (n *InitializerList) checkEnum(c *ctx, t *EnumType, off int64, outer bool) *InitializerList {
	if n.Designation != nil {
		if !outer {
			return n
		}

		c.errors.add(errorf("%v: unexpected designation", n.Designation.Position()))
		return nil
	}

	if IsScalarType(t) {
		n.Initializer.check(c, t, off)
		return n.InitializerList
	}

	c.errors.add(errorf("TODO %T <- ...", t))
	return nil
}

func (n *InitializerList) checkPointer(c *ctx, t *PointerType, off int64, outer bool) *InitializerList {
	if n.Designation != nil {
		if !outer {
			return n
		}

		c.errors.add(errorf("%v: unexpected designation", n.Designation.Position()))
		return nil
	}

	if IsScalarType(t) {
		n.Initializer.check(c, t, off)
		return n.InitializerList
	}

	c.errors.add(errorf("TODO %T <- ...", t))
	return nil
}

func (n *InitializerList) checkPredefined(c *ctx, t *PredefinedType, off int64, outer bool) *InitializerList {
	if n.Designation != nil {
		if !outer {
			return n
		}

		c.errors.add(errorf("%v: unexpected designation", n.Designation.Position()))
		return nil
	}

	if IsScalarType(t) {
		n.Initializer.check(c, t, off)
		return n.InitializerList
	}

	c.errors.add(errorf("TODO %T <- ...", t))
	return nil
}

func (n *InitializerList) checkStruct(c *ctx, t *StructType, off int64, outer bool) *InitializerList {
	f := t.NamedFieldByIndex(0)
	for f != nil && n != nil {
		if n.Designation != nil {
			if !outer {
				return n
			}

			dl := n.Designation.DesignatorList
			nm := dl.Designator.name(c)
			if nm == "" {
				return nil
			}

			if f = t.FieldByName(nm); f == nil {
				c.errors.add(errorf("%v: %v has no member %v", n.Position(), t, nm))
				return nil
			}

			if dl := dl.DesignatorList; dl != nil {
				if n = n.checkDesignatorList(dl, c, f.Type(), off+f.Offset(), false); n == nil {
					return nil
				}
			}
		}

		if f == nil {
			return nil
		}

		n.Initializer.check(c, f.Type(), off+f.Offset())
		n = n.InitializerList
		f = t.NamedFieldByIndex(f.ordinal + 1)
	}
	return n
}

func (n *InitializerList) checkDesignatorList(dl *DesignatorList, c *ctx, t Type, off int64, ranged bool) *InitializerList {
	for ; dl != nil; dl = dl.DesignatorList {
		switch x := t.(type) {
		case *StructType:
			nm := dl.Designator.name(c)
			if nm == "" {
				return nil
			}

			f := x.FieldByName(nm)
			if f == nil {
				c.errors.add(errorf("%v: type %s has no member %s", dl.Designator.Position(), t, nm))
				return nil
			}

			t = f.Type()
			off += f.Offset()
		case *UnionType:
			nm := dl.Designator.name(c)
			if nm == "" {
				return nil
			}

			f := x.FieldByName(nm)
			if f == nil {
				c.errors.add(errorf("%v: type %s has no member %s", dl.Designator.Position(), t, nm))
				return nil
			}

			t = f.Type()
			off += f.Offset()
		case *ArrayType:
			lo, hi := dl.Designator.index(c)
			if lo < 0 {
				return nil
			}

			if hi > lo {
				c.errors.add(errorf("TODO: %T", x))
				return nil
			}

			t = x.Elem()
			off += lo * t.Size()
			if hi < 0 {
				hi = lo
			}
			if e := x.Len(); e >= 0 {
				if lo >= e {
					c.errors.add(errorf("%v: index out of range: %v", dl.Position(), lo))
					return nil
				}

				if hi >= e {
					c.errors.add(errorf("%v: index out of range: %v", dl.Position(), hi))
					return nil
				}
			}
			switch {
			case hi < 0:
				n.Initializer.nelems = 1
			default:
				if ranged {
					c.errors.add(errorf("%v: nested ranged desinations", dl.Position()))
					return nil
				}

				n.Initializer.nelems = hi - lo + 1
				ranged = true
			}
		default:
			c.errors.add(errorf("internal error: %T", x))
			return nil
		}
	}
	switch {
	case n.Initializer.Case == InitializerInitList:
		n.Initializer.check(c, t, off)
		return n.InitializerList
	default:
		n2 := *n
		n2.Designation = nil
		return n2.check(c, t, off, false)
	}
}

func (n *Designator) name(c *ctx) string {
	switch n.Case {
	case
		DesignatorIndex,  // '[' ConstantExpression ']'
		DesignatorIndex2: // '[' ConstantExpression "..." ConstantExpression ']'

		c.errors.add(errorf("%v: expected field name"))
	case DesignatorField: // '.' IDENTIFIER
		return n.Token2.SrcStr()
	case DesignatorField2: // IDENTIFIER ':'
		return n.Token.SrcStr()
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return ""
}

func (n *InitializerList) checkUnion(c *ctx, t *UnionType, off int64, outer bool) *InitializerList {
	f := t.NamedFieldByIndex(0)
	if n.Designation != nil {
		if !outer {
			return n
		}

		dl := n.Designation.DesignatorList
		nm := dl.Designator.name(c)
		if nm == "" {
			return nil
		}

		if f = t.FieldByName(nm); f == nil {
			c.errors.add(errorf("%v: %v has no member %v", n.Position(), t, nm))
			return nil
		}

		if dl := dl.DesignatorList; dl != nil {
			return n.checkDesignatorList(dl, c, f.Type(), off+f.Offset(), false)
		}
	}

	if f == nil {
		return nil
	}

	n.Initializer.check(c, f.Type(), off+f.Offset())
	return n.InitializerList
}

//  Declarator:
//          Pointer DirectDeclarator
func (n *Declarator) check(c *ctx, t Type) (r Type) {
	if t == nil {
		c.errors.add(errorf("TODO %T internal error", n))
		return
	}

	if n == nil {
		return t
	}

	defer func() {
		if r == nil || r == Invalid {
			c.errors.add(errorf("TODO %T missed/failed type check", n))
		}
	}()

	r = n.DirectDeclarator.check(c, n.Pointer.check(c, t))
	if n.isTypename {
		r = r.setName(n)
	}
	n.typ = r
	return n.Type()
}

//  DirectDeclarator:
//          IDENTIFIER                                                             // Case DirectDeclaratorIdent
//  |       '(' Declarator ')'                                                     // Case DirectDeclaratorDecl
//  |       DirectDeclarator '[' TypeQualifiers AssignmentExpression ']'           // Case DirectDeclaratorArr
//  |       DirectDeclarator '[' "static" TypeQualifiers AssignmentExpression ']'  // Case DirectDeclaratorStaticArr
//  |       DirectDeclarator '[' TypeQualifiers "static" AssignmentExpression ']'  // Case DirectDeclaratorArrStatic
//  |       DirectDeclarator '[' TypeQualifiers '*' ']'                            // Case DirectDeclaratorStar
//  |       DirectDeclarator '(' ParameterTypeList ')'                             // Case DirectDeclaratorFuncParam
//  |       DirectDeclarator '(' IdentifierList ')'                                // Case DirectDeclaratorFuncIdent
func (n *DirectDeclarator) check(c *ctx, t Type) (r Type) {
	if t == nil {
		c.errors.add(errorf("TODO %T internal error", n))
		return
	}

	defer func() {
		if r == nil || r == Invalid {
			c.errors.add(errorf("TODO %T missed/failed type check", n))
		}
	}()

	switch n.Case {
	case DirectDeclaratorIdent: // IDENTIFIER
		return t
	case DirectDeclaratorDecl: // '(' Declarator ')'
		return n.Declarator.check(c, t)
	case DirectDeclaratorArr: // DirectDeclarator '[' TypeQualifiers AssignmentExpression ']'
		return n.DirectDeclarator.check(c, c.newArrayType(t, arraySize(c, n.AssignmentExpression), n.AssignmentExpression))
	case DirectDeclaratorStaticArr: // DirectDeclarator '[' "static" TypeQualifiers AssignmentExpression ']'
		return n.DirectDeclarator.check(c, c.newArrayType(t, arraySize(c, n.AssignmentExpression), n.AssignmentExpression))
	case DirectDeclaratorArrStatic: // DirectDeclarator '[' TypeQualifiers "static" AssignmentExpression ']'
		return n.DirectDeclarator.check(c, c.newArrayType(t, arraySize(c, n.AssignmentExpression), n.AssignmentExpression))
	case DirectDeclaratorStar: // DirectDeclarator '[' TypeQualifiers '*' ']'
		c.errors.add(errorf("TODO %v", n.Case))
	case DirectDeclaratorFuncParam: // DirectDeclarator '(' ParameterTypeList ')'
		fp, isVariadic := n.ParameterTypeList.check(c)
		return n.DirectDeclarator.check(c, c.newFunctionType(t, fp, isVariadic))
	case DirectDeclaratorFuncIdent: // DirectDeclarator '(' IdentifierList ')'
		return n.DirectDeclarator.check(c, c.newFunctionType(t, nil, false))
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return t //TODO-
}

func arraySize(c *ctx, n ExpressionNode) int64 {
	if n == nil {
		return -1
	}

	v, ok := int64Value(c, n)
	if !ok { // VLA
		return -1
	}

	if v < 0 {
		c.errors.add(errorf("%v: invalid array size, %s: %v", n.Position(), NodeSource(n), v))
		return -1
	}

	return v
}

//  ParameterTypeList:
//          ParameterList            // Case ParameterTypeListList
//  |       ParameterList ',' "..."  // Case ParameterTypeListVar
func (n *ParameterTypeList) check(c *ctx) (fp []*ParameterDeclaration, isVariadic bool) {
	if n == nil {
		return nil, false
	}

	fp = n.ParameterList.check(c)
	return fp, n.Case == ParameterTypeListVar
}

//  ParameterList:
//          ParameterDeclaration
//  |       ParameterList ',' ParameterDeclaration
func (n *ParameterList) check(c *ctx) (fp []*ParameterDeclaration) {
	for ; n != nil; n = n.ParameterList {
		n.ParameterDeclaration.check(c)
		fp = append(fp, n.ParameterDeclaration)
	}
	return fp
}

//  ParameterDeclaration:
//          DeclarationSpecifiers Declarator          // Case ParameterDeclarationDecl
//  |       DeclarationSpecifiers AbstractDeclarator  // Case ParameterDeclarationAbstract
func (n *ParameterDeclaration) check(c *ctx) {
	var isExtern, isStatic, isAtomic, isThreadLocal, isConst, isVolatile, isInline, isRegister, isAuto, isNoreturn, isRestrict bool
	var alignas int
	switch n.Case {
	case ParameterDeclarationDecl: // DeclarationSpecifiers Declarator
		n.Declarator.isParam = true
		n.typ = n.Declarator.check(c, n.DeclarationSpecifiers.check(c, &isExtern, &isStatic, &isAtomic, &isThreadLocal, &isConst, &isVolatile, &isInline, &isRegister, &isAuto, &isNoreturn, &isRestrict, &alignas))
		n.Declarator.isConst = isConst
		n.Declarator.isVolatile = isVolatile
		n.Declarator.isRegister = isRegister
		n.Declarator.isRestrict = isRestrict
		n.Declarator.isNoreturn = isNoreturn
		if isExtern || isStatic || isAtomic || isThreadLocal || isInline || isAuto || alignas != 0 {
			c.errors.add(errorf("%v: invalid specifier(s) for parameter: abc", n.Declarator.Position()))
		}
	case ParameterDeclarationAbstract: // DeclarationSpecifiers AbstractDeclarator
		n.typ = n.AbstractDeclarator.check(c, n.DeclarationSpecifiers.check(c, &isExtern, &isStatic, &isAtomic, &isThreadLocal, &isConst, &isVolatile, &isInline, &isRegister, &isAuto, &isNoreturn, &isRestrict, &alignas))
		if isExtern || isStatic || isAtomic || isThreadLocal || isInline || isAuto || isNoreturn || alignas != 0 {
			c.errors.add(errorf("%v: invalid specifier(s) for unnamed parameter", n.Position()))
		}
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
}

//  AbstractDeclarator:
//          Pointer                           // Case AbstractDeclaratorPtr
//  |       Pointer DirectAbstractDeclarator  // Case AbstractDeclaratorDecl
func (n *AbstractDeclarator) check(c *ctx, t Type) (r Type) {
	if t == nil {
		c.errors.add(errorf("TODO %T internal error", n))
		return
	}

	if n == nil {
		return t
	}

	defer func() {
		if r == nil || r == Invalid {
			c.errors.add(errorf("TODO %T missed/failed type check", n))
		}
	}()

	n.typ = n.DirectAbstractDeclarator.check(c, n.Pointer.check(c, t))
	return n.Type()
}

//  DirectAbstractDeclarator:
//          '(' AbstractDeclarator ')'                                                     // Case DirectAbstractDeclaratorDecl
//  |       DirectAbstractDeclarator '[' TypeQualifiers AssignmentExpression ']'           // Case DirectAbstractDeclaratorArr
//  |       DirectAbstractDeclarator '[' "static" TypeQualifiers AssignmentExpression ']'  // Case DirectAbstractDeclaratorStaticArr
//  |       DirectAbstractDeclarator '[' TypeQualifiers "static" AssignmentExpression ']'  // Case DirectAbstractDeclaratorArrStatic
//  |       DirectAbstractDeclarator '[' '*' ']'                                           // Case DirectAbstractDeclaratorArrStar
//  |       DirectAbstractDeclarator '(' ParameterTypeList ')'                             // Case DirectAbstractDeclaratorFunc
func (n *DirectAbstractDeclarator) check(c *ctx, t Type) (r Type) {
	if t == nil {
		c.errors.add(errorf("TODO %T internal error", n))
		return
	}

	if n == nil {
		return t
	}

	defer func() {
		if r == nil || r == Invalid {
			c.errors.add(errorf("TODO %T missed/failed type check", n))
		}
	}()

	switch n.Case {
	case DirectAbstractDeclaratorDecl: // '(' AbstractDeclarator ')'
		return n.AbstractDeclarator.check(c, t)
	case DirectAbstractDeclaratorArr: // DirectAbstractDeclarator '[' TypeQualifiers AssignmentExpression ']'
		return n.DirectAbstractDeclarator.check(c, c.newArrayType(t, arraySize(c, n.AssignmentExpression), n.AssignmentExpression))
	case DirectAbstractDeclaratorStaticArr: // DirectAbstractDeclarator '[' "static" TypeQualifiers AssignmentExpression ']'
		c.errors.add(errorf("TODO %v", n.Case))
	case DirectAbstractDeclaratorArrStatic: // DirectAbstractDeclarator '[' TypeQualifiers "static" AssignmentExpression ']'
		c.errors.add(errorf("TODO %v", n.Case))
	case DirectAbstractDeclaratorArrStar: // DirectAbstractDeclarator '[' '*' ']'
		c.errors.add(errorf("TODO %v", n.Case))
	case DirectAbstractDeclaratorFunc: // DirectAbstractDeclarator '(' ParameterTypeList ')'
		fp, isVariadic := n.ParameterTypeList.check(c)
		return n.DirectAbstractDeclarator.check(c, c.newFunctionType(t, fp, isVariadic))
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return t //TODO-
}

//  Pointer:
//          '*' TypeQualifiers          // Case PointerTypeQual
//  |       '*' TypeQualifiers Pointer  // Case PointerPtr
//  |       '^' TypeQualifiers          // Case PointerBlock
func (n *Pointer) check(c *ctx, t Type) (r Type) {
	if t == nil {
		c.errors.add(errorf("TODO %T internal error", n))
		return
	}

	if n == nil {
		return t
	}

	defer func() {
		if r == nil || r == Invalid {
			c.errors.add(errorf("TODO %T missed/failed type check", n))
		}
	}()

	switch n.Case {
	case PointerTypeQual: // '*' TypeQualifiers
		return c.newPointerType(t)
	case PointerPtr: // '*' TypeQualifiers Pointer
		return n.Pointer.check(c, c.newPointerType(t))
	case PointerBlock: // '^' TypeQualifiers
		return n.Pointer.check(c, c.newPointerType(t))
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return t //TODO-
}

func ts2String(a []TypeSpecifierCase) string {
	sort.Slice(a, func(i, j int) bool { return a[i] < a[j] })
	var b strings.Builder
	for _, v := range a {
		b.WriteByte(byte(v))
	}
	return b.String()
}

//  DeclarationSpecifiers:
//          StorageClassSpecifier DeclarationSpecifiers  // Case DeclarationSpecifiersStorage
//  |       TypeSpecifier DeclarationSpecifiers          // Case DeclarationSpecifiersTypeSpec
//  |       TypeQualifier DeclarationSpecifiers          // Case DeclarationSpecifiersTypeQual
//  |       FunctionSpecifier DeclarationSpecifiers      // Case DeclarationSpecifiersFunc
//  |       AlignmentSpecifier DeclarationSpecifiers     // Case DeclarationSpecifiersAlignSpec
//  |       "__attribute__"                              // Case DeclarationSpecifiersAttr
func (n *DeclarationSpecifiers) check(c *ctx, isExtern, isStatic, isAtomic, isThreadLocal, isConst, isVolatile, isInline, isRegister, isAuto, isNoreturn, isRestrict *bool, alignas *int) (r Type) {
	if n == nil {
		return c.intT
	}

	n0 := n
	var attr *Attributes
	var ts []TypeSpecifierCase

	defer func(n *DeclarationSpecifiers) {
		if r == nil || r == Invalid {
			//panic(todo("%v: %v %v", n.Position(), ts, TypeString(r)))
			c.errors.add(errorf("TODO %T missed/failed type check: %v", n, ts))
			return
		}

		if _, ok := r.(*PredefinedType); ok && r.Size() < 0 { // Not supported on target
			c.errors.add(errorf("%s not supported on %s/%s", r, c.ast.ABI.goos, c.ast.ABI.goarch))
			r = Invalid
		}
		if attr != nil {
			r = r.setAttr(attr)
		}
		n.typ = r
	}(n)

	var attrs []*Attributes
	for ; n != nil; n = n.DeclarationSpecifiers {
		switch n.Case {
		case DeclarationSpecifiersStorage: // StorageClassSpecifier DeclarationSpecifiers
			n.StorageClassSpecifier.check(c, isExtern, isStatic, isThreadLocal, isRegister, isAuto)
		case DeclarationSpecifiersTypeSpec: // TypeSpecifier DeclarationSpecifiers
			ts = append(ts, n.TypeSpecifier.Case)
			r = n.TypeSpecifier.check(c, isAtomic)
		case DeclarationSpecifiersTypeQual: // TypeQualifier DeclarationSpecifiers
			n.TypeQualifier.check(c, isConst, isVolatile, isAtomic, isRestrict)
		case DeclarationSpecifiersFunc: // FunctionSpecifier DeclarationSpecifiers
			n.FunctionSpecifier.check(c, isInline, isNoreturn)
		case DeclarationSpecifiersAlignSpec: // AlignmentSpecifier DeclarationSpecifiers
			if v := n.AlignmentSpecifier.check(c).Align(); v > 0 {
				*alignas = v
			}
		case DeclarationSpecifiersAttr:
			if attr := n.AttributeSpecifierList.check(c); attr != nil {
				attrs = append(attrs, attr)
			}
		default:
			c.errors.add(errorf("internal error: %v", n.Case))
		}
	}
	switch len(attrs) {
	case 0:
		// ok
	case 1:
		attr = attrs[0]
	default:
		c.errors.add(errorf("TODO %T", n0.Position()))
	}

	t, ok := c.builtinTypes[ts2String(ts)]
	if ok {
		return t
	}

	switch len(ts) {
	case 0:
		return c.intT
	case 1:
		switch ts[0] {
		case
			TypeSpecifierAtomic,
			TypeSpecifierEnum,
			TypeSpecifierStructOrUnion,
			TypeSpecifierTypeName,
			TypeSpecifierTypeofExpr:

			return r
		}
	}

	return nil
}

//  AttributeSpecifierList:
//          AttributeSpecifier
//  |       AttributeSpecifierList AttributeSpecifier
func (n *AttributeSpecifierList) check(c *ctx) *Attributes {
	attr := newAttributes()
	for ; n != nil; n = n.AttributeSpecifierList {
		n.AttributeSpecifier.check(c, attr)
	}
	if attr.isNonZero {
		return attr
	}

	return nil
}

//  AttributeSpecifier:
//          "__attribute__" '(' '(' AttributeValueList ')' ')'
func (n *AttributeSpecifier) check(c *ctx, attr *Attributes) {
	n.AttributeValueList.check(c, attr)
}

//  AttributeValueList:
//          AttributeValue
//  |       AttributeValueList ',' AttributeValue
func (n *AttributeValueList) check(c *ctx, attr *Attributes) {
	for ; n != nil; n = n.AttributeValueList {
		n.AttributeValue.check(c, attr)
	}
}

//  AttributeValue:
//          IDENTIFIER                                 // Case AttributeValueIdent
//  |       IDENTIFIER '(' ArgumentExpressionList ')'  // Case AttributeValueExpr
func (n *AttributeValue) check(c *ctx, attr *Attributes) {
	switch n.Case {
	case AttributeValueIdent: // IDENTIFIER
		// ok
	case AttributeValueExpr: // IDENTIFIER '(' ArgumentExpressionList ')'
		n.ArgumentExpressionList.check(c, decay|ignoreUndefined)
		switch n.Token.SrcStr() {
		case "aligned":
			e := n.ArgumentExpressionList.AssignmentExpression
			if n.ArgumentExpressionList.ArgumentExpressionList != nil {
				c.errors.add(errorf("%v: expected one expression", e.Position()))
				break
			}

			v, ok := int64Value(c, e)
			if !ok {
				c.errors.add(errorf("%v: expected a constant integer value", e.Position()))
				return
			}

			if attr.Aligned() > 0 {
				c.errors.add(errorf("%v: multiple 'aligned' specifications", e.Position()))
				return
			}

			if v <= 0 {
				c.errors.add(errorf("%v: alignment must be positive", e.Position()))
				return
			}

			attr.setAligned(v)
		case
			"__vector_size__",
			"vector_size":

			e := n.ArgumentExpressionList.AssignmentExpression
			if n.ArgumentExpressionList.ArgumentExpressionList != nil {
				c.errors.add(errorf("%v: expected one expression", e.Position()))
				break
			}

			v, ok := int64Value(c, e)
			if !ok {
				c.errors.add(errorf("%v: expected a constant integer value", e.Position()))
				return
			}

			if attr.VectorSize() > 0 {
				c.errors.add(errorf("%v: multiple 'vector_size' specifications", e.Position()))
				return
			}

			if v <= 0 {
				c.errors.add(errorf("%v: vector size must be positive", e.Position()))
				return
			}

			if v&(v-1) != 0 {
				c.errors.add(errorf("%v: vector size must be a power of two: %v", e.Position(), v))
				return
			}

			attr.setVectorSize(v)
		}
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
}

//  ArgumentExpressionList:
//          AssignmentExpression
//  |       ArgumentExpressionList ',' AssignmentExpression
func (n *ArgumentExpressionList) check(c *ctx, mode flags) (r []ExpressionNode) {
	for ; n != nil; n = n.ArgumentExpressionList {
		n.AssignmentExpression.check(c, mode)
		n.AssignmentExpression.eval(c, mode)
		r = append(r, n.AssignmentExpression)
	}
	return r
}

//  AlignmentSpecifier:
//          "_Alignas" '(' TypeName ')'            // Case AlignmentSpecifierType
//  |       "_Alignas" '(' ConstantExpression ')'  // Case AlignmentSpecifierExpr
func (n *AlignmentSpecifier) check(c *ctx) (r Type) {
	switch n.Case {
	case AlignmentSpecifierType: // "_Alignas" '(' TypeName ')'
		return n.TypeName.check(c)
	case AlignmentSpecifierExpr: // "_Alignas" '(' ConstantExpression ')'
		return n.ConstantExpression.check(c, decay)
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
		return nil
	}
}

//  FunctionSpecifier:
//          "inline"     // Case FunctionSpecifierInline
//  |       "_Noreturn"  // Case FunctionSpecifierNoreturn
func (n *FunctionSpecifier) check(c *ctx, isInline, isNoreturn *bool) {
	switch n.Case {
	case FunctionSpecifierInline: // "inline"
		*isInline = true
	case FunctionSpecifierNoreturn: // "_Noreturn"
		*isNoreturn = true
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
}

//  TypeQualifier:
//          "const"          // Case TypeQualifierConst
//  |       "restrict"       // Case TypeQualifierRestrict
//  |       "volatile"       // Case TypeQualifierVolatile
//  |       "_Atomic"        // Case TypeQualifierAtomic
//  |       "_Nonnull"       // Case TypeQualifierNonnull
//  |       "__attribute__"  // Case TypeQualifierAttr
func (n *TypeQualifier) check(c *ctx, isConst, isVolatile, isAtomic, isRestrict *bool) {
	switch n.Case {
	case TypeQualifierConst: // "const"
		*isConst = true
	case TypeQualifierRestrict: // "restrict"
		*isRestrict = true
	case TypeQualifierVolatile: // "volatile"
		*isVolatile = true
	case TypeQualifierAtomic: // "_Atomic"
		*isAtomic = true
	case TypeQualifierNonnull: // "_Nonnull"
		c.errors.add(errorf("TODO %v", n.Case))
	case TypeQualifierAttr: // AttributeSpecifierList
		n.AttributeSpecifierList.check(c)
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
}

//  StorageClassSpecifier:
//          "typedef"             // Case StorageClassSpecifierTypedef
//  |       "extern"              // Case StorageClassSpecifierExtern
//  |       "static"              // Case StorageClassSpecifierStatic
//  |       "auto"                // Case StorageClassSpecifierAuto
//  |       "register"            // Case StorageClassSpecifierRegister
//  |       "_Thread_local"       // Case StorageClassSpecifierThreadLocal
//  |       "__declspec" '(' ')'  // Case StorageClassSpecifierDeclspec
func (n *StorageClassSpecifier) check(c *ctx, isExtern, isStatic, isThreadLocal, isRegister, isAuto *bool) {
	switch n.Case {
	case StorageClassSpecifierTypedef: // "typedef"
		// ok
	case StorageClassSpecifierExtern: // "extern"
		*isExtern = true
	case StorageClassSpecifierStatic: // "static"
		*isStatic = true
	case StorageClassSpecifierAuto: // "auto"
		*isAuto = true
	case StorageClassSpecifierRegister: // "register"
		*isRegister = true
	case StorageClassSpecifierThreadLocal: // "_Thread_local"
		*isThreadLocal = true
	case StorageClassSpecifierDeclspec: // "__declspec" '(' ')'
		c.errors.add(errorf("TODO %v", n.Case))
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
}

//  TypeSpecifier:
//          "void"                           // Case TypeSpecifierVoid
//  |       "char"                           // Case TypeSpecifierChar
//  |       "short"                          // Case TypeSpecifierShort
//  |       "int"                            // Case TypeSpecifierInt
//  |       "__int128"                       // Case TypeSpecifierInt128
//  |       "__uint128_t"                    // Case TypeSpecifierUint128
//  |       "long"                           // Case TypeSpecifierLong
//  |       "float"                          // Case TypeSpecifierFloat
//  |       "_Float16"                       // Case TypeSpecifierFloat16
//  |       "_Decimal128"                    // Case TypeSpecifierDecimal128
//  |       "_Decimal32"                     // Case TypeSpecifierDecimal32
//  |       "_Decimal64"                     // Case TypeSpecifierDecimal64
//  |       "_Float128"                      // Case TypeSpecifierFloat128
//  |       "_Float128x"                     // Case TypeSpecifierFloat128x
//  |       "double"                         // Case TypeSpecifierDouble
//  |       "signed"                         // Case TypeSpecifierSigned
//  |       "unsigned"                       // Case TypeSpecifierUnsigned
//  |       "_Bool"                          // Case TypeSpecifierBool
//  |       "_Complex"                       // Case TypeSpecifierComplex
//  |       "_Imaginary"                     // Case TypeSpecifierImaginary
//  |       StructOrUnionSpecifier           // Case TypeSpecifierStructOrUnion
//  |       EnumSpecifier                    // Case TypeSpecifierEnum
//  |       TYPENAME                         // Case TypeSpecifierTypeName
//  |       "typeof" '(' ExpressionList ')'  // Case TypeSpecifierTypeofExpr
//  |       "typeof" '(' TypeName ')'        // Case TypeSpecifierTypeofType
//  |       AtomicTypeSpecifier              // Case TypeSpecifierAtomic
//  |       "_Float32"                       // Case TypeSpecifierFloat32
//  |       "_Float64"                       // Case TypeSpecifierFloat64
//  |       "_Float32x"                      // Case TypeSpecifierFloat32x
//  |       "_Float64x"                      // Case TypeSpecifierFloat64x
func (n *TypeSpecifier) check(c *ctx, isAtomic *bool) (r Type) {
	if n == nil {
		return Invalid
	}

	switch n.Case {
	case TypeSpecifierVoid: // "void"
		// ok
	case TypeSpecifierChar: // "char"
		// ok
	case TypeSpecifierShort: // "short"
		// ok
	case TypeSpecifierInt: // "int"
		// ok
	case TypeSpecifierInt128: // "__int128"
		// ok
	case TypeSpecifierUint128: // "__uint128_t"
		// ok
	case TypeSpecifierLong: // "long"
		// ok
	case TypeSpecifierFloat: // "float"
		// ok
	case TypeSpecifierFloat16: // "_Float16"
		// ok
	case TypeSpecifierDecimal128: // "_Decimal128"
		// ok
	case TypeSpecifierDecimal32: // "_Decimal32"
		// ok
	case TypeSpecifierDecimal64: // "_Decimal64"
		// ok
	case TypeSpecifierFloat128: // "_Float128"
		// ok
	case TypeSpecifierFloat128x: // "_Float128x"
		// ok
	case TypeSpecifierDouble: // "double"
		// ok
	case TypeSpecifierSigned: // "signed"
		// ok
	case TypeSpecifierUnsigned: // "unsigned"
		// ok
	case TypeSpecifierBool: // "_Bool"
		// ok
	case TypeSpecifierComplex: // "_Complex"
		// ok
	case TypeSpecifierImaginary: // "_Imaginary"
		c.errors.add(errorf("TODO %v", n.Case))
	case TypeSpecifierStructOrUnion: // StructOrUnionSpecifier
		return n.StructOrUnionSpecifier.check(c)
	case TypeSpecifierEnum: // EnumSpecifier
		return n.EnumSpecifier.check(c)
	case TypeSpecifierTypeName: // TYPENAME
		if x, ok := n.LexicalScope().ident(n.Token).(*Declarator); ok && x.isTypename {
			return x.Type()
		}

		c.errors.add(errorf("%v: undefined type name: %s", n.Position(), n.Token.Src()))
	case TypeSpecifierTypeofExpr: // "typeof" '(' ExpressionList ')'
		return n.ExpressionList.check(c, 0)
	case TypeSpecifierTypeofType: // "typeof" '(' TypeName ')'
		return n.TypeName.check(c)
	case TypeSpecifierAtomic: // AtomicTypeSpecifier
		*isAtomic = true
		return n.AtomicTypeSpecifier.check(c)
	case TypeSpecifierFloat32: // "_Float32"
		// ok
	case TypeSpecifierFloat64: // "_Float64"
		// ok
	case TypeSpecifierFloat32x: // "_Float32x"
		// ok
	case TypeSpecifierFloat64x: // "_Float64x"
		// ok
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return nil
}

func (n *AtomicTypeSpecifier) check(c *ctx) (r Type) {
	return n.TypeName.check(c)
}

//  EnumSpecifier:
//          "enum" IDENTIFIER '{' EnumeratorList ',' '}'  // Case EnumSpecifierDef
//  |       "enum" IDENTIFIER                             // Case EnumSpecifierTag
func (n *EnumSpecifier) check(c *ctx) (r Type) {
	if n == nil {
		return Invalid
	}

	if n.typ != nil {
		return n.Type()
	}

	defer func() {
		if r == nil || r == Invalid {
			c.errors.add(errorf("TODO %T missed/failed type check", n))
		}
	}()

	tag := n.Token2
	switch n.Case {
	case EnumSpecifierDef: // "enum" IDENTIFIER '{' EnumeratorList ',' '}'
		var t Type
		var iota, min int64
		var max uint64
		var list []*Enumerator
		for l := n.EnumeratorList; l != nil; l = l.EnumeratorList {
			en := l.Enumerator
			list = append(list, en)
			iota = en.check(c, iota)
			switch x := en.Value().(type) {
			case Int64Value:
				v := int64(x)
				min = mathutil.MinInt64(min, v)
				if v >= 0 {
					max = mathutil.MaxUint64(max, uint64(v))
				}
			case UInt64Value:
				v := uint64(x)
				max = mathutil.MaxUint64(max, v)
			}
		}
		switch {
		case min >= math.MinInt32 && max <= math.MaxInt32:
			t = c.intT
		case min >= 0 && max <= math.MaxUint32:
			t = c.ast.kinds[UInt]
		case max < math.MaxInt64:
			t = c.ast.kinds[Long]
			if t.Size() < 8 {
				t = c.ast.kinds[LongLong]
			}
		default:
			t = c.ast.kinds[ULong]
			if t.Size() < 8 {
				t = c.ast.kinds[ULongLong]
			}
		}
		for _, v := range list {
			v.typ = t
		}
		n.typ = c.newEnumType(n.LexicalScope(), tag, t, list)
	case EnumSpecifierTag: // "enum" IDENTIFIER
		if x := n.LexicalScope().enum(n.Token2); x != nil {
			switch {
			case x.typ == nil:
				t := c.newEnumType(n.LexicalScope(), tag, nil, nil)
				t.forward = x
				n.typ = t
			default:
				n.typ = x.typ
			}
			break
		}

		t := c.newEnumType(n.LexicalScope(), tag, nil, nil)
		t.isIncomplete0 = true
		n.typ = t
		c.ast.Scope.declare(&c.errors, tag.SrcStr(), n)
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return n.Type()
}

func (n *Enumerator) check(c *ctx, iota int64) int64 {
	switch n.Case {
	case EnumeratorIdent: // IDENTIFIER
		n.typ = c.int64T
		n.val = Int64Value(iota)
	case EnumeratorExpr: // IDENTIFIER '=' ConstantExpression
		n.typ = n.ConstantExpression.check(c, decay)
		switch n.val = n.ConstantExpression.Value(); x := n.Value().(type) {
		case Int64Value:
			return int64(x) + 1
		case UInt64Value:
			return int64(x) + 1
		case *UnknownValue:
			// ok
		default:
			c.errors.add(errorf("internal error: %T", x))
		}
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return iota + 1
}

//  StructOrUnionSpecifier:
//          StructOrUnion IDENTIFIER '{' StructDeclarationList '}'  // Case StructOrUnionSpecifierDef
//  |       StructOrUnion IDENTIFIER                                // Case StructOrUnionSpecifierTag
func (n *StructOrUnionSpecifier) check(c *ctx) (r Type) {
	if n == nil {
		return Invalid
	}

	if n.typ != nil {
		return n.Type()
	}

	tag := n.Token
	switch n.Case {
	case StructOrUnionSpecifierDef: // StructOrUnion IDENTIFIER '{' StructDeclarationList '}'
		defer func() {
			if r == nil || r == Invalid {
				c.errors.add(errorf("TODO %T missed/failed type check", n))
			}
		}()

		switch {
		case n.StructOrUnion.Case == StructOrUnionUnion:
			n.typ = c.newUnionType(n.LexicalScope(), tag, nil, -1, 1)
		default:
			n.typ = c.newStructType(n.LexicalScope(), tag, nil, -1, 1)
		}

		n.StructDeclarationList.check(c, n)
	case StructOrUnionSpecifierTag: // StructOrUnion IDENTIFIER
		if x := n.LexicalScope().structOrUnion(n.Token); x != nil {
			if n.StructOrUnion.Case != x.StructOrUnion.Case {
				c.errors.add(errorf("%v: mismatched struct/union tag", n.Token.Position()))
				break
			}

			switch {
			case x.typ == nil:
				switch {
				case n.StructOrUnion.Case == StructOrUnionUnion:
					t := c.newUnionType(n.LexicalScope(), tag, nil, -1, 1)
					t.forward = x
					n.typ = t
				default:
					t := c.newStructType(n.LexicalScope(), tag, nil, -1, 1)
					t.forward = x
					n.typ = t
				}
			default:
				n.typ = x.typ
			}
			break
		}

		switch {
		case n.StructOrUnion.Case == StructOrUnionUnion:
			t := c.newUnionType(n.LexicalScope(), tag, nil, -1, 1)
			t.isIncomplete0 = true
			n.typ = t
		default:
			t := c.newStructType(n.LexicalScope(), tag, nil, -1, 1)
			t.isIncomplete0 = true
			n.typ = t
		}
		c.ast.Scope.declare(&c.errors, tag.SrcStr(), n)
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return n.Type()
}

//  StructDeclarationList:
//          StructDeclaration
//  |       StructDeclarationList StructDeclaration
func (n *StructDeclarationList) check(c *ctx, s *StructOrUnionSpecifier) {
	if n == nil {
		return
	}

	defer func() {
		if s.typ == nil || s.typ == Invalid {
			c.errors.add(errorf("TODO %T missed/failed type check", n))
		}
	}()

	var fields []*Field
	for l := n; l != nil; l = l.StructDeclarationList {
		fields = append(fields, l.StructDeclaration.check(c)...)
	}
	// for _, v := range fields {
	// 	switch {
	// 	case v.declarator == nil:
	// 		trc("-: - %s", v.Type())
	// 	default:
	// 		trc("%v: %s %s", v.declarator.Position(), v.Name(), v.Type())
	// 	}
	// }

	isUnion := s.StructOrUnion.Case == StructOrUnionUnion
	var brk, unionBits int64
	maxAlignBytes := 1
	for i, f := range fields {
		if f == nil {
			c.errors.add(errorf("TODO %T", n))
			return
		}

		f.ordinal = i

		switch {
		case f.isBitField:
			f.accessBytes = bits2AccessBytes(f.valueBits)
			switch {
			case isUnion:
				f.mask = (uint64(1)<<f.valueBits - 1)
				unionBits = mathutil.MaxInt64(unionBits, 8*f.accessBytes)
			default:
				brkOffBytes := brk >> 3
				if mod := brkOffBytes % f.accessBytes; mod != 0 {
					brkOffBytes += f.accessBytes - mod
					brk = brkOffBytes << 3
				}
				f.offsetBytes = brkOffBytes
				f.offsetBits = int(brk - 8*f.offsetBytes)
				f.mask = (uint64(1)<<f.valueBits - 1) << f.offsetBits
				brk += f.valueBits
			}
		default:
			sz := f.Type().Size()
			if f.Type().IsIncomplete() && f.Type().Kind() == Array { // Flexible array member
				sz = 0
			}
			al := f.Type().Align()
			if al > maxAlignBytes {
				maxAlignBytes = al
			}
			if !isUnion {
				brk = roundup(brk, 8*int64(al))
			}
			f.accessBytes = sz
			f.offsetBytes = brk >> 3
			f.valueBits = 8 * sz
			switch {
			case isUnion:
				unionBits = mathutil.MaxInt64(unionBits, 8*sz)
			default:
				brk += 8 * sz
			}
		}
	}
	brk = roundup(brk, int64(maxAlignBytes*8))
	switch {
	case isUnion:
		t := s.typ.(*UnionType)
		t.fields = fields
		t.size = unionBits >> 3
		t.align = maxAlignBytes
	default:
		t := s.typ.(*StructType)
		t.fields = fields
		t.size = brk >> 3
		t.align = maxAlignBytes
	}
}

func (n *StructDeclaration) check(c *ctx) (r []*Field) {
	switch n.Case {
	case StructDeclarationDecl: // DeclarationSpecifiers InitDeclaratorList AttributeSpecifierList ';'
		var isAtomic, isConst, isVolatile, isRestrict bool
		var alignas int
		t := n.SpecifierQualifierList.check(c, &isAtomic, &isConst, &isVolatile, &isRestrict, &alignas)
		switch {
		case n.StructDeclaratorList == nil:
			return []*Field{{typ: newTyper(t)}}
		default:
			return n.StructDeclaratorList.check(c, t, isAtomic, isConst, isVolatile, isRestrict)
		}
	case StructDeclarationAssert: // StaticAssertDeclaration
		n.StaticAssertDeclaration.check(c)
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return nil
}

func (n *SpecifierQualifierList) check(c *ctx, isAtomic, isConst, isVolatile, isRestrict *bool, alignas *int) (r Type) {
	if n == nil {
		return c.intT
	}

	var ts []TypeSpecifierCase

	defer func() {
		if r == nil || r == Invalid {
			c.errors.add(errorf("TODO missed/failed type check %v: %v %T", n.Position(), ts, n))
		}
	}()

	for ; n != nil; n = n.SpecifierQualifierList {
		switch n.Case {
		case SpecifierQualifierListTypeSpec: // TypeSpecifier SpecifierQualifierList
			ts = append(ts, n.TypeSpecifier.Case)
			r = n.TypeSpecifier.check(c, isAtomic)
		case SpecifierQualifierListTypeQual: // TypeQualifier SpecifierQualifierList
			n.TypeQualifier.check(c, isConst, isVolatile, isAtomic, isRestrict)
		case SpecifierQualifierListAlignSpec: // AlignmentSpecifier SpecifierQualifierList
			if v := n.AlignmentSpecifier.check(c).Align(); v > 0 {
				*alignas = v
			}
		default:
			c.errors.add(errorf("internal error: %v", n.Case))
		}
	}
	t, ok := c.builtinTypes[ts2String(ts)]
	if ok {
		return t
	}

	switch len(ts) {
	case 0:
		return c.intT
	case 1:
		switch ts[0] {
		case
			TypeSpecifierAtomic,
			TypeSpecifierEnum,
			TypeSpecifierStructOrUnion,
			TypeSpecifierTypeName,
			TypeSpecifierTypeofExpr,
			TypeSpecifierTypeofType:

			return r
		}
	}

	return nil
}

func (n *StructDeclaratorList) check(c *ctx, t Type, isAtomic, isConst, isVolatile, isRestrict bool) (r []*Field) {
	if t == nil {
		c.errors.add(errorf("TODO %T internal error", n))
		return
	}

	for ; n != nil; n = n.StructDeclaratorList {
		r = append(r, n.StructDeclarator.check(c, t, isAtomic, isConst, isVolatile, isRestrict))
	}
	return r
}

func (n *StructDeclarator) check(c *ctx, t Type, isAtomic, isConst, isVolatile, isRestrict bool) (r *Field) {
	if t == nil {
		c.errors.add(errorf("TODO %T internal error", n))
		return
	}

	if n.Declarator != nil {
		n.Declarator.isAtomic = isAtomic
		n.Declarator.isConst = isConst
		n.Declarator.isVolatile = isVolatile
		n.Declarator.isRestrict = isRestrict
	}
	switch n.Case {
	case StructDeclaratorDecl: // Declarator
		return &Field{declarator: n.Declarator, typ: newTyper(n.Declarator.check(c, t))}
	case StructDeclaratorBitField: // Declarator ':' ConstantExpression
		t := n.ConstantExpression.check(c, decay)
		if !IsIntegerType(t) {
			c.errors.add(errorf("%v: expected integer expression: %s", n.ConstantExpression.Position(), t))
			break
		}

		var bits int64
		switch x := n.ConstantExpression.Value().(type) {
		case Int64Value:
			bits = int64(x)
		case UInt64Value:
			bits = int64(x)
		}
		if bits < 0 || bits > 64 {
			c.errors.add(errorf("%v: value out of range: %v", n.ConstantExpression.Position(), bits))
			break
		}

		return &Field{declarator: n.Declarator, typ: newTyper(n.Declarator.check(c, t)), valueBits: bits, isBitField: true}
	default:
		c.errors.add(errorf("internal error: %v %T", n.Case, n))
	}
	return nil //TODO-
}

//  AssignmentExpression:
//          ConditionalExpression                       // Case AssignmentExpressionCond
//  |       UnaryExpression '=' AssignmentExpression    // Case AssignmentExpressionAssign
//  |       UnaryExpression "*=" AssignmentExpression   // Case AssignmentExpressionMul
//  |       UnaryExpression "/=" AssignmentExpression   // Case AssignmentExpressionDiv
//  |       UnaryExpression "%=" AssignmentExpression   // Case AssignmentExpressionMod
//  |       UnaryExpression "+=" AssignmentExpression   // Case AssignmentExpressionAdd
//  |       UnaryExpression "-=" AssignmentExpression   // Case AssignmentExpressionSub
//  |       UnaryExpression "<<=" AssignmentExpression  // Case AssignmentExpressionLsh
//  |       UnaryExpression ">>=" AssignmentExpression  // Case AssignmentExpressionRsh
//  |       UnaryExpression "&=" AssignmentExpression   // Case AssignmentExpressionAnd
//  |       UnaryExpression "^=" AssignmentExpression   // Case AssignmentExpressionXor
//  |       UnaryExpression "|=" AssignmentExpression   // Case AssignmentExpressionOr
func (n *AssignmentExpression) check(c *ctx, mode flags) (r Type) {
	if n == nil {
		return Invalid
	}

	defer func() {
		if r == nil || r == Invalid {
			c.errors.add(errorf("TODO %T missed/failed type check", n))
		}
	}()

	switch n.Case {
	case AssignmentExpressionCond: // ConditionalExpression
		n.typ = n.ConditionalExpression.check(c, mode)
	case
		AssignmentExpressionAssign, // UnaryExpression '=' AssignmentExpression
		AssignmentExpressionMul,    // UnaryExpression "*=" AssignmentExpression
		AssignmentExpressionDiv,    // UnaryExpression "/=" AssignmentExpression
		AssignmentExpressionMod,    // UnaryExpression "%=" AssignmentExpression
		AssignmentExpressionAdd,    // UnaryExpression "+=" AssignmentExpression
		AssignmentExpressionSub,    // UnaryExpression "-=" AssignmentExpression
		AssignmentExpressionLsh,    // UnaryExpression "<<=" AssignmentExpression
		AssignmentExpressionRsh,    // UnaryExpression ">>=" AssignmentExpression
		AssignmentExpressionAnd,    // UnaryExpression "&=" AssignmentExpression
		AssignmentExpressionXor,    // UnaryExpression "^=" AssignmentExpression
		AssignmentExpressionOr:     // UnaryExpression "|=" AssignmentExpression

		n.typ = n.UnaryExpression.check(c, mode)
		a := n.Type()
		b := n.AssignmentExpression.check(c, mode)
		if !isModifiableLvalue(a) {
			c.errors.add(errorf("%v: left operand shall be a modifiable lvalue", n.UnaryExpression.Position()))
			break
		}

		switch {
		case
			// — the left operand has qualified or unqualified arithmetic type and the
			// right has arithmetic type;
			IsArithmeticType(a) && IsArithmeticType(b),

			// — the left operand has a qualified or unqualified version of a structure or
			// union type compatible with the type of the right;
			a.Kind() == Struct && b.Kind() == Struct || a.Kind() == Union && b.Kind() == Union,

			// — both operands are pointers to qualified or unqualified versions of
			// compatible types, and the type pointed to by the left has all the qualifiers
			// of the type pointed to by the right;
			//
			// — one operand is a pointer to an object or incomplete type and the other is
			// a pointer to a qualified or unqualified version of void, and the type
			// pointed to by the left has all the qualifiers of the type pointed to by the
			// right;
			isPointerType(a) && isPointerType(b),

			// — the left operand is a pointer and the right is a null pointer constant; or
			isPointerType(a) && IsIntegerType(b),

			// — the left operand has type _Bool and the right is a pointer.
			a.Kind() == Bool && isPointerType(b):

			// ok
		}
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return n.Type()
}

//  ConditionalExpression:
//          LogicalOrExpression                                           // Case ConditionalExpressionLOr
//  |       LogicalOrExpression '?' ExpressionList ':' ConditionalExpression  // Case ConditionalExpressionCond
func (n *ConditionalExpression) check(c *ctx, mode flags) (r Type) {
	if n == nil {
		return Invalid
	}

	defer func() {
		if r == nil || r == Invalid {
			c.errors.add(errorf("TODO %T missed/failed type check", n))
			return
		}
	}()

	switch n.Case {
	case ConditionalExpressionLOr: // LogicalOrExpression
		n.typ = n.LogicalOrExpression.check(c, mode)
	case ConditionalExpressionCond: // LogicalOrExpression '?' ExpressionList ':' ConditionalExpression
		mode = mode.add(decay)
		t1 := n.LogicalOrExpression.check(c, mode)
		if !IsScalarType(t1) {
			c.errors.add(errorf("%v: operand shall have scalar type: %s", n.LogicalOrExpression.Position(), t1))
			break
		}

		t2 := t1
		if n.ExpressionList != nil {
			t2 = n.ExpressionList.check(c, mode)
			n.ExpressionList.eval(c, decay)
		}
		switch t3 := n.ConditionalExpression.check(c, mode); {
		case
			// both operands have arithmetic type;
			IsArithmeticType(t2) && IsArithmeticType(t3):
			n.typ = UsualArithmeticConversions(t2, t3)
		case
			// both operands have the same structure or union type;
			(t2.Kind() == Struct || t2.Kind() == Union) && t3.Kind() == t2.Kind():
			n.typ = t2
		case
			// both operands have void type;
			t2.Kind() == Void && t3.Kind() == Void:
			n.typ = t2
		case
			// both operands are pointers to qualified or unqualified versions of compatible types;
			isPointerType(t2) && isPointerType(t3):
			n.typ = t2
		case
			// one operand is a pointer and the other is a null pointer constant; or
			isPointerType(t2) && IsIntegerType(t3):
			n.typ = t2
		case
			IsIntegerType(t2) && isPointerType(t3):
			n.typ = t3
		case t2.Kind() == Void:
			n.typ = t2
		case t3.Kind() == Void:
			n.typ = t3
		default:
			c.errors.add(errorf("TODO %v, t1 %v, t2 %v, t3 %v", n.Case, t1, t2, t3))
		}
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return n.Type()
}

//  LogicalOrExpression:
//          LogicalAndExpression                           // Case LogicalOrExpressionLAnd
//  |       LogicalOrExpression "||" LogicalAndExpression  // Case LogicalOrExpressionLOr
func (n *LogicalOrExpression) check(c *ctx, mode flags) (r Type) {
	if n == nil {
		return Invalid
	}

	defer func() {
		if r == nil || r == Invalid {
			c.errors.add(errorf("TODO %T missed/failed type check", n))
			return
		}
	}()

	switch n.Case {
	case LogicalOrExpressionLAnd: // LogicalAndExpression
		n.typ = n.LogicalAndExpression.check(c, mode)
	case LogicalOrExpressionLOr: // LogicalOrExpression "||" LogicalAndExpression
		mode = mode.add(decay)
		switch a, b := n.LogicalOrExpression.check(c, mode), n.LogicalAndExpression.check(c, mode); {
		case !IsScalarType(a) || !IsScalarType(b):
			c.errors.add(errorf("%v: operands shall be scalars: %s and %s", n.Token.Position(), a, b))
		default:
			n.typ = c.intT
		}
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return n.Type()
}

//  LogicalAndExpression:
//          InclusiveOrExpression                            // Case LogicalAndExpressionOr
//  |       LogicalAndExpression "&&" InclusiveOrExpression  // Case LogicalAndExpressionLAnd
func (n *LogicalAndExpression) check(c *ctx, mode flags) (r Type) {
	if n == nil {
		return Invalid
	}

	defer func() {
		if r == nil || r == Invalid {
			c.errors.add(errorf("TODO %T missed/failed type check", n))
			return
		}
	}()

	switch n.Case {
	case LogicalAndExpressionOr: // InclusiveOrExpression
		n.typ = n.InclusiveOrExpression.check(c, mode)
	case LogicalAndExpressionLAnd: // LogicalAndExpression "&&" InclusiveOrExpression
		mode = mode.add(decay)
		switch a, b := n.LogicalAndExpression.check(c, mode), n.InclusiveOrExpression.check(c, mode); {
		case !IsScalarType(a) || !IsScalarType(b):
			c.errors.add(errorf("%v: operands shall be scalars: %s and %s", n.Token.Position(), a, b))
		default:
			n.typ = c.intT
		}
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return n.Type()
}

//  InclusiveOrExpression:
//          ExclusiveOrExpression                            // Case InclusiveOrExpressionXor
//  |       InclusiveOrExpression '|' ExclusiveOrExpression  // Case InclusiveOrExpressionOr
func (n *InclusiveOrExpression) check(c *ctx, mode flags) (r Type) {
	if n == nil {
		return Invalid
	}

	defer func() {
		if r == nil || r == Invalid {
			c.errors.add(errorf("TODO %T missed/failed type check", n))
			return
		}
	}()

	switch n.Case {
	case InclusiveOrExpressionXor: // ExclusiveOrExpression
		n.typ = n.ExclusiveOrExpression.check(c, mode)
	case InclusiveOrExpressionOr: // InclusiveOrExpression '|' ExclusiveOrExpression
		mode = mode.add(decay)
		switch a, b := n.InclusiveOrExpression.check(c, mode), n.ExclusiveOrExpression.check(c, mode); {
		case !IsIntegerType(a) || !IsIntegerType(b):
			c.errors.add(errorf("%v: operands shall have integer type: %s and %s", n.Token.Position(), a, b))
		default:
			n.typ = UsualArithmeticConversions(a, b)
		}
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return n.Type()
}

//  ExclusiveOrExpression:
//          AndExpression                            // Case ExclusiveOrExpressionAnd
//  |       ExclusiveOrExpression '^' AndExpression  // Case ExclusiveOrExpressionXor
func (n *ExclusiveOrExpression) check(c *ctx, mode flags) (r Type) {
	if n == nil {
		return Invalid
	}

	defer func() {
		if r == nil || r == Invalid {
			c.errors.add(errorf("TODO %T missed/failed type check", n))
			return
		}
	}()

	switch n.Case {
	case ExclusiveOrExpressionAnd: // AndExpression
		n.typ = n.AndExpression.check(c, mode)
	case ExclusiveOrExpressionXor: // ExclusiveOrExpression '^' AndExpression
		mode = mode.add(decay)
		switch a, b := n.ExclusiveOrExpression.check(c, mode), n.AndExpression.check(c, mode); {
		case !IsIntegerType(a) || !IsIntegerType(b):
			c.errors.add(errorf("%v: operands shall have integer type: %s and %s", n.Token.Position(), a, b))
		default:
			n.typ = UsualArithmeticConversions(a, b)
		}
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return n.Type()
}

//  AndExpression:
//          EqualityExpression                    // Case AndExpressionEq
//  |       AndExpression '&' EqualityExpression  // Case AndExpressionAnd
func (n *AndExpression) check(c *ctx, mode flags) (r Type) {
	if n == nil {
		return Invalid
	}

	defer func() {
		if r == nil || r == Invalid {
			c.errors.add(errorf("TODO %T missed/failed type check", n))
			return
		}
	}()

	switch n.Case {
	case AndExpressionEq: // EqualityExpression
		n.typ = n.EqualityExpression.check(c, mode)
	case AndExpressionAnd: // AndExpression '&' EqualityExpression
		mode = mode.add(decay)
		switch a, b := n.AndExpression.check(c, mode), n.EqualityExpression.check(c, mode); {
		case !IsIntegerType(a) || !IsIntegerType(b):
			c.errors.add(errorf("%v: operands shall have integer type: %s and %s", n.Token.Position(), a, b))
		default:
			n.typ = UsualArithmeticConversions(a, b)
		}
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return n.Type()
}

//  EqualityExpression:
//          RelationalExpression                          // Case EqualityExpressionRel
//  |       EqualityExpression "==" RelationalExpression  // Case EqualityExpressionEq
//  |       EqualityExpression "!=" RelationalExpression  // Case EqualityExpressionNeq
func (n *EqualityExpression) check(c *ctx, mode flags) (r Type) {
	if n == nil {
		return Invalid
	}

	defer func() {
		if r == nil || r == Invalid {
			c.errors.add(errorf("TODO %T missed/failed type check", n))
			return
		}
	}()

	switch n.Case {
	case EqualityExpressionRel: // RelationalExpression
		n.typ = n.RelationalExpression.check(c, mode)
	case
		EqualityExpressionEq,  // EqualityExpression "==" RelationalExpression
		EqualityExpressionNeq: // EqualityExpression "!=" RelationalExpression

		mode = mode.add(decay)
		switch a, b := n.EqualityExpression.check(c, mode), n.RelationalExpression.check(c, mode); {
		case
			// both operands have arithmetic type;
			IsArithmeticType(a) && IsArithmeticType(b),

			// both operands are pointers to qualified or unqualified versions of
			// compatible types;
			//
			// one operand is a pointer to an object or incomplete type and the other is a
			// pointer to a qualified or unqualified version of void;
			isPointerType(a) && isPointerType(b),

			// one operand is a pointer and the other is a null pointer constant.
			isPointerType(a) && IsIntegerType(b) || isPointerType(b) && IsIntegerType(a):

			n.typ = c.intT
		default:
			c.errors.add(errorf("%v: invalid operands: %v and %v", n.Token.Position(), a, b))
		}
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return n.Type()
}

//  RelationalExpression:
//          ShiftExpression                            // Case RelationalExpressionShift
//  |       RelationalExpression '<' ShiftExpression   // Case RelationalExpressionLt
//  |       RelationalExpression '>' ShiftExpression   // Case RelationalExpressionGt
//  |       RelationalExpression "<=" ShiftExpression  // Case RelationalExpressionLeq
//  |       RelationalExpression ">=" ShiftExpression  // Case RelationalExpressionGeq
func (n *RelationalExpression) check(c *ctx, mode flags) (r Type) {
	if n == nil {
		return Invalid
	}

	defer func() {
		if r == nil || r == Invalid {
			c.errors.add(errorf("TODO %T missed/failed type check", n))
			return
		}
	}()

	switch n.Case {
	case RelationalExpressionShift: // ShiftExpression
		n.typ = n.ShiftExpression.check(c, mode)
	case
		RelationalExpressionLt,  // RelationalExpression '<' ShiftExpression
		RelationalExpressionGt,  // RelationalExpression '>' ShiftExpression
		RelationalExpressionLeq, // RelationalExpression "<=" ShiftExpression
		RelationalExpressionGeq: // RelationalExpression ">=" ShiftExpression

		n.typ = c.intT
		mode = mode.add(decay)
		switch a, b := n.RelationalExpression.check(c, mode), n.ShiftExpression.check(c, mode); {
		case
			// both operands have real type;
			IsRealType(a) && IsRealType(b),
			// both operands are pointers to qualified or unqualified versions of
			// compatible object types;
			//
			// both operands are pointers to qualified or unqualified versions of
			// compatible incomplete types
			//
			// gcc allows mixing ints and pointers
			IsScalarType(a) && IsScalarType(b):

			// ok
		default:
			c.errors.add(errorf("%v: invalid operands: %s and %s", n.Token.Position(), a, b))
		}
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return n.Type()
}

//  ShiftExpression:
//          AdditiveExpression                       // Case ShiftExpressionAdd
//  |       ShiftExpression "<<" AdditiveExpression  // Case ShiftExpressionLsh
//  |       ShiftExpression ">>" AdditiveExpression  // Case ShiftExpressionRsh
func (n *ShiftExpression) check(c *ctx, mode flags) (r Type) {
	if n == nil {
		return Invalid
	}

	defer func() {
		if r == nil || r == Invalid {
			c.errors.add(errorf("TODO %T missed/failed type check", n))
			return
		}
	}()

	switch n.Case {
	case ShiftExpressionAdd: // AdditiveExpression
		n.typ = n.AdditiveExpression.check(c, mode)
	case
		ShiftExpressionLsh, // ShiftExpression "<<" AdditiveExpression
		ShiftExpressionRsh: // ShiftExpression ">>" AdditiveExpression

		mode = mode.add(decay)
		switch a, b := n.ShiftExpression.check(c, mode), n.AdditiveExpression.check(c, mode); {
		case !IsScalarType(a) || !IsScalarType(b):
			c.errors.add(errorf("%v: operands shall be a scalars: %s and %s", n.Token.Position(), a, b))
		default:
			n.typ = IntegerPromotion(a)
		}
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return n.Type()
}

//  AdditiveExpression:
//          MultiplicativeExpression                         // Case AdditiveExpressionMul
//  |       AdditiveExpression '+' MultiplicativeExpression  // Case AdditiveExpressionAdd
//  |       AdditiveExpression '-' MultiplicativeExpression  // Case AdditiveExpressionSub
func (n *AdditiveExpression) check(c *ctx, mode flags) (r Type) {
	if n == nil {
		return Invalid
	}

	defer func() {
		if r == nil || r == Invalid {
			c.errors.add(errorf("TODO %T missed/failed type check", n))
			return
		}
	}()

	switch n.Case {
	case AdditiveExpressionMul: // MultiplicativeExpression
		n.typ = n.MultiplicativeExpression.check(c, mode)
	case AdditiveExpressionAdd: // AdditiveExpression '+' MultiplicativeExpression
		mode = mode.add(decay)
		switch a, b := n.AdditiveExpression.check(c, mode), n.MultiplicativeExpression.check(c, mode); {
		case
			// For addition, either both operands shall have arithmetic type
			IsArithmeticType(a) && IsArithmeticType(b):
			n.typ = UsualArithmeticConversions(a, b)
		case
			// or one operand shall be a pointer to an object type and the other shall have
			// integer type.
			isPointerType(a) && IsIntegerType(b):
			n.typ = a
		case IsIntegerType(a) && isPointerType(b):
			n.typ = b
		default:
			c.errors.add(errorf("%v: invalid operands: %s and %s", n.Token.Position(), a, b))
		}
	case AdditiveExpressionSub: // AdditiveExpression '-' MultiplicativeExpression
		mode = mode.add(decay)
		switch a, b := n.AdditiveExpression.check(c, mode), n.MultiplicativeExpression.check(c, mode); {
		case
			// both operands have arithmetic type;
			IsArithmeticType(a) && IsArithmeticType(b):
			n.typ = UsualArithmeticConversions(a, b)
		case
			// both operands are pointers to qualified or unqualified versions of
			// compatible object types;
			isPointerType(a) && isPointerType(b):
			n.typ = c.ptrDiffT(n)
		case
			// the left operand is a pointer to an object type and the right operand has
			// integer type.
			isPointerType(a) && IsIntegerType(b):
			n.typ = a
		default:
			c.errors.add(errorf("%v: invalid operands: %s and %s", n.Token.Position(), a, b))
		}
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return n.Type()
}

//  MultiplicativeExpression:
//          CastExpression                               // Case MultiplicativeExpressionCast
//  |       MultiplicativeExpression '*' CastExpression  // Case MultiplicativeExpressionMul
//  |       MultiplicativeExpression '/' CastExpression  // Case MultiplicativeExpressionDiv
//  |       MultiplicativeExpression '%' CastExpression  // Case MultiplicativeExpressionMod
func (n *MultiplicativeExpression) check(c *ctx, mode flags) (r Type) {
	if n == nil {
		return Invalid
	}

	defer func() {
		if r == nil || r == Invalid {
			c.errors.add(errorf("TODO %T missed/failed type check", n))
			return
		}
	}()

	switch n.Case {
	case MultiplicativeExpressionCast: // CastExpression
		n.typ = n.CastExpression.check(c, mode)
	case
		MultiplicativeExpressionMul, // MultiplicativeExpression '*' CastExpression
		MultiplicativeExpressionDiv: // MultiplicativeExpression '/' CastExpression

		mode = mode.add(decay)
		switch a, b := n.MultiplicativeExpression.check(c, mode), n.CastExpression.check(c, mode); {
		case !IsArithmeticType(a) || !IsArithmeticType(b):
			c.errors.add(errorf("%v: operands shall have arithmetic type: %s and %s", n.Token.Position(), a, b))
		default:
			n.typ = UsualArithmeticConversions(a, b)
		}
	case MultiplicativeExpressionMod: // MultiplicativeExpression '%' CastExpression
		mode = mode.add(decay)
		switch a, b := n.MultiplicativeExpression.check(c, mode), n.CastExpression.check(c, mode); {
		case !IsIntegerType(a) || !IsIntegerType(b):
			c.errors.add(errorf("%v: operands shall have integer type: %s and %s", n.Token.Position(), a, b))
		default:
			n.typ = UsualArithmeticConversions(a, b)
		}
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return n.Type()
}

//  CastExpression:
//          UnaryExpression                  // Case CastExpressionUnary
//  |       '(' TypeName ')' CastExpression  // Case CastExpressionCast
func (n *CastExpression) check(c *ctx, mode flags) (r Type) {
	if n == nil {
		return Invalid
	}

	defer func() {
		if r == nil || r == Invalid {
			c.errors.add(errorf("TODO %T missed/failed type check", n))
			return
		}
	}()

	switch n.Case {
	case CastExpressionUnary: // UnaryExpression
		n.typ = n.UnaryExpression.check(c, mode)
	case CastExpressionCast: // '(' TypeName ')' CastExpression
		n.typ = n.TypeName.check(c)
		n.CastExpression.check(c, mode.add(decay))
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return n.Type()
}

func (n *TypeName) check(c *ctx) (r Type) {
	var dummy bool
	var dummyInt int
	n.typ = n.AbstractDeclarator.check(c, n.SpecifierQualifierList.check(c, &dummy, &dummy, &dummy, &dummy, &dummyInt))
	return n.Type()
}

//  UnaryExpression:
//          PostfixExpression            // Case UnaryExpressionPostfix
//  |       "++" UnaryExpression         // Case UnaryExpressionInc
//  |       "--" UnaryExpression         // Case UnaryExpressionDec
//  |       '&' CastExpression           // Case UnaryExpressionAddrof
//  |       '*' CastExpression           // Case UnaryExpressionDeref
//  |       '+' CastExpression           // Case UnaryExpressionPlus
//  |       '-' CastExpression           // Case UnaryExpressionMinus
//  |       '~' CastExpression           // Case UnaryExpressionCpl
//  |       '!' CastExpression           // Case UnaryExpressionNot
//  |       "sizeof" UnaryExpression     // Case UnaryExpressionSizeofExpr
//  |       "sizeof" '(' TypeName ')'    // Case UnaryExpressionSizeofType
//  |       "&&" IDENTIFIER              // Case UnaryExpressionLabelAddr
//  |       "_Alignof" UnaryExpression   // Case UnaryExpressionAlignofExpr
//  |       "_Alignof" '(' TypeName ')'  // Case UnaryExpressionAlignofType
//  |       "__imag__" UnaryExpression   // Case UnaryExpressionImag
//  |       "__real__" UnaryExpression   // Case UnaryExpressionReal
func (n *UnaryExpression) check(c *ctx, mode flags) (r Type) {
	if n == nil {
		return Invalid
	}

	defer func() {
		if r == nil || r == Invalid {
			c.errors.add(errorf("TODO %T missed/failed type check %v", n, n.Case))
			return
		}
	}()

	switch n.Case {
	case UnaryExpressionPostfix: // PostfixExpression
		n.typ = n.PostfixExpression.check(c, mode)
	case
		UnaryExpressionInc, // "++" UnaryExpression
		UnaryExpressionDec: // "--" UnaryExpression

		n.typ = n.UnaryExpression.check(c, mode.add(decay))
		if !IsRealType(n.Type()) && !isPointerType(n.Type()) {
			c.errors.add(errorf("%v: operand shall have real or pointer type: %s", n.UnaryExpression.Position(), n.Type()))
		}
	case UnaryExpressionAddrof: // '&' CastExpression
		switch t := n.CastExpression.check(c, mode.del(decay)); {
		case
			// The operand of the unary & operator shall be either a function designator,
			t.Kind() == Function,

			// the result of a [] or unary * operator, or an lvalue that designates an
			// object that is not a bit-field and is not declared with the register
			// storage-class specifier.
			isLvalue(t),

			// gcc extension: gcc-9.1.0/gcc/testsuite/gcc.c-torture/compile/20010327-1.c
			t.Kind() == Void:

			n.typ = c.newPointerType(t)
		default:
			c.errors.add(errorf("%v: invalid operand: %s", n.CastExpression.Position(), t))
		}
	case UnaryExpressionDeref: // '*' CastExpression
		switch t := n.CastExpression.check(c, mode.add(decay)); t.Kind() {
		case Ptr:
			n.typ = c.decay(t.(*PointerType).Elem(), mode)
		default:
			c.errors.add(errorf("%v: operand shall be a pointer: %s", n.CastExpression.Position(), t))
		}
	case
		UnaryExpressionPlus,  // '+' CastExpression
		UnaryExpressionMinus: // '-' CastExpression

		n.typ = IntegerPromotion(n.CastExpression.check(c, mode.add(decay)))
		if !IsArithmeticType(n.Type()) {
			c.errors.add(errorf("%v: expected arithmetic type: %s", n.Position(), n.CastExpression.Type()))
		}
	case UnaryExpressionCpl: // '~' CastExpression
		t := n.CastExpression.check(c, mode.add(decay))
		switch {
		case IsIntegerType(t):
			n.typ = IntegerPromotion(t)
		case IsComplexType(t): // GCC extension, complex conjugate
			n.typ = t
		default:
			c.errors.add(errorf("%v: expected integer type: %s", n.Position(), n.CastExpression.Type()))
		}
	case UnaryExpressionNot: // '!' CastExpression
		t := n.CastExpression.check(c, mode.add(decay))
		if !IsScalarType(t) {
			c.errors.add(errorf("%v: expected scalar type: %s", n.Position(), n.CastExpression.Type()))
		}
		n.typ = c.intT
	case UnaryExpressionSizeofExpr: // "sizeof" UnaryExpression
		n.typ = c.sizeT(n)
		t := n.UnaryExpression.check(c, mode.del(decay))
		if t.Kind() == Function {
			t = c.newPointerType(t)
		}
		if t.IsIncomplete() {
			if x, ok := t.(*ArrayType); ok && x.IsVLA() {
				break
			}

			c.errors.add(errorf("%v: sizeof incomplete type: %s", n.UnaryExpression.Position(), t))
		}
		n.val = UInt64Value(t.Size())
	case UnaryExpressionSizeofType: // "sizeof" '(' TypeName ')'
		n.typ = c.sizeT(n)
		t := n.TypeName.check(c)
		if t.IsIncomplete() {
			if x, ok := t.(*ArrayType); ok && x.IsVLA() {
				break
			}

			c.errors.add(errorf("%v: sizeof incomplete type: %s", n.TypeName.Position(), t))
		}
		n.val = UInt64Value(t.Size())
	case UnaryExpressionLabelAddr: // "&&" IDENTIFIER
		n.typ = c.pvoidT
	case UnaryExpressionAlignofExpr: // "_Alignof" UnaryExpression
		t := n.UnaryExpression.check(c, mode.add(decay))
		n.val, n.typ = UInt64Value(t.Align()), c.sizeT(n)
	case UnaryExpressionAlignofType: // "_Alignof" '(' TypeName ')'
		t := n.TypeName.check(c)
		n.val, n.typ = UInt64Value(t.Align()), c.sizeT(n)
	case
		UnaryExpressionImag, // "__imag__" UnaryExpression
		UnaryExpressionReal: // "__real__" UnaryExpression

		t := n.UnaryExpression.check(c, mode.add(decay))
		if !IsComplexType(t) {
			c.errors.add(errorf("%v: expected complex type: %s", n.Position(), t))
			break
		}

		n.typ = c.ast.kinds[correspondingRealKinds[t.Kind()]]
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return n.Type()
}

//  PostfixExpression:
//          PrimaryExpression                                 // Case PostfixExpressionPrimary
//  |       PostfixExpression '[' ExpressionList ']'          ;// Case PostfixExpressionIndex
//  |       PostfixExpression '(' ArgumentExpressionList ')'  // Case PostfixExpressionCall
//  |       PostfixExpression '.' IDENTIFIER                  // Case PostfixExpressionSelect
//  |       PostfixExpression "->" IDENTIFIER                 // Case PostfixExpressionPSelect
//  |       PostfixExpression "++"                            // Case PostfixExpressionInc
//  |       PostfixExpression "--"                            // Case PostfixExpressionDec
//  |       '(' TypeName ')' '{' InitializerList ',' '}'      // Case PostfixExpressionComplit
func (n *PostfixExpression) check(c *ctx, mode flags) (r Type) {
	if n == nil {
		return Invalid
	}

	defer func() {
		if r == nil || r == Invalid {
			c.errors.add(errorf("TODO %T missed/failed type check %v", n, n.Case))
			return
		}

		r = c.decay(r, mode)
		n.typ = r
	}()

out:
	switch n.Case {
	case PostfixExpressionPrimary: // PrimaryExpression
		n.typ = n.PrimaryExpression.check(c, mode)
	case PostfixExpressionIndex: // PostfixExpression '[' ExpressionList ']'
		// One of the expressions shall have type ‘‘pointer to object type’’, the other
		// expression shall have integer type, and the result has type ‘‘type’’.
		mode = mode.add(decay)
		switch t1, t2 := n.PostfixExpression.check(c, mode), n.ExpressionList.check(c, mode); {
		case isPointerType(t1) && IsIntegerType(t2):
			n.typ = t1.(*PointerType).Elem()
		case isPointerType(t2) && IsIntegerType(t1):
			n.typ = t2.(*PointerType).Elem()
		case isVectorType(t1) && IsIntegerType(t2):
			n.typ = t1
		case isVectorType(t2) && IsIntegerType(t1):
			n.typ = t2
		default:
			c.errors.add(errorf("%v: one of the expressions shall be a pointer and the other shall have integer type: %s and %s", n.Token.Position(), t1, t2))
			n.typ = c.intT
		}
	case PostfixExpressionCall: // PostfixExpression '(' ArgumentExpressionList ')'
		switch isName(n.PostfixExpression) {
		case "__builtin_types_compatible_p_impl":
			n.typ = c.intT
			args := n.ArgumentExpressionList.check(c, 0)
			if len(args) != 2 {
				c.errors.add(errorf("%v: expected two arguments: (%s)", n.Position(), NodeSource(n.ArgumentExpressionList)))
				break out
			}

			n.val = bool2int(args[0].Type().isCompatible(args[1].Type()))
			break out
		case "__builtin_constant_p":
			n.typ = c.intT
			args := n.ArgumentExpressionList.check(c, decay)
			if len(args) != 1 {
				c.errors.add(errorf("%v: expected one argument: (%s)", n.Position(), NodeSource(n.ArgumentExpressionList)))
				break out
			}

			n.val = bool2int(args[0].Value() != Unknown)
			break out
		default:
			mode = mode.add(implicitFuncDef)
		}
		t := n.PostfixExpression.check(c, mode.add(decay))
		n.ArgumentExpressionList.check(c, decay)
		if t == nil || mode.has(asmArgList) {
			n.typ = c.intT
			break
		}

		if t.Kind() != Ptr {
			c.errors.add(errorf("%v: expected pointer to function: %s", n.Position(), t))
			break
		}

		pt := t.(*PointerType)
		if pt.Elem().Kind() != Function {
			c.errors.add(errorf("%v: expected pointer to function: %s", n.Position(), pt))
			break
		}

		ft := pt.Elem().(*FunctionType)
		n.typ = ft.Result()
	case PostfixExpressionSelect: // PostfixExpression '.' IDENTIFIER
		nm := n.Token2.SrcStr()
		switch t := n.PostfixExpression.check(c, mode.add(decay)); t.Kind() {
		case Struct:
			st := t.(*StructType)
			f := st.FieldByName(nm)
			if f == nil {
				c.errors.add(errorf("%v: type %s has no member named %s", n.Token.Position(), st, nm))
				break
			}

			n.typ = f.Type()
			n.field = f
		case Union:
			st := t.(*UnionType)
			f := st.FieldByName(nm)
			if f == nil {
				c.errors.add(errorf("%v: type %s has no member named %s", n.Token.Position(), st, nm))
				break
			}

			n.typ = f.Type()
			n.field = f
		default:
			c.errors.add(errorf("%v: expected a struct or union: %s", n.Token.Position(), t))
		}
	case PostfixExpressionPSelect: // PostfixExpression "->" IDENTIFIER
		nm := n.Token2.SrcStr()
		switch t := n.PostfixExpression.check(c, mode.add(decay)); t.Kind() {
		case Ptr:
			switch et := t.(*PointerType).Elem(); et.Kind() {
			case Struct:
				st := et.(*StructType)
				f := st.FieldByName(nm)
				if f == nil {
					c.errors.add(errorf("%v: type %s has no member named %s", n.Token.Position(), st, nm))
					break
				}

				n.typ = f.Type()
				n.field = f
			case Union:
				st := et.(*UnionType)
				f := st.FieldByName(nm)
				if f == nil {
					c.errors.add(errorf("%v: type %s has no member named %s", n.Token.Position(), st, nm))
					break
				}

				n.typ = f.Type()
				n.field = f
			default:
				c.errors.add(errorf("%v: expected a pointer to struct or union: %s", n.Token.Position(), t))
			}
		default:
			c.errors.add(errorf("%v: expected a pointer: %s", n.Token.Position(), t))
		}
	case
		PostfixExpressionInc, // PostfixExpression "++"
		PostfixExpressionDec: // PostfixExpression "--"
		switch t := n.PostfixExpression.check(c, mode.add(decay)); {
		case
			// The operand of the postfix increment or decrement operator shall have
			// qualified or unqualified real or pointer type and shall be a modifiable
			// lvalue.
			realKinds[t.Kind()] || isPointerType(t):

			n.typ = t
			if !isModifiableLvalue(t) {
				c.errors.add(errorf("%v: operand shall be a modifiable lvalue: %s", n.PostfixExpression.Position(), t))
			}
		default:
			c.errors.add(errorf("%v: invalid operand: %s", n.PostfixExpression.Position(), t))
		}
	case PostfixExpressionComplit: // '(' TypeName ')' '{' InitializerList ',' '}'
		n.typ = n.TypeName.check(c)
		if n.InitializerList == nil {
			n.val = Zero
			break
		}

		n.InitializerList.check(c, n.Type(), 0, true)
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return n.Type()
}

func isName(n Node) string {
	for {
		switch x := n.(type) {
		case *ExpressionList:
			if x.ExpressionList != nil {
				return ""
			}

			n = x.AssignmentExpression
		case *PrimaryExpression:
			if x.Case == PrimaryExpressionIdent {
				return x.Token.SrcStr()
			}

			return ""
		default:
			return ""
		}
	}
}

//  PrimaryExpression:
//          IDENTIFIER                 // Case PrimaryExpressionIdent
//  |       INTCONST                   // Case PrimaryExpressionInt
//  |       FLOATCONST                 // Case PrimaryExpressionFloat
//  |       CHARCONST                  // Case PrimaryExpressionChar
//  |       LONGCHARCONST              // Case PrimaryExpressionLChar
//  |       STRINGLITERAL              // Case PrimaryExpressionString
//  |       LONGSTRINGLITERAL          // Case PrimaryExpressionLString
//  |       '(' ExpressionList ')'     // Case PrimaryExpressionExpr
//  |       '(' CompoundStatement ')'  // Case PrimaryExpressionStmt
//  |       GenericSelection           // Case PrimaryExpressionGeneric
func (n *PrimaryExpression) check(c *ctx, mode flags) (r Type) {
	if n == nil {
		return Invalid
	}

	defer func() {
		if r == nil || r == Invalid {
			if !mode.has(ignoreUndefined) {
				c.errors.add(errorf("TODO %T missed/failed type check %v", n, n.Case))
			}
			return
		}

		if n.m != nil && n.m.Value() == Unknown && n.Value() != Unknown {
			n.m.val = n.val
			n.m.typ = n.typ
		}
		n.typ = c.decay(r, mode)
		r = n.Type()
	}()

out:
	switch n.Case {
	case PrimaryExpressionIdent: // IDENTIFIER
		var d *Declarator
		switch x := n.LexicalScope().ident(n.Token).(type) {
		case *Declarator:
			d = x
			n.typ = d.Type()
		case *Enumerator:
			n.resolvedTo = x
			n.val = x.val
			n.typ = x.Type()
			break out
		case *Parameter:
			n.resolvedTo = x
			n.typ = x.Type()
			break out
		default:
			d = n.LexicalScope().builtin(n.Token)
			if d == nil {
				if mode.has(implicitFuncDef) {
					n.typ = c.implicitFunc
					break out
				}

				if mode.has(ignoreUndefined) {
					n.typ = c.intT
					break out
				}

				c.errors.add(errorf("%v: undefined: %s", n.Position(), n.Token.Src()))
				break out
			}
		}

		n.resolvedTo = d
		n.typ = d.Type()
	case PrimaryExpressionInt: // INTCONST
		n.val, n.typ = n.intConst(c)
	case PrimaryExpressionFloat: // FLOATCONST
		n.val, n.typ = n.floatConst(c)
	case PrimaryExpressionChar: // CHARCONST
		n.val, n.typ = n.charConst(c)
	case PrimaryExpressionLChar: // LONGCHARCONST
		n.typ = c.wcharT(n)
		n.val, n.typ = n.charConst(c)
	case PrimaryExpressionString: // STRINGLITERAL
		n.val, n.typ = n.stringConst(c)
	case PrimaryExpressionLString: // LONGSTRINGLITERAL
		n.val, n.typ = n.stringConst(c)
	case PrimaryExpressionExpr: // '(' ExpressionList ')'
		n.typ = n.ExpressionList.check(c, mode)
	case PrimaryExpressionStmt: // '(' CompoundStatement ')'
		n.typ = n.CompoundStatement.check(c)
	case PrimaryExpressionGeneric: // GenericSelection
		n.typ = n.GenericSelection.check(c, mode)
	default:
		c.errors.add(errorf("internal error: %v", n.Case))
	}
	return n.Type()
}

func (n *GenericSelection) check(c *ctx, mode flags) (r Type) {
	n.assoc, n.typ = n.GenericAssociationList.check(c, mode, n.AssignmentExpression.check(c, mode.add(decay)))
	return n.Type()
}

func (n *GenericAssociationList) check(c *ctx, mode flags, ctrl Type) (assoc *GenericAssociation, r Type) {
	n0 := n
	var deflt *GenericAssociation
	for ; n != nil; n = n.GenericAssociationList {
		switch assoc = n.GenericAssociation; assoc.Case {
		case GenericAssociationType: // TypeName ':' AssignmentExpression
			if t := assoc.TypeName.check(c); ctrl.isCompatible(t) {
				return assoc, assoc.AssignmentExpression.check(c, decay)
			}
		case GenericAssociationDefault: //  "default" ':' AssignmentExpression
			if deflt != nil {
				c.errors.add(errorf("%v: multiple defaults, previous at %v", assoc.Position(), deflt.Position()))
				break
			}

			assoc.AssignmentExpression.check(c, decay)
			deflt = assoc
		default:
			c.errors.add(errorf("TODO internal error: %v", assoc.Case))
		}
	}
	if deflt != nil {
		return deflt, deflt.AssignmentExpression.Type()
	}

	c.errors.add(errorf("%v: failed to find a matching association for %s", n0.Position(), ctrl))
	return nil, Invalid
}

func (n *PrimaryExpression) floatConst(c *ctx) (v Value, t Type) {
	s := strings.ToLower(n.Token.SrcStr())
	val, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return Float64Value(val), c.ast.kinds[Double]
	}

	// https://gcc.gnu.org/onlinedocs/gcc/Decimal-Float.html
	//
	// Use a suffix ‘df’ or ‘DF’ in a literal constant of type _Decimal32, ‘dd’ or
	// ‘DD’ for _Decimal64, and ‘dl’ or ‘DL’ for _Decimal128.

	// https://gcc.gnu.org/onlinedocs/gcc-9.1.0/gcc/Floating-Types.html
	//
	// Constants with these types use suffixes fn or Fn and fnx or Fnx.

	var bf *big.Float
	// Longer suffixes must come first.
	switch {
	case strings.HasSuffix(s, "f128x"):
		bf, _, err = big.ParseFloat(s[:len(s)-len("f128x")], 0, longDoublePrec, big.ToNearestEven)
		if err == nil {
			return (*LongDoubleValue)(bf), c.ast.kinds[Float128x]
		}
	case strings.HasSuffix(s, "f32x"):
		bf, _, err = big.ParseFloat(s[:len(s)-len("f32x")], 0, longDoublePrec, big.ToNearestEven)
		if err == nil {
			return (*LongDoubleValue)(bf), c.ast.kinds[Float32x]
		}
	case strings.HasSuffix(s, "f64x"):
		bf, _, err = big.ParseFloat(s[:len(s)-len("f64x")], 0, longDoublePrec, big.ToNearestEven)
		if err == nil {
			return (*LongDoubleValue)(bf), c.ast.kinds[Float64x]
		}
	case strings.HasSuffix(s, "f128"):
		bf, _, err = big.ParseFloat(s[:len(s)-len("f128")], 0, longDoublePrec, big.ToNearestEven)
		if err == nil {
			return (*LongDoubleValue)(bf), c.ast.kinds[Float128]
		}
	case strings.HasSuffix(s, "f16"):
		bf, _, err = big.ParseFloat(s[:len(s)-len("f16")], 0, longDoublePrec, big.ToNearestEven)
		if err == nil {
			return (*LongDoubleValue)(bf), c.ast.kinds[Float16]
		}
	case strings.HasSuffix(s, "f32"):
		bf, _, err = big.ParseFloat(s[:len(s)-len("f32")], 0, longDoublePrec, big.ToNearestEven)
		if err == nil {
			return (*LongDoubleValue)(bf), c.ast.kinds[Float32]
		}
	case strings.HasSuffix(s, "f64"):
		bf, _, err = big.ParseFloat(s[:len(s)-len("f64")], 0, longDoublePrec, big.ToNearestEven)
		if err == nil {
			return (*LongDoubleValue)(bf), c.ast.kinds[Float64]
		}
	case
		strings.HasSuffix(s, "il"),
		strings.HasSuffix(s, "jl"),
		strings.HasSuffix(s, "li"),
		strings.HasSuffix(s, "lj"):

		bf, _, err = big.ParseFloat(s[:len(s)-len("il")], 0, longDoublePrec, big.ToNearestEven)
		if err == nil {
			return &ComplexLongDoubleValue{Im: (*LongDoubleValue)(bf)}, c.ast.kinds[LongDouble]
		}
	case
		strings.HasSuffix(s, "fi"),
		strings.HasSuffix(s, "fj"),
		strings.HasSuffix(s, "if"),
		strings.HasSuffix(s, "jf"):

		if val, err = strconv.ParseFloat(s[:len(s)-len("fi")], 64); err == nil {
			return Complex64Value(complex(0, val)), c.ast.kinds[Float]
		}
	case strings.HasSuffix(s, "df"):
		bf, _, err = big.ParseFloat(s[:len(s)-len("df")], 0, longDoublePrec, big.ToNearestEven)
		if err == nil {
			return (*LongDoubleValue)(bf), c.ast.kinds[Decimal32]
		}
	case strings.HasSuffix(s, "dd"):
		bf, _, err = big.ParseFloat(s[:len(s)-len("dd")], 0, longDoublePrec, big.ToNearestEven)
		if err == nil {
			return (*LongDoubleValue)(bf), c.ast.kinds[Decimal64]
		}
	case strings.HasSuffix(s, "dl"):
		bf, _, err = big.ParseFloat(s[:len(s)-len("dl")], 0, longDoublePrec, big.ToNearestEven)
		if err == nil {
			return (*LongDoubleValue)(bf), c.ast.kinds[Decimal128]
		}
	case strings.HasSuffix(s, "f"):
		if val, err = strconv.ParseFloat(s[:len(s)-len("f")], 64); err == nil {
			return Float64Value(val), c.ast.kinds[Float]
		}
	case strings.HasSuffix(s, "l"):
		bf, _, err = big.ParseFloat(s[:len(s)-len("l")], 0, longDoublePrec, big.ToNearestEven)
		if err == nil {
			return (*LongDoubleValue)(bf), c.ast.kinds[LongDouble]
		}
	case strings.HasSuffix(s, "i"), strings.HasSuffix(s, "j"):
		if val, err = strconv.ParseFloat(s[:len(s)-len("i")], 64); err == nil {
			return Complex128Value(complex(0, val)), c.ast.kinds[ComplexDouble]
		}
	}
	c.errors.add(errorf("TODO %T %v %s FERR %v", n, n.Case, s, err))
	return Unknown, Invalid
}

func (n *PrimaryExpression) stringConst(c *ctx) (v Value, t Type) {
	s := stringConst(func(msg string, args ...interface{}) {
		c.errors.add(errorf(msg, args...))
	}, n.Token)
	switch n.Case {
	case PrimaryExpressionString:
		return StringValue(s), c.newArrayType(c.ast.kinds[Char], int64(len(s)), nil)
	case PrimaryExpressionLString:
		switch t = c.wcharT(n); t.Size() {
		case 2:
			v := UTF16StringValue(utf16.Encode([]rune(s)))
			return v, c.newArrayType(t, int64(len(v)), nil)
		case 4:
			v := UTF32StringValue([]rune(s))
			return v, c.newArrayType(t, int64(len(v)), nil)
		}
	default:
		c.errors.add(errorf("TODO %v", n.Case))
	}
	return n.Value(), n.Type()
}

func (n *PrimaryExpression) charConst(c *ctx) (v Value, t Type) {
	n.typ = c.intT
	switch n.Case {
	case PrimaryExpressionLChar:
		n.typ = c.wcharT(n)
		fallthrough
	case PrimaryExpressionChar:
		r := charConst(func(msg string, args ...interface{}) {
			c.errors.add(errorf(msg, args...))
		}, n.Token)
		n.val = Int64Value(r)
	default:
		c.errors.add(errorf("TODO %v", n.Case))
	}
	return n.Value(), n.Type()
}

func (n *PrimaryExpression) intConst(c *ctx) (v Value, t Type) {
	s0 := n.Token.SrcStr()
	s := strings.TrimRight(s0, "uUlL")
	prefix := 0
	var base int
	switch {
	case strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X"):
		prefix = 2
		base = 16
	case strings.HasPrefix(s, "0b") || strings.HasPrefix(s, "0B"):
		prefix = 2
		base = 2
	case strings.HasPrefix(s, "0"):
		base = 8
	default:
		base = 10
	}
	s = s[prefix:]
	val, err := strconv.ParseUint(s, base, 64)
	if err != nil {
		c.errors.add(errorf("%v: %v", n.Position(), err))
		return Unknown, c.intT
	}

	suffix := s0[prefix+len(s):]
	switch suffix = strings.ToLower(suffix); suffix {
	case "":
		if base == 10 {
			return n.intConst2(c, s0, val, Int, Long, LongLong)
		}

		return n.intConst2(c, s0, val, Int, UInt, Long, ULong, LongLong, ULongLong)
	case "u":
		return n.intConst2(c, s0, val, UInt, ULong, ULongLong)
	case "l":
		if base == 10 {
			return n.intConst2(c, s0, val, Long, LongLong)
		}

		return n.intConst2(c, s0, val, Long, ULong, LongLong, ULongLong)
	case "lu", "ul":
		return n.intConst2(c, s0, val, ULong, ULongLong)
	case "ll":
		if base == 10 {
			return n.intConst2(c, s0, val, LongLong)
		}

		return n.intConst2(c, s0, val, LongLong, ULongLong)
	case "llu", "ull":
		return n.intConst2(c, s0, val, ULongLong)
	default:
		c.errors.add(errorf("%v: invalid suffix", n.Position()))
		return Unknown, c.intT
	}
}

func (n *PrimaryExpression) intConst2(c *ctx, s string, val uint64, list ...Kind) (v Value, t Type) {
	abi := c.ast.ABI
	b := bits.Len64(val)
	for _, k := range list {
		sign := 0
		if abi.isSignedInteger(k) {
			sign = 1
		}
		if int(abi.types[k].size)*8 >= b+sign {
			switch {
			case sign == 0:
				return UInt64Value(val), c.ast.kinds[k]
			default:
				return Int64Value(val), c.ast.kinds[k]
			}

		}
	}

	c.errors.add(errorf("%v: invalid integer constant", n.Position()))
	return Unknown, c.intT
}

//  ExpressionList:
//          AssignmentExpression
//  |       ExpressionList ',' AssignmentExpression
func (n *ExpressionList) check(c *ctx, mode flags) (r Type) {
	if n == nil {
		return Invalid
	}

	n0 := n
	defer func() {
		if r == nil || r == Invalid {
			c.errors.add(errorf("TODO %T missed/failed type check", n))
		}
	}()

	for ; n != nil; n = n.ExpressionList {
		n0.typ = n.AssignmentExpression.check(c, mode)
	}
	return n0.Type()
}

//  ConstantExpression:
//          ConditionalExpression
func (n *ConstantExpression) check(c *ctx, mode flags) (r Type) {
	if n == nil {
		return Invalid
	}

	defer func() {
		if r == nil || r == Invalid {
			c.errors.add(errorf("TODO %T missed/failed type check", n))
		}
	}()

	n.typ = n.ConditionalExpression.check(c, mode)
	if n.val = n.ConditionalExpression.eval(c, mode); n.Value() == Unknown {
		c.errors.add(errorf("%v: cannot evaluate constant expression: %s", n.Position(), NodeSource(n)))
	}
	return n.Type()
}
