// Copyright 2019 The CC Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cc // import "modernc.org/cc/v3"

import (
	"fmt"
	"math"
)

var (
	_ Value = Complex128Value(0)
	_ Value = Complex64Value(0)
	_ Value = Float32Value(0)
	_ Value = Float64Value(0)
	_ Value = Int64Value(0)
	_ Value = StringValue(0)
	_ Value = Uint64Value(0)
	_ Value = WideStringValue(0)

	_ Operand = (*funcDesignator)(nil)
	_ Operand = (*lvalue)(nil)
	_ Operand = (*operand)(nil)
	_ Operand = noOperand

	noOperand = &operand{typ: noType}
)

type Operand interface {
	Declarator() *Declarator
	IsLValue() bool
	IsNonZero() bool
	IsZero() bool
	Type() Type
	Value() Value
	convertFromInt(*context, Node, Type) Operand
	convertTo(*context, Node, Type) Operand
	convertToInt(*context, Node, Type) Operand
	integerPromotion(*context, Node) Operand
	normalize(*context) Operand
}

type Value interface {
	add(b Value) Value
	and(b Value) Value
	div(b Value) Value
	eq(b Value) Value
	ge(b Value) Value
	gt(b Value) Value
	isNonZero() bool
	isZero() bool
	le(b Value) Value
	lsh(b Value) Value
	lt(b Value) Value
	mod(b Value) Value
	mul(b Value) Value
	neg() Value
	neq(b Value) Value
	or(b Value) Value
	rsh(b Value) Value
	sub(b Value) Value
	xor(b Value) Value
}

type WideStringValue StringID

func (v WideStringValue) eq(b Value) Value  { return boolValue(v == b.(WideStringValue)) }
func (v WideStringValue) neq(b Value) Value { return boolValue(v != b.(WideStringValue)) }
func (v WideStringValue) add(b Value) Value { panic("internal error") } //TODOOK
func (v WideStringValue) and(b Value) Value { panic("internal error") } //TODOOK
func (v WideStringValue) div(b Value) Value { panic("internal error") } //TODOOK
func (v WideStringValue) isNonZero() bool   { return true }
func (v WideStringValue) isZero() bool      { return false }
func (v WideStringValue) lsh(b Value) Value { panic("internal error") } //TODOOK
func (v WideStringValue) mod(b Value) Value { panic("internal error") } //TODOOK
func (v WideStringValue) mul(b Value) Value { panic("internal error") } //TODOOK
func (v WideStringValue) neg() Value        { panic("internal error") } //TODOOK
func (v WideStringValue) or(b Value) Value  { panic("internal error") } //TODOOK
func (v WideStringValue) rsh(b Value) Value { panic("internal error") } //TODOOK
func (v WideStringValue) sub(b Value) Value { panic("internal error") } //TODOOK
func (v WideStringValue) xor(b Value) Value { panic("internal error") } //TODOOK

func (v WideStringValue) le(b Value) Value {
	return boolValue(StringID(v).String() <= StringID(b.(WideStringValue)).String())
}

func (v WideStringValue) ge(b Value) Value {
	return boolValue(StringID(v).String() >= StringID(b.(WideStringValue)).String())
}

func (v WideStringValue) gt(b Value) Value {
	return boolValue(StringID(v).String() > StringID(b.(WideStringValue)).String())
}

func (v WideStringValue) lt(b Value) Value {
	return boolValue(StringID(v).String() < StringID(b.(WideStringValue)).String())
}

type StringValue StringID

func (v StringValue) eq(b Value) Value  { return boolValue(v == b.(StringValue)) }
func (v StringValue) neq(b Value) Value { return boolValue(v != b.(StringValue)) }
func (v StringValue) add(b Value) Value { panic("internal error") } //TODOOK
func (v StringValue) and(b Value) Value { panic("internal error") } //TODOOK
func (v StringValue) div(b Value) Value { panic("internal error") } //TODOOK
func (v StringValue) isNonZero() bool   { return true }
func (v StringValue) isZero() bool      { return false }
func (v StringValue) lsh(b Value) Value { panic("internal error") } //TODOOK
func (v StringValue) mod(b Value) Value { panic("internal error") } //TODOOK
func (v StringValue) mul(b Value) Value { panic("internal error") } //TODOOK
func (v StringValue) neg() Value        { panic("internal error") } //TODOOK
func (v StringValue) or(b Value) Value  { panic("internal error") } //TODOOK
func (v StringValue) rsh(b Value) Value { panic("internal error") } //TODOOK
func (v StringValue) sub(b Value) Value { panic("internal error") } //TODOOK
func (v StringValue) xor(b Value) Value { panic("internal error") } //TODOOK

func (v StringValue) le(b Value) Value {
	return boolValue(StringID(v).String() <= StringID(b.(StringValue)).String())
}

func (v StringValue) ge(b Value) Value {
	return boolValue(StringID(v).String() >= StringID(b.(StringValue)).String())
}

func (v StringValue) gt(b Value) Value {
	return boolValue(StringID(v).String() > StringID(b.(StringValue)).String())
}

func (v StringValue) lt(b Value) Value {
	return boolValue(StringID(v).String() < StringID(b.(StringValue)).String())
}

type Int64Value int64

func (v Int64Value) add(b Value) Value { return v + b.(Int64Value) }
func (v Int64Value) and(b Value) Value { return v & b.(Int64Value) }
func (v Int64Value) eq(b Value) Value  { return boolValue(v == b.(Int64Value)) }
func (v Int64Value) ge(b Value) Value  { return boolValue(v >= b.(Int64Value)) }
func (v Int64Value) gt(b Value) Value  { return boolValue(v > b.(Int64Value)) }
func (v Int64Value) isNonZero() bool   { return v != 0 }
func (v Int64Value) isZero() bool      { return v == 0 }
func (v Int64Value) le(b Value) Value  { return boolValue(v <= b.(Int64Value)) }
func (v Int64Value) lt(b Value) Value  { return boolValue(v < b.(Int64Value)) }
func (v Int64Value) mul(b Value) Value { return v * b.(Int64Value) }
func (v Int64Value) neg() Value        { return -v }
func (v Int64Value) neq(b Value) Value { return boolValue(v != b.(Int64Value)) }
func (v Int64Value) or(b Value) Value  { return v | b.(Int64Value) }
func (v Int64Value) sub(b Value) Value { return v - b.(Int64Value) }
func (v Int64Value) xor(b Value) Value { return v ^ b.(Int64Value) }

func (v Int64Value) div(b Value) Value {
	if b.isZero() {
		return nil
	}

	return v / b.(Int64Value)
}

func (v Int64Value) lsh(b Value) Value {
	switch y := b.(type) {
	case Int64Value:
		return v << uint64(y)
	case Uint64Value:
		return v << y
	default:
		panic("internal error") //TODOOK
	}
}

func (v Int64Value) rsh(b Value) Value {
	switch y := b.(type) {
	case Int64Value:
		return v >> uint64(y)
	case Uint64Value:
		return v >> y
	default:
		panic("internal error") //TODOOK
	}
}

func (v Int64Value) mod(b Value) Value {
	if b.isZero() {
		return nil
	}

	return v % b.(Int64Value)
}

type Uint64Value uint64

func (v Uint64Value) add(b Value) Value { return v + b.(Uint64Value) }
func (v Uint64Value) and(b Value) Value { return v & b.(Uint64Value) }
func (v Uint64Value) eq(b Value) Value  { return boolValue(v == b.(Uint64Value)) }
func (v Uint64Value) ge(b Value) Value  { return boolValue(v >= b.(Uint64Value)) }
func (v Uint64Value) gt(b Value) Value  { return boolValue(v > b.(Uint64Value)) }
func (v Uint64Value) isNonZero() bool   { return v != 0 }
func (v Uint64Value) isZero() bool      { return v == 0 }
func (v Uint64Value) le(b Value) Value  { return boolValue(v <= b.(Uint64Value)) }
func (v Uint64Value) lt(b Value) Value  { return boolValue(v < b.(Uint64Value)) }
func (v Uint64Value) mul(b Value) Value { return v * b.(Uint64Value) }
func (v Uint64Value) neg() Value        { return -v }
func (v Uint64Value) neq(b Value) Value { return boolValue(v != b.(Uint64Value)) }
func (v Uint64Value) or(b Value) Value  { return v | b.(Uint64Value) }
func (v Uint64Value) sub(b Value) Value { return v - b.(Uint64Value) }
func (v Uint64Value) xor(b Value) Value { return v ^ b.(Uint64Value) }

func (v Uint64Value) div(b Value) Value {
	if b.isZero() {
		return nil
	}

	return v / b.(Uint64Value)
}

func (v Uint64Value) lsh(b Value) Value {
	switch y := b.(type) {
	case Int64Value:
		return v << uint64(y)
	case Uint64Value:
		return v << y
	default:
		panic("internal error") //TODOOK
	}
}

func (v Uint64Value) rsh(b Value) Value {
	switch y := b.(type) {
	case Int64Value:
		return v >> uint64(y)
	case Uint64Value:
		return v >> y
	default:
		panic("internal error") //TODOOK
	}
}

func (v Uint64Value) mod(b Value) Value {
	if b.isZero() {
		return nil
	}

	return v % b.(Uint64Value)
}

type Float32Value float32

func (v Float32Value) add(b Value) Value { return v + b.(Float32Value) }
func (v Float32Value) and(b Value) Value { panic("internal error") } //TODOOK
func (v Float32Value) div(b Value) Value { return v / b.(Float32Value) }
func (v Float32Value) eq(b Value) Value  { return boolValue(v == b.(Float32Value)) }
func (v Float32Value) ge(b Value) Value  { return boolValue(v >= b.(Float32Value)) }
func (v Float32Value) gt(b Value) Value  { return boolValue(v > b.(Float32Value)) }
func (v Float32Value) isNonZero() bool   { return v != 0 }
func (v Float32Value) isZero() bool      { return v == 0 }
func (v Float32Value) le(b Value) Value  { return boolValue(v <= b.(Float32Value)) }
func (v Float32Value) lsh(b Value) Value { panic("internal error") } //TODOOK
func (v Float32Value) lt(b Value) Value  { return boolValue(v < b.(Float32Value)) }
func (v Float32Value) mod(b Value) Value { panic("internal error") } //TODOOK
func (v Float32Value) mul(b Value) Value { return v * b.(Float32Value) }
func (v Float32Value) neg() Value        { return -v }
func (v Float32Value) neq(b Value) Value { return boolValue(v != b.(Float32Value)) }
func (v Float32Value) or(b Value) Value  { panic("internal error") } //TODOOK
func (v Float32Value) rsh(b Value) Value { panic("internal error") } //TODOOK
func (v Float32Value) sub(b Value) Value { return v - b.(Float32Value) }
func (v Float32Value) xor(b Value) Value { panic("internal error") } //TODOOK

type Float64Value float64

func (v Float64Value) add(b Value) Value { return v + b.(Float64Value) }
func (v Float64Value) and(b Value) Value { panic("internal error") } //TODOOK
func (v Float64Value) div(b Value) Value { return v / b.(Float64Value) }
func (v Float64Value) eq(b Value) Value  { return boolValue(v == b.(Float64Value)) }
func (v Float64Value) ge(b Value) Value  { return boolValue(v >= b.(Float64Value)) }
func (v Float64Value) gt(b Value) Value  { return boolValue(v > b.(Float64Value)) }
func (v Float64Value) isNonZero() bool   { return v != 0 }
func (v Float64Value) isZero() bool      { return v == 0 }
func (v Float64Value) le(b Value) Value  { return boolValue(v <= b.(Float64Value)) }
func (v Float64Value) lsh(b Value) Value { panic("internal error") } //TODOOK
func (v Float64Value) lt(b Value) Value  { return boolValue(v < b.(Float64Value)) }
func (v Float64Value) mod(b Value) Value { panic("internal error") } //TODOOK
func (v Float64Value) mul(b Value) Value { return v * b.(Float64Value) }
func (v Float64Value) neg() Value        { return -v }
func (v Float64Value) neq(b Value) Value { return boolValue(v != b.(Float64Value)) }
func (v Float64Value) or(b Value) Value  { panic("internal error") } //TODOOK
func (v Float64Value) rsh(b Value) Value { panic("internal error") } //TODOOK
func (v Float64Value) sub(b Value) Value { return v - b.(Float64Value) }
func (v Float64Value) xor(b Value) Value { panic("internal error") } //TODOOK

type Complex64Value complex128

func (v Complex64Value) add(b Value) Value { return v + b.(Complex64Value) }
func (v Complex64Value) and(b Value) Value { panic("internal error") } //TODOOK
func (v Complex64Value) div(b Value) Value { return v / b.(Complex64Value) }
func (v Complex64Value) eq(b Value) Value  { return boolValue(v == b.(Complex64Value)) }
func (v Complex64Value) ge(b Value) Value  { panic("internal error") } //TODOOK }
func (v Complex64Value) gt(b Value) Value  { panic("internal error") } //TODOOK }
func (v Complex64Value) isNonZero() bool   { return v != 0 }
func (v Complex64Value) isZero() bool      { return v == 0 }
func (v Complex64Value) le(b Value) Value  { panic("internal error") } //TODOOK }
func (v Complex64Value) lsh(b Value) Value { panic("internal error") } //TODOOK
func (v Complex64Value) lt(b Value) Value  { panic("internal error") } //TODOOK }
func (v Complex64Value) mod(b Value) Value { panic("internal error") } //TODOOK
func (v Complex64Value) mul(b Value) Value { return v * b.(Complex64Value) }
func (v Complex64Value) neg() Value        { return -v }
func (v Complex64Value) neq(b Value) Value { return boolValue(v != b.(Complex64Value)) }
func (v Complex64Value) or(b Value) Value  { panic("internal error") } //TODOOK
func (v Complex64Value) rsh(b Value) Value { panic("internal error") } //TODOOK
func (v Complex64Value) sub(b Value) Value { return v - b.(Complex64Value) }
func (v Complex64Value) xor(b Value) Value { panic("internal error") } //TODOOK

type Complex128Value complex128

func (v Complex128Value) add(b Value) Value { return v + b.(Complex128Value) }
func (v Complex128Value) and(b Value) Value { panic("internal error") } //TODOOK
func (v Complex128Value) div(b Value) Value { return v / b.(Complex128Value) }
func (v Complex128Value) eq(b Value) Value  { return boolValue(v == b.(Complex128Value)) }
func (v Complex128Value) ge(b Value) Value  { panic("internal error") } //TODOOK }
func (v Complex128Value) gt(b Value) Value  { panic("internal error") } //TODOOK }
func (v Complex128Value) isNonZero() bool   { return v != 0 }
func (v Complex128Value) isZero() bool      { return v == 0 }
func (v Complex128Value) le(b Value) Value  { panic("internal error") } //TODOOK }
func (v Complex128Value) lsh(b Value) Value { panic("internal error") } //TODOOK
func (v Complex128Value) lt(b Value) Value  { panic("internal error") } //TODOOK }
func (v Complex128Value) mod(b Value) Value { panic("internal error") } //TODOOK
func (v Complex128Value) mul(b Value) Value { return v * b.(Complex128Value) }
func (v Complex128Value) neg() Value        { return -v }
func (v Complex128Value) neq(b Value) Value { return boolValue(v != b.(Complex128Value)) }
func (v Complex128Value) or(b Value) Value  { panic("internal error") } //TODOOK
func (v Complex128Value) rsh(b Value) Value { panic("internal error") } //TODOOK
func (v Complex128Value) sub(b Value) Value { return v - b.(Complex128Value) }
func (v Complex128Value) xor(b Value) Value { panic("internal error") } //TODOOK

type lvalue struct {
	Operand
	declarator *Declarator
}

func (o *lvalue) Declarator() *Declarator { return o.declarator }
func (o *lvalue) IsLValue() bool          { return true }

func (o *lvalue) convertTo(ctx *context, n Node, to Type) (r Operand) {
	return &lvalue{Operand: o.Operand.convertTo(ctx, n, to), declarator: o.declarator}
}

type funcDesignator struct {
	Operand
	declarator *Declarator
}

func (o *funcDesignator) Declarator() *Declarator { return o.declarator }
func (o *funcDesignator) IsLValue() bool          { return false }

func (o *funcDesignator) convertTo(ctx *context, n Node, to Type) (r Operand) {
	return &lvalue{Operand: o.Operand.convertTo(ctx, n, to), declarator: o.declarator}
}

type operand struct {
	typ   Type
	value Value

	//TODO isLvalue bool or wrapper type
}

func (o *operand) Declarator() *Declarator { return nil }
func (o *operand) IsLValue() bool          { return false }
func (o *operand) IsNonZero() bool         { return o.value != nil && o.value.isNonZero() }
func (o *operand) IsZero() bool            { return o.value != nil && o.value.isZero() }
func (o *operand) Type() Type              { return o.typ }
func (o *operand) Value() Value            { return o.value }

// [0]6.3.1.8
//
// Many operators that expect operands of arithmetic type cause conversions and
// yield result types in a similar way. The purpose is to determine a common
// real type for the operands and result. For the specified operands, each
// operand is converted, without change of type domain, to a type whose
// corresponding real type is the common real type. Unless explicitly stated
// otherwise, the common real type is also the corresponding real type of the
// result, whose type domain is the type domain of the operands if they are the
// same, and complex otherwise. This pattern is called the usual arithmetic
// conversions:
func usualArithmeticConversions(ctx *context, n Node, a, b Operand) (Operand, Operand) {
	if a.Type().Kind() == Invalid || b.Type().Kind() == Invalid {
		return noOperand, noOperand
	}

	if !a.Type().IsArithmeticType() || !b.Type().IsArithmeticType() {
		panic("internal error") //TODOOK
	}

	if a.Type() == nil || b.Type() == nil {
		return a, b
	}

	a = a.normalize(ctx)
	b = b.normalize(ctx)

	at := a.Type()
	bt := b.Type()

	// First, if the corresponding real type of either operand is long
	// double, the other operand is converted, without change of type
	// domain, to a type whose corresponding real type is long double.
	if at.Kind() == ComplexLongDouble || bt.Kind() == ComplexLongDouble {
		return a.convertTo(ctx, n, ctx.cfg.ABI.Type(ComplexLongDouble)), b.convertTo(ctx, n, ctx.cfg.ABI.Type(ComplexLongDouble))
	}

	if at.Kind() == LongDouble || bt.Kind() == LongDouble {
		return a.convertTo(ctx, n, ctx.cfg.ABI.Type(LongDouble)), b.convertTo(ctx, n, ctx.cfg.ABI.Type(LongDouble))
	}

	// Otherwise, if the corresponding real type of either operand is
	// double, the other operand is converted, without change of type
	// domain, to a type whose corresponding real type is double.
	if at.Kind() == ComplexDouble || bt.Kind() == ComplexDouble {
		return a.convertTo(ctx, n, ctx.cfg.ABI.Type(ComplexDouble)), b.convertTo(ctx, n, ctx.cfg.ABI.Type(ComplexDouble))
	}

	if at.Kind() == Double || bt.Kind() == Double {
		return a.convertTo(ctx, n, ctx.cfg.ABI.Type(Double)), b.convertTo(ctx, n, ctx.cfg.ABI.Type(Double))
	}

	// Otherwise, if the corresponding real type of either operand is
	// float, the other operand is converted, without change of type
	// domain, to a type whose corresponding real type is float.
	if at.Kind() == ComplexFloat || bt.Kind() == ComplexFloat {
		return a.convertTo(ctx, n, ctx.cfg.ABI.Type(ComplexFloat)), b.convertTo(ctx, n, ctx.cfg.ABI.Type(ComplexFloat))
	}

	if at.Kind() == Float || bt.Kind() == Float {
		return a.convertTo(ctx, n, ctx.cfg.ABI.Type(Float)), b.convertTo(ctx, n, ctx.cfg.ABI.Type(Float))
	}

	if !a.Type().IsIntegerType() || !b.Type().IsIntegerType() {
		panic("internal error") //TODOOK
	}

	// Otherwise, the integer promotions are performed on both operands.
	a = a.integerPromotion(ctx, n)
	b = b.integerPromotion(ctx, n)
	at = a.Type()
	bt = b.Type()

	// Then the following rules are applied to the promoted operands:

	// If both operands have the same type, then no further conversion is
	// needed.
	if at.Kind() == bt.Kind() {
		return a, b
	}

	// Otherwise, if both operands have signed integer types or both have
	// unsigned integer types, the operand with the type of lesser integer
	// conversion rank is converted to the type of the operand with greater
	// rank.
	abi := ctx.cfg.ABI
	if abi.isSignedInteger(at.Kind()) == abi.isSignedInteger(bt.Kind()) {
		t := a.Type()
		if intConvRank[bt.Kind()] > intConvRank[at.Kind()] {
			t = b.Type()
		}
		return a.convertTo(ctx, n, t), b.convertTo(ctx, n, t)

	}

	// Otherwise, if the operand that has unsigned integer type has rank
	// greater or equal to the rank of the type of the other operand, then
	// the operand with signed integer type is converted to the type of the
	// operand with unsigned integer type.
	switch {
	case a.Type().IsSignedType(): // b is unsigned
		if intConvRank[bt.Kind()] >= intConvRank[a.Type().Kind()] {
			return a.convertTo(ctx, n, b.Type()), b
		}
	case b.Type().IsSignedType(): // a is unsigned
		if intConvRank[at.Kind()] >= intConvRank[b.Type().Kind()] {
			return a, b.convertTo(ctx, n, a.Type())
		}
	default:
		panic(fmt.Errorf("TODO %v %v", a, b))
	}

	// Otherwise, if the type of the operand with signed integer type can
	// represent all of the values of the type of the operand with unsigned
	// integer type, then the operand with unsigned integer type is
	// converted to the type of the operand with signed integer type.
	var signed Type
	switch {
	case abi.isSignedInteger(at.Kind()): // b is unsigned
		signed = a.Type()
		if intConvRank[bt.Kind()] >= intConvRank[at.Kind()] {
			return a.convertTo(ctx, n, b.Type()), b
		}
	case abi.isSignedInteger(bt.Kind()): // a is unsigned
		signed = b.Type()
		if intConvRank[at.Kind()] >= intConvRank[bt.Kind()] {
			return a, b.convertTo(ctx, n, a.Type())
		}

	}

	// Otherwise, both operands are converted to the unsigned integer type
	// corresponding to the type of the operand with signed integer type.
	var typ Type
	switch signed.Kind() {
	case Int:
		//TODO if a.IsEnumConst || b.IsEnumConst {
		//TODO 	return a, b
		//TODO }

		typ = abi.Type(UInt)
	case Long:
		typ = abi.Type(ULong)
	case LongLong:
		typ = abi.Type(ULongLong)
	default:
		panic("internal error") //TODOOK
	}
	return a.convertTo(ctx, n, typ), b.convertTo(ctx, n, typ)
}

// [0]6.3.1.1-2
//
// If an int can represent all values of the original type, the value is
// converted to an int; otherwise, it is converted to an unsigned int. These
// are called the integer promotions. All other types are unchanged by the
// integer promotions.
func (o *operand) integerPromotion(ctx *context, n Node) Operand {
	t := o.Type()
	if t2 := integerPromotion(ctx, t); t2.Kind() != t.Kind() {
		return o.convertTo(ctx, n, t2)
	}

	return o
}

// [0]6.3.1.1-2
//
// If an int can represent all values of the original type, the value is
// converted to an int; otherwise, it is converted to an unsigned int. These
// are called the integer promotions. All other types are unchanged by the
// integer promotions.
func integerPromotion(ctx *context, t Type) Type {
	// github.com/gcc-mirror/gcc/gcc/testsuite/gcc.c-torture/execute/bf-sign-2.c
	//
	// This test checks promotion of bitfields.  Bitfields
	// should be promoted very much like chars and shorts:
	//
	// Bitfields (signed or unsigned) should be promoted to
	// signed int if their value will fit in a signed int,
	// otherwise to an unsigned int if their value will fit
	// in an unsigned int, otherwise we don't promote them
	// (ANSI/ISO does not specify the behavior of bitfields
	// larger than an unsigned int).
	if t.IsBitFieldType() {
		f := t.BitField()
		intBits := int(ctx.cfg.ABI.Types[Int].Size) * 8
		switch {
		case t.IsSignedType():
			if f.BitFieldWidth() < intBits-1 {
				return ctx.cfg.ABI.Type(Int)
			}
		default:
			if f.BitFieldWidth() < intBits {
				return ctx.cfg.ABI.Type(Int)
			}
		}
		return t
	}

	switch t.Kind() {
	case Invalid:
		return t
	case Char, SChar, UChar, Short, UShort:
		return ctx.cfg.ABI.Type(Int)
	default:
		return t
	}
}

func (o *operand) convertTo(ctx *context, n Node, to Type) (r Operand) {
	if o.Type().Kind() == Invalid {
		return o
	}

	v := o.Value()
	if v == nil {
		return &operand{typ: to}
	}

	if o.Type().Kind() == to.Kind() {
		return (&operand{typ: to, value: v}).normalize(ctx)
	}

	if o.Type().IsIntegerType() {
		return o.convertFromInt(ctx, n, to)
	}

	if to.IsIntegerType() {
		return o.convertToInt(ctx, n, to)
	}

	switch o.Type().Kind() {
	case Array:
		switch to.Kind() {
		case Ptr:
			return &operand{typ: to, value: v}
		}
	case ComplexFloat:
		v := v.(Complex64Value)
		switch to.Kind() {
		case ComplexDouble, ComplexLongDouble:
			return (&operand{typ: to, value: Complex128Value(v)}).normalize(ctx)
		case Float:
			return (&operand{typ: to, value: Float32Value(real(v))}).normalize(ctx)
		case Double:
			return (&operand{typ: to, value: Float64Value(real(v))}).normalize(ctx)
		}
	case ComplexDouble:
		v := v.(Complex128Value)
		switch to.Kind() {
		case ComplexFloat:
			return (&operand{typ: to, value: Complex64Value(v)}).normalize(ctx)
		case ComplexLongDouble:
			return (&operand{typ: to, value: v}).normalize(ctx)
		case Float:
			return (&operand{typ: to, value: Float32Value(real(v))}).normalize(ctx)
		case Double:
			return (&operand{typ: to, value: Float64Value(real(v))}).normalize(ctx)
		}
	case Float:
		v := v.(Float32Value)
		switch to.Kind() {
		case ComplexFloat:
			return (&operand{typ: to, value: Complex64Value(complex(v, 0))}).normalize(ctx)
		case ComplexDouble, ComplexLongDouble:
			return (&operand{typ: to, value: Complex128Value(complex(v, 0))}).normalize(ctx)
		case Double:
			return (&operand{typ: to, value: Float64Value(v)}).normalize(ctx)
		}
	case Double:
		v := v.(Float64Value)
		switch to.Kind() {
		case ComplexFloat:
			return (&operand{typ: to, value: Complex64Value(complex(v, 0))}).normalize(ctx)
		case ComplexDouble, ComplexLongDouble:
			return (&operand{typ: to, value: Complex128Value(complex(v, 0))}).normalize(ctx)
		case LongDouble:
			return (&operand{typ: to, value: v}).normalize(ctx)
		case Float:
			return (&operand{typ: to, value: Float32Value(v)}).normalize(ctx)
		}
	}
	panic("TODO")
}

type signedSaturationLimit struct {
	fmin, fmax float64
	min, max   int64
}

type unsignedSaturationLimit struct {
	fmax float64
	max  uint64
}

var (
	signedSaturationLimits = [...]signedSaturationLimit{
		1: {math.Nextafter(math.MinInt8, 0), math.Nextafter(math.MaxInt8, 0), math.MinInt8, math.MaxInt8},
		2: {math.Nextafter(math.MinInt16, 0), math.Nextafter(math.MaxInt16, 0), math.MinInt16, math.MaxInt16},
		4: {math.Nextafter(math.MinInt32, 0), math.Nextafter(math.MaxInt32, 0), math.MinInt32, math.MaxInt32},
		8: {math.Nextafter(math.MinInt64, 0), math.Nextafter(math.MaxInt64, 0), math.MinInt32, math.MaxInt64},
	}

	unsignedSaturationLimits = [...]unsignedSaturationLimit{
		1: {math.Nextafter(math.MaxUint8, 0), math.MaxUint8},
		2: {math.Nextafter(math.MaxUint16, 0), math.MaxUint16},
		4: {math.Nextafter(math.MaxUint32, 0), math.MaxUint32},
		8: {math.Nextafter(math.MaxUint64, 0), math.MaxUint64},
	}
)

func (o *operand) convertToInt(ctx *context, n Node, to Type) (r Operand) {
	v := o.Value()
	switch o.Type().Kind() {
	case Float:
		v := float64(v.(Float32Value))
		switch {
		case to.IsSignedType():
			limits := &signedSaturationLimits[to.Size()]
			if v > limits.fmax {
				return (&operand{typ: to, value: Int64Value(limits.max)}).normalize(ctx)
			}

			if v < limits.fmin {
				return (&operand{typ: to, value: Int64Value(limits.min)}).normalize(ctx)
			}

			return (&operand{typ: to, value: Int64Value(v)}).normalize(ctx)
		default:
			limits := &unsignedSaturationLimits[to.Size()]
			if v > limits.fmax {
				return (&operand{typ: to, value: Uint64Value(limits.max)}).normalize(ctx)
			}

			return (&operand{typ: to, value: Uint64Value(v)}).normalize(ctx)
		}
	case Double, LongDouble:
		v := float64(v.(Float64Value))
		switch {
		case to.IsSignedType():
			limits := &signedSaturationLimits[to.Size()]
			if v > limits.fmax {
				return (&operand{typ: to, value: Int64Value(limits.max)}).normalize(ctx)
			}

			if v < limits.fmin {
				return (&operand{typ: to, value: Int64Value(limits.min)}).normalize(ctx)
			}

			return (&operand{typ: to, value: Int64Value(v)}).normalize(ctx)
		default:
			limits := &unsignedSaturationLimits[to.Size()]
			if v > limits.fmax {
				return (&operand{typ: to, value: Uint64Value(limits.max)}).normalize(ctx)
			}

			return (&operand{typ: to, value: Uint64Value(v)}).normalize(ctx)
		}
	case Ptr:
		var v uint64
		switch x := o.Value().(type) {
		case Int64Value:
			v = uint64(x)
		case Uint64Value:
			v = uint64(x)
		default:
			panic("TODO")
		}
		switch {
		case to.IsSignedType():
			return (&operand{typ: to, value: Int64Value(v)}).normalize(ctx)
		default:
			return (&operand{typ: to, value: Uint64Value(v)}).normalize(ctx)
		}
	case Array:
		return &operand{typ: to}
	}
	panic("TODO")
}

func (o *operand) convertFromInt(ctx *context, n Node, to Type) (r Operand) {
	var v uint64
	switch x := o.Value().(type) {
	case Int64Value:
		v = uint64(x)
	case Uint64Value:
		v = uint64(x)
	default:
		panic("TODO")
	}

	if to.IsIntegerType() {
		switch {
		case to.IsSignedType():
			return (&operand{typ: to, value: Int64Value(v)}).normalize(ctx)
		default:
			return (&operand{typ: to, value: Uint64Value(v)}).normalize(ctx)
		}
	}

	switch to.Kind() {
	case ComplexFloat:
		switch {
		case o.Type().IsSignedType():
			return (&operand{typ: to, value: Complex64Value(complex(float64(int64(v)), 0))}).normalize(ctx)
		default:
			return (&operand{typ: to, value: Complex64Value(complex(float64(v), 0))}).normalize(ctx)
		}
	case ComplexDouble, ComplexLongDouble:
		switch {
		case o.Type().IsSignedType():
			return (&operand{typ: to, value: Complex128Value(complex(float64(int64(v)), 0))}).normalize(ctx)
		default:
			return (&operand{typ: to, value: Complex128Value(complex(float64(v), 0))}).normalize(ctx)
		}
	case Float:
		switch {
		case o.Type().IsSignedType():
			return (&operand{typ: to, value: Float32Value(float64(int64(v)))}).normalize(ctx)
		default:
			return (&operand{typ: to, value: Float32Value(float64(v))}).normalize(ctx)
		}
	case Double, LongDouble:
		switch {
		case o.Type().IsSignedType():
			return (&operand{typ: to, value: Float64Value(int64(v))}).normalize(ctx)
		default:
			return (&operand{typ: to, value: Float64Value(v)}).normalize(ctx)
		}
	case Ptr:
		return (&operand{typ: to, value: Uint64Value(v)}).normalize(ctx)
	case Struct, Union, Void, Int128, UInt128:
		return &operand{typ: to}
	}
	panic("TODO")
}

func (o *operand) normalize(ctx *context) (r Operand) {
	if o.Type().IsIntegerType() {
		switch x := o.Value().(type) {
		case Int64Value:
			if v := convertInt64(int64(x), o.Type(), ctx); v != int64(x) { //TODO ???
				return &operand{o.Type(), Int64Value(v)}
			}
		case Uint64Value:
			v := uint64(x)
			switch o.Type().Size() {
			case 1:
				v &= math.MaxUint8
			case 2:
				v &= math.MaxUint16
			case 4:
				v &= math.MaxUint32
			}
			if v != uint64(x) {
				return &operand{o.Type(), Uint64Value(v)}
			}
		case nil:
			// ok
		default:
			panic(fmt.Errorf("internal error: %T", x)) //TODOOK
		}
		return o
	}

	switch o.Type().Kind() {
	case ComplexFloat:
		switch x := o.Value().(type) {
		case Complex64Value, nil:
			return o
		default:
			panic(fmt.Errorf("internal error: %T", x)) //TODOOK
		}
	case ComplexDouble, ComplexLongDouble:
		switch x := o.Value().(type) {
		case Complex128Value, nil:
			return o
		default:
			panic(fmt.Errorf("internal error: %T", x)) //TODOOK
		}
	case Float:
		switch x := o.Value().(type) {
		case Float32Value, nil:
			return o
		default:
			panic(fmt.Errorf("internal error: %T", x)) //TODOOK
		}
	case Double, LongDouble:
		switch x := o.Value().(type) {
		case Float64Value, nil:
			return o
		default:
			panic(fmt.Errorf("internal error: %T", x)) //TODOOK
		}
	case Ptr:
		switch x := o.Value().(type) {
		case Int64Value, Uint64Value, nil:
			return o
		default:
			panic(fmt.Errorf("internal error: %T", x)) //TODOOK
		}
	}
	panic("TODO")
}

func convertInt64(n int64, t Type, ctx *context) int64 {
	abi := ctx.cfg.ABI
	k := t.Kind()
	if k == Enum {
		//TODO
	}
	signed := abi.isSignedInteger(k)
	switch sz := abi.size(k); sz {
	case 1:
		switch {
		case signed:
			switch {
			case int8(n) < 0:
				return n | ^math.MaxUint8
			default:
				return n & math.MaxUint8
			}
		default:
			return n & math.MaxUint8
		}
	case 2:
		switch {
		case signed:
			switch {
			case int16(n) < 0:
				return n | ^math.MaxUint16
			default:
				return n & math.MaxUint16
			}
		default:
			return n & math.MaxUint16
		}
	case 4:
		switch {
		case signed:
			switch {
			case int32(n) < 0:
				return n | ^math.MaxUint32
			default:
				return n & math.MaxUint32
			}
		default:
			return n & math.MaxUint32
		}
	case 8:
		return n
	default:
		panic("internal error") //TODOOK
	}
}

func boolValue(b bool) Value {
	if b {
		return Int64Value(1)
	}

	return Int64Value(0)
}
