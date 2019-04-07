%{
// Copyright 2019 The CC Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on [0], 6.5-6.10. 

package cc // import "modernc.org/cc/v3"

%}

%union {
	Token			Token
	node			Node
}

%token
	/*yy:token "%c"		*/	IDENTIFIER
	/*yy:token "%c_e"	*/	ENUMCONST
	/*yy:token "%c_t"	*/	TYPEDEFNAME
	/*yy:token "%d"		*/	INTCONST
	/*yy:token "'%c'"	*/	CHARCONST
	/*yy:token "1.%d"	*/	FLOATCONST
	/*yy:token "L'%c'"	*/	LONGCHARCONST
	/*yy:token "L\"%c\""	*/	LONGSTRINGLITERAL
	/*yy:token "\"%c\""	*/	STRINGLITERAL

	ACCUM			"_Accum"
	ADDASSIGN		"+="
	ALIGNAS			"_Alignas"
	ALIGNOF			"_Alignof"
	ANDAND			"&&"
	ANDASSIGN		"&="
	ARROW			"->"
	ASM			"asm"
	ATOMIC			"_Atomic"
	ATTRIBUTE		"__attribute__"
	AUTO			"auto"
	BOOL			"_Bool"
	BREAK			"break"
	BUILTINTYPESCOMPATIBLE	"__builtin_types_compatible_p"
	CASE			"case"
	CHAR			"char"
	COMPLEX			"_Complex"
	CONST			"const"
	CONTINUE		"continue"
	DDD			"..."
	DEC			"--"
	DECIMAL128		"_Decimal128"
	DECIMAL32		"_Decimal32"
	DECIMAL64		"_Decimal64"
	DEFAULT			"default"
	DIVASSIGN		"/="
	DO			"do"
	DOUBLE			"double"
	ELSE			"else"
	ENUM			"enum"
	EQ			"=="
	EXTERN			"extern"
	FLOAT			"float"
	FLOAT128		"_Float128"
	FLOAT16			"__fp16"
	FLOAT80			"__float80"
	FOR			"for"
	FRACT			"_Fract"
	GEQ			">="
	GOTO			"goto"
	IF			"if"
	IMAG			"__imag__"
	INC			"++"
	INLINE			"inline"
	INT			"int"
	INT128			"__int128"
	LABEL			"__label__"
	LEQ			"<="
	LONG			"long"
	LSH			"<<"
	LSHASSIGN		"<<="
	MODASSIGN		"%="
	MULASSIGN		"*="
	NEQ			"!="
	NORETURN		"_Noreturn"
	ORASSIGN		"|="
	OROR			"||"
	PPNUMBER		"preprocessing number"
	PPPASTE			"##"
	REAL			"__real__"
	REGISTER		"register"
	RESTRICT		"restrict"
	RETURN			"return"
	RSH			">>"
	RSHASSIGN		">>="
	SAT			"_Sat"
	SHORT			"short"
	SIGNED			"signed"
	SIZEOF			"sizeof"
	STATIC			"static"
	STRUCT			"struct"
	SUBASSIGN		"-="
	SWITCH			"switch"
	THREADLOCAL		"_Thread_local"
	TYPEDEF			"typedef"
	TYPEOF			"typeof"
	UNION			"union"
	UNSIGNED		"unsigned"
	VOID			"void"
	VOLATILE		"volatile"
	WHILE			"while"
	XORASSIGN		"^="

%precedence	BELOW_ELSE
%precedence	ELSE

%precedence	BELOW_ATTRIBUTE
%precedence	ATTRIBUTE

%start TranslationUnit

%%

		        /* [0], 6.5.1 Primary expressions */
			/*yy:field	lexicalScope	Scope	*/
			/*yy:field	resolvedIn	Scope	*/
			/*yy:example int i = x; */
/*yy:case Ident      */
			PrimaryExpression:
				IDENTIFIER
			/*yy:example int i = 42; */
/*yy:case Int        */ |	INTCONST
			/*yy:example int i = 3.14; */
/*yy:case Float      */ |	FLOATCONST
			/*yy:example enum e {a}; int i = a; */
/*yy:case Enum       */ |	ENUMCONST
			/*yy:example int i = 'x'; */
/*yy:case Char       */ |	CHARCONST
			/*yy:example int i = L'x'; */
/*yy:case LChar      */ |	LONGCHARCONST
			/*yy:example char *c = "x"; */
/*yy:case String     */ |	STRINGLITERAL
			/*yy:example char *c = L"x"; */
/*yy:case LString    */ |	LONGSTRINGLITERAL
			/*yy:example int i = (x+y); */
/*yy:case Expr       */ |	'(' Expression ')'
			/*yy:example int i = ({x();}); */
/*yy:case Stmt       */	|	'(' CompoundStatement ')'

		        /* [0], 6.5.2 Postfix operators */
			/*yy:example int i = x; */
/*yy:case Primary    */ PostfixExpression:
				PrimaryExpression
			/*yy:example int i = x[y]; */
/*yy:case Index      */ |	PostfixExpression '[' Expression ']'
			/*yy:example int i = x(y); */
/*yy:case Call       */ |	PostfixExpression '(' ArgumentExpressionList ')'
			/*yy:example int i = x.y; */
/*yy:case Select     */ |	PostfixExpression '.' IDENTIFIER
			/*yy:example int i = x->y; */
/*yy:case PSelect    */ |	PostfixExpression "->" IDENTIFIER
			/*yy:example int i = x++; */
/*yy:case Inc        */ |	PostfixExpression "++"
			/*yy:example int i = x--; */
/*yy:case Dec        */ |	PostfixExpression "--"
			/*yy:example int i = (int[]){y}; */
/*yy:case Complit    */ |	'(' TypeName ')' '{' InitializerList ',' '}'
			/*yy:example int i = __builtin_types_compatible_p(int, double); */
/*yy:case TypeCmp   */  |	"__builtin_types_compatible_p" '(' TypeName ',' TypeName ')'

			/*yy:example int i = f(x); */
			ArgumentExpressionList:
				AssignmentExpression
			/*yy:example int i = f(x, y); */
			|	ArgumentExpressionList ',' AssignmentExpression

			/* [0], 6.5.3 Unary operators */
			/*yy:field	lexicalScope	Scope	*/
			/*yy:example int i = x; */
/*yy:case Postfix    */ UnaryExpression:
				PostfixExpression
			/*yy:example int i = ++x; */
/*yy:case Inc        */ |	"++" UnaryExpression
			/*yy:example int i = --x; */
/*yy:case Dec        */ |	"--" UnaryExpression
			/*yy:example int *i = &x; */
/*yy:case Addrof     */ |	'&' CastExpression
			/*yy:example int i = *x; */
/*yy:case Deref      */ |	'*' CastExpression
			/*yy:example int i = +x; */
/*yy:case Plus       */ |	'+' CastExpression
			/*yy:example int i = -x; */
/*yy:case Minus      */ |	'-' CastExpression
			/*yy:example int i = ~x; */
/*yy:case Cpl        */ |	'~' CastExpression
			/*yy:example int i = !x; */
/*yy:case Not        */ |	'!' CastExpression
			/*yy:example int i = sizeof x; */
/*yy:case SizeofExpr */ |	"sizeof" UnaryExpression
			/*yy:example int i = sizeof(int); */
/*yy:case SizeofType */ |	"sizeof" '(' TypeName ')'
			/*yy:example int f() { L: &&L; }*/
/*yy:case LabelAddr  */ |	"&&" IDENTIFIER
			/*yy:example int i = _Alignof(x); */
/*yy:case AlignofExpr*/ |	"_Alignof" UnaryExpression
			/*yy:example int i = _Alignof(int); */
/*yy:case AlignofType*/ |	"_Alignof" '(' TypeName ')'
			/*yy:example double i = __imag__ x; */
/*yy:case Imag       */ |	"__imag__" UnaryExpression
			/*yy:example double i = __real__ x; */
/*yy:case Real       */ |	"__real__" UnaryExpression

			/* [0], 6.5.4 Cast operators */
			/*yy:example int i = 42; */
/*yy:case Unary      */ CastExpression:
				UnaryExpression
			/*yy:example int i = (int)3.14; */
/*yy:case Cast       */ |	'(' TypeName ')' CastExpression

			/* [0], 6.5.5 Multiplicative operators */
			/*yy:example int i = x;*/
/*yy:case Cast       */ MultiplicativeExpression:
				CastExpression
			/*yy:example int i = x * y;*/
/*yy:case Mul        */ |	MultiplicativeExpression '*' CastExpression
			/*yy:example int i = x / y;*/
/*yy:case Div        */ |	MultiplicativeExpression '/' CastExpression
			/*yy:example int i = x % y;*/
/*yy:case Mod        */ |	MultiplicativeExpression '%' CastExpression

			/* [0], 6.5.6 Additive operators */
			/*yy:example int i = x; */
/*yy:case Mul        */ AdditiveExpression:
				MultiplicativeExpression
			/*yy:example int i = x+y; */
/*yy:case Add        */ |	AdditiveExpression '+' MultiplicativeExpression
			/*yy:example int i = x-y; */
/*yy:case Sub        */ |	AdditiveExpression '-' MultiplicativeExpression

			/* [0], 6.5.7 Bitwise shift operators */
			/*yy:example int i = x; */
/*yy:case Add        */ ShiftExpression:
				AdditiveExpression
			/*yy:example int i = x << y; */
/*yy:case Lsh        */ |	ShiftExpression "<<" AdditiveExpression
			/*yy:example int i = x >> y; */
/*yy:case Rsh        */ |	ShiftExpression ">>" AdditiveExpression

			/* [0], 6.5.8 Relational operators */
			/*yy:example int i = x; */
/*yy:case Shift      */ RelationalExpression:
				ShiftExpression        
			/*yy:example int i = x < y; */
/*yy:case Lt         */ |	RelationalExpression '<'  ShiftExpression
			/*yy:example int i = x > y; */
/*yy:case Gt         */ |	RelationalExpression '>'  ShiftExpression
			/*yy:example int i = x <= y; */
/*yy:case Leq        */ |	RelationalExpression "<=" ShiftExpression
			/*yy:example int i = x >= y; */
/*yy:case Geq        */ |	RelationalExpression ">=" ShiftExpression

			/* [0], 6.5.9 Equality operators */
			/*yy:example int i = x; */
/*yy:case Rel        */ EqualityExpression:
				RelationalExpression
			/*yy:example int i = x == y; */
/*yy:case Eq         */ |	EqualityExpression "==" RelationalExpression
			/*yy:example int i = x != y; */
/*yy:case Neq        */ |	EqualityExpression "!=" RelationalExpression

			/* [0], 6.5.10 Bitwise AND operator */
			/*yy:example int i = x; */
/*yy:case Eq         */ AndExpression:
				EqualityExpression
			/*yy:example int i = x & y; */
/*yy:case And        */ |	AndExpression '&' EqualityExpression

			/* [0], 6.5.11 Bitwise exclusive OR operator */
			/*yy:example int i = x; */
/*yy:case And        */ ExclusiveOrExpression:
				AndExpression
			/*yy:example int i = x^y; */
/*yy:case Xor        */ |	ExclusiveOrExpression '^' AndExpression

			/* [0], 6.5.12 Bitwise inclusive OR operator */
			/*yy:example int i = x; */
/*yy:case Xor        */ InclusiveOrExpression:
				ExclusiveOrExpression
			/*yy:example int i = x|y; */
/*yy:case Or         */ |	InclusiveOrExpression '|' ExclusiveOrExpression

			/* [0], 6.5.13 Logical AND operator */
			/*yy:example int i = x;*/
/*yy:case Or         */ LogicalAndExpression:
				InclusiveOrExpression
			/*yy:example int i = x && y;*/
/*yy:case LAnd       */ |	LogicalAndExpression "&&" InclusiveOrExpression

			/* [0], 6.5.14 Logical OR operator */
			/*yy:example int i = x;*/
/*yy:case LAnd       */ LogicalOrExpression:
				LogicalAndExpression
			/*yy:example int i = x || y;*/
/*yy:case LOr        */ |	LogicalOrExpression "||" LogicalAndExpression

			/* [0], 6.5.15 Conditional operator */
			/*yy:example int i = x; */
/*yy:case LOr        */ ConditionalExpression:
				LogicalOrExpression
			/*yy:example int i = x ? y : z; */
/*yy:case Cond       */ |	LogicalOrExpression '?' Expression ':' ConditionalExpression

			/* [0], 6.5.16 Assignment operators */
			/*yy:example int i = x; } */
/*yy:case Cond       */ AssignmentExpression:
				ConditionalExpression
			/*yy:example int f() { x = y; } */
/*yy:case Assign     */ |	UnaryExpression '=' AssignmentExpression
			/*yy:example int f() { x *= y; } */
/*yy:case Mul        */ |	UnaryExpression "*=" AssignmentExpression
			/*yy:example int f() { x /= y; } */
/*yy:case Div        */ |	UnaryExpression "/=" AssignmentExpression
			/*yy:example int f() { x %= y; } */
/*yy:case Mod        */ |	UnaryExpression "%=" AssignmentExpression
			/*yy:example int f() { x += y; } */
/*yy:case Add        */ |	UnaryExpression "+=" AssignmentExpression
			/*yy:example int f() { x -= y; } */
/*yy:case Sub        */ |	UnaryExpression "-=" AssignmentExpression
			/*yy:example int f() { x <<= y; } */
/*yy:case Lsh        */ |	UnaryExpression "<<=" AssignmentExpression
			/*yy:example int f() { x >>= y; } */
/*yy:case Rsh        */ |	UnaryExpression ">>=" AssignmentExpression
			/*yy:example int f() { x &= y; } */
/*yy:case And        */ |	UnaryExpression "&=" AssignmentExpression
			/*yy:example int f() { x ^= y; } */
/*yy:case Xor        */ |	UnaryExpression "^=" AssignmentExpression
			/*yy:example int f() { x |= y; } */
/*yy:case Or         */ |	UnaryExpression "|=" AssignmentExpression

			/* [0], 6.5.17 Comma operator */
			/*yy:example int f() { i = x; }; */
/*yy:case Assign     */ Expression:
				AssignmentExpression
			/*yy:example int f() { x, y; }; */
/*yy:case Comma      */ |	Expression ',' AssignmentExpression

			/* [0], 6.6 Constant expressions */
			/*yy:example struct { int i:3; }; */
			ConstantExpression:
				ConditionalExpression

			/* [0], 6.7 Declarations */
			/*yy:example int i, j; */
			Declaration:
				DeclarationSpecifiers InitDeclaratorList ';'

			/*yy:example static int i; */
/*yy:case Storage    */ DeclarationSpecifiers:
				StorageClassSpecifier DeclarationSpecifiers
			/*yy:example int i; */
/*yy:case TypeSpec   */ |	TypeSpecifier DeclarationSpecifiers
			/*yy:example volatile int i; */
/*yy:case TypeQual   */ |	TypeQualifier DeclarationSpecifiers
			/*yy:example inline int f() {} */
/*yy:case Func       */ |	FunctionSpecifier DeclarationSpecifiers
			/*yy:example _Alignas(double) int i; */
/*yy:case AlignSpec  */ |	AlignmentSpecifier DeclarationSpecifiers
			/*yy:example int __attribute__((a)) i; */
/*yy:case Attribute  */ |	AttributeSpecifier DeclarationSpecifiers

			/*yy:example int i; */
			InitDeclaratorList:
				InitDeclarator
			/*yy:example int i, j; */
			|	InitDeclaratorList ',' AttributeSpecifierList InitDeclarator

			/*yy:example int i; */
/*yy:case Decl       */ InitDeclarator:
				Declarator AttributeSpecifierList
			/*yy:example int i = x; */
/*yy:case Init       */ |	Declarator AttributeSpecifierList '=' Initializer

			/* [0], 6.7.1 Storage-class specifiers */
			/*yy:example typedef int int_t;*/
/*yy:case Typedef    */ StorageClassSpecifier:
				"typedef"
			/*yy:example extern int i;*/
/*yy:case Extern     */ |	"extern"
			/*yy:example static int i;*/
/*yy:case Static     */ |	"static"
			/*yy:example auto int i;*/
/*yy:case Auto       */ |	"auto"
			/*yy:example register int i;*/
/*yy:case Register   */ |	"register"
			/*yy:example _Thread_local int i;*/
/*yy:case ThreadLocal*/ |	"_Thread_local"

			/* [0], 6.7.2 Type specifiers */
			/*yy:field	resolvedIn	Scope	// Case TypedefName */
			/*yy:example void i(); */
/*yy:case Void       */ TypeSpecifier:
				"void"
			/*yy:example char i; */
/*yy:case Char       */ |	"char"
			/*yy:example short i; */
/*yy:case Short      */ |	"short"
			/*yy:example int i; */
/*yy:case Int        */ |	"int"
			/*yy:example __int128 i; */
/*yy:case Int128     */ |	"__int128"
			/*yy:example long i; */
/*yy:case Long       */ |	"long"
			/*yy:example float i; */
/*yy:case Float      */ |	"float"
			/*yy:example __fp16 i; */
/*yy:case Float16    */ |	"__fp16"
			/*yy:example _Decimal32 i; */
/*yy:case Decimal32  */ |	"_Decimal32"
			/*yy:example _Decimal64 i; */
/*yy:case Decimal64  */ |	"_Decimal64"
			/*yy:example _Decimal128 i; */
/*yy:case Decimal128 */ |	"_Decimal128"
			/*yy:example _Float128 i; */
/*yy:case Float128   */ |	"_Float128"
			/*yy:example __float80 i; */
/*yy:case Float80    */ |	"__float80"
			/*yy:example double i; */
/*yy:case Double     */ |	"double"
			/*yy:example signed i; */
/*yy:case Signed     */ |	"signed"
			/*yy:example unsigned i; */
/*yy:case Unsigned   */ |	"unsigned"
			/*yy:example _Bool i; */
/*yy:case Bool       */ |	"_Bool"
			/*yy:example _Complex i; */
/*yy:case Complex    */ |	"_Complex"
			/*yy:example struct s i; */
/*yy:case Struct     */ |	StructOrUnionSpecifier
			/*yy:example enum e i; */
/*yy:case Enum       */ |	EnumSpecifier
			/*yy:example typedef const T; T i; */
/*yy:case TypedefName*/ |	TYPEDEFNAME
			/*yy:example typeof(42) i; */
/*yy:case TypeofExpr */ |	"typeof" '(' Expression ')'
			/*yy:example typedef const T; typeof(T) i; */
/*yy:case TypeofType */ |	"typeof" '(' TypeName ')'
			/*yy:example _Atomic(int) i; */
/*yy:case Atomic     */ |	AtomicTypeSpecifier
			/*yy:example _Fract i; */
/*yy:case Fract      */ |	"_Fract"
			/*yy:example _Sat i; */
/*yy:case Sat        */ |	"_Sat"
			/*yy:example _Accum i; */
/*yy:case Accum      */ |	"_Accum"

			/* [0], 6.7.2.1 Structure and union specifiers */
			/*yy:field	lexicalScope	Scope	*/
			/*yy:example struct s { int i; }; */
/*yy:case Def        */ StructOrUnionSpecifier:
				StructOrUnion AttributeSpecifierList IDENTIFIER '{' StructDeclarationList '}'
			/*yy:example struct s v; */
/*yy:case Tag        */ |	StructOrUnion AttributeSpecifierList IDENTIFIER

			/*yy:example struct { int i; } s; */
/*yy:case Struct     */ StructOrUnion:
				"struct"
			/*yy:example union { int i; double d; } u; */
/*yy:case Union      */ |	"union"
		
			/*yy:example struct{ int i; } */
			StructDeclarationList:
				StructDeclaration
			/*yy:example struct{ int i; double d; } */
			|	StructDeclarationList StructDeclaration
		
			/*yy:example struct{ int i; } */
			StructDeclaration:
				SpecifierQualifierList StructDeclaratorList ';'
		
			/*yy:example struct {int i;};*/
/*yy:case TypeSpec   */ SpecifierQualifierList:
				TypeSpecifier SpecifierQualifierList
			/*yy:example struct {const int i;};*/
/*yy:case TypeQual   */ |	TypeQualifier SpecifierQualifierList
			/*yy:example struct {_Alignas(double) int i;};*/
/*yy:case AlignSpec  */ |	AlignmentSpecifier SpecifierQualifierList
			/*yy:example struct {__attribute__((a)) int i;};*/
/*yy:case Attribute  */ |	AttributeSpecifier SpecifierQualifierList

			/*yy:example struct{ int i; } */
			StructDeclaratorList:
				StructDeclarator
			/*yy:example struct{ int i, j; } */
			|	StructDeclaratorList ',' StructDeclarator
		
			/*yy:example struct{ int i; } */
/*yy:case Decl       */ StructDeclarator:
				Declarator
			/*yy:example struct{ int i:3; } */
/*yy:case BitField   */ |	Declarator ':' ConstantExpression AttributeSpecifierList

			/* [0], 6.7.2.2 Enumeration specifiers */
			/*yy:field	lexicalScope	Scope	*/
			/*yy:example enum e {a}; */
/*yy:case Def        */ EnumSpecifier:
				"enum" AttributeSpecifierList IDENTIFIER '{' EnumeratorList ',' '}'
			/*yy:example enum e i; */
/*yy:case Tag        */ |	"enum" AttributeSpecifierList IDENTIFIER

			/*yy:example enum e {a}; */
			EnumeratorList:
				Enumerator
			/*yy:example enum e {a, b}; */
			|	EnumeratorList ',' Enumerator

			/*yy:example enum e {a}; */
			/*yy:field	lexicalScope	Scope	*/
/*yy:case Ident      */ Enumerator:
				IDENTIFIER AttributeSpecifierList
			/*yy:example enum e {a = 42}; */
/*yy:case Expr       */ |	IDENTIFIER AttributeSpecifierList '=' ConstantExpression

			/* [2], 6.7.2.4 Atomic type specifiers */
			/*yy:example _Atomic(int) i; */
			AtomicTypeSpecifier:
				"_Atomic" '(' TypeName ')'

			/* [0], 6.7.3 Type qualifiers */
			/*yy:example const int i; */
/*yy:case Const      */ TypeQualifier:
				"const"
			/*yy:example restrict int i; */
/*yy:case Restrict   */ |	"restrict"
			/*yy:example volatile int i; */
/*yy:case Volatile   */ |	"volatile"
			/*yy:example _Atomic int i; */
/*yy:case Atomic     */ |	"_Atomic"

			/* [0], 6.7.4 Function specifiers */
			/*yy:example inline int f() {}*/
/*yy:case Inline     */ FunctionSpecifier:
				"inline"
			/*yy:example _Noreturn int f() {}*/
/*yy:case Noreturn   */ |	"_Noreturn"

			/* [0], 6.7.5 Declarators */
			/*yy:field	IsTypedefName	bool */
			/*yy:example int *p; */
			Declarator:
				Pointer DirectDeclarator AttributeSpecifierList %prec BELOW_ATTRIBUTE

			/* [2], 6.7.5 Alignment specifier */
			/*yy:example _Alignas(double) char c; */
/*yy:case AlignasType*/ AlignmentSpecifier:
				"_Alignas" '(' TypeName ')'
			/*yy:example _Alignas(0ll) char c; */
/*yy:case AlignasExpr*/ |	"_Alignas" '(' ConstantExpression ')'

			/*yy:field	lexicalScope	Scope	*/
			/*yy:field	paramScope	Scope */
			/*yy:example int i; */
/*yy:case Ident      */ DirectDeclarator:
				IDENTIFIER Asm
			/*yy:example int (f); */
/*yy:case Decl       */ |	'(' AttributeSpecifierList Declarator ')'
			/*yy:example int i[const 42]; */
/*yy:case Arr        */ |	DirectDeclarator '[' TypeQualifiers AssignmentExpression ']'
			/*yy:example int i[static const 42]; */
/*yy:case StaticArr  */ |	DirectDeclarator '[' "static" TypeQualifiers AssignmentExpression ']'
			/*yy:example int i[const static 42]; */
/*yy:case ArrStatic  */ |	DirectDeclarator '[' TypeQualifiers "static" AssignmentExpression ']'
			/*yy:example int i[const *]; */
/*yy:case Star       */ |	DirectDeclarator '[' TypeQualifiers '*' ']'
			/*yy:example int f(int i); */
/*yy:case FuncParam  */ |	DirectDeclarator '(' ParameterTypeList ')'
			/*yy:example int f(a); */
/*yy:case FuncIdent  */ |	DirectDeclarator '(' IdentifierList ')'

			/*yy:example int *p; */
/*yy:case TypeQual   */ Pointer:
				'*' TypeQualifiers
			/*yy:example int **p; */
/*yy:case Ptr        */ |	'*' TypeQualifiers Pointer

			/*yy:example int * const i; */
/*yy:case TypeQual   */ TypeQualifiers:
				TypeQualifier
			/*yy:example int * __attribute__((a)) i; */
/*yy:case Attribute  */ |	AttributeSpecifier
			/*yy:example int * const volatile i; */
			|	TypeQualifiers TypeQualifier
			/*yy:example int * __attribute__((a)) __attribute__((b)) i; */
			|	TypeQualifiers AttributeSpecifier

			/*yy:example int f(int i) {} */
/*yy:case List       */ ParameterTypeList:
				ParameterList
			/*yy:example int f(int i, ...) {} */
/*yy:case Var        */ |	ParameterList ',' "..."

			/*yy:example int f(int i) {} */
			ParameterList:
				ParameterDeclaration
			/*yy:example int f(int i, int j) {} */
			|	ParameterList ',' ParameterDeclaration

			/*yy:example int f(int i) {} */
/*yy:case Decl       */ ParameterDeclaration:
				DeclarationSpecifiers Declarator AttributeSpecifierList
			/*yy:example int f(int*) {} */
/*yy:case Abstract   */ |	DeclarationSpecifiers AbstractDeclarator

			/*yy:field	lexicalScope	Scope	*/
			/*yy:example int f(i) int i; {}*/
			IdentifierList:
				IDENTIFIER
			/*yy:example int f(i, j) int i, j; {}*/
			|	IdentifierList ',' IDENTIFIER

			/* [0], 6.7.6 Type names */
			/*yy:example int i = (int)x; */
			TypeName:
				SpecifierQualifierList AbstractDeclarator

			/*yy:example void f(int*); */
/*yy:case Ptr        */ AbstractDeclarator:
				Pointer
			/*yy:example void f(int()); */
/*yy:case Decl       */ |	Pointer DirectAbstractDeclarator

			/*yy:field	paramScope	Scope */
			/*yy:example void f(int()); */
/*yy:case Decl       */ DirectAbstractDeclarator:
				'(' AbstractDeclarator ')'
			/*yy:example void f(int[const 42]); */
/*yy:case Arr        */ |	DirectAbstractDeclarator '[' TypeQualifiers AssignmentExpression ']'
			/*yy:example void f(int[static const 42]); */
/*yy:case StaticArr  */ |	DirectAbstractDeclarator '[' "static" TypeQualifiers AssignmentExpression ']'
			/*yy:example void f(int[const static 42]); */
/*yy:case ArrStatic  */ |	DirectAbstractDeclarator '[' TypeQualifiers "static" AssignmentExpression ']'
			/*yy:example void f(int[*]); */
/*yy:case ArrStar    */ |	DirectAbstractDeclarator '[' '*' ']'
			/*yy:example void f(int(int)); */
/*yy:case Func       */ |	DirectAbstractDeclarator '(' ParameterTypeList ')'

			/* [0], 6.7.8 Initialization */
			/*yy:example int i = x; */
/*yy:case Expr       */ Initializer:
				AssignmentExpression
			/*yy:example int i[] = { x }; */
/*yy:case InitList   */ |	'{' InitializerList ',' '}'

			/*yy:example int i[] = { [10] = x }; */
			InitializerList:
				Designation Initializer
			/*yy:example int i[] = { [10] = x, [20] = y }; */
			|	InitializerList ',' Designation Initializer

			/*yy:example int a[] = { [42] = 314 }; */
			Designation:
				DesignatorList '='

			/*yy:example int a[] = { [42] = 314 }; */
			DesignatorList:
				Designator
			/*yy:example int a[100][] = { [42][12] = 314 }; */
			|	DesignatorList Designator

			/*yy:field	lexicalScope	Scope	*/
			/*yy:example int a[] = { [42] = 314 }; */
/*yy:case Index      */ Designator:
				'[' ConstantExpression ']'
			/*yy:example struct t s = { .fld = 314 }; */
/*yy:case Field      */ |	'.' IDENTIFIER
			/*yy:example struct t s = { fld: 314 }; */
/*yy:case Field2     */ |	IDENTIFIER ':'

			/* [0], 6.8 Statements and blocks */
			/*yy:example int f() { L: x(); }*/
/*yy:case Labeled    */ Statement:
				LabeledStatement
			/*yy:example int f() { { y(); } }*/
/*yy:case Compound   */ |	CompoundStatement
			/*yy:example int f() { x(); }*/
/*yy:case Expr       */ |	ExpressionStatement
			/*yy:example int f() { if(x) y(); }*/
/*yy:case Selection  */ |	SelectionStatement
			/*yy:example int f() { for(;;) x(); }*/
/*yy:case Iteration  */ |	IterationStatement
			/*yy:example int f() { return x; }*/
/*yy:case Jump       */ |	JumpStatement
			/*yy:example int f() { asm("nop"); }*/
/*yy:case Asm        */ |	AsmStatement

			/* [0], 6.8.1 Labeled statements */
			/*yy:field	lexicalScope	Scope	*/
			/*yy:example int f() { L: goto L; } */
/*yy:case Label      */ LabeledStatement:
				IDENTIFIER ':' AttributeSpecifierList Statement
			/*yy:example int f() { switch(i) case 42: x(); } */
/*yy:case CaseLabel  */ |	"case" ConstantExpression ':' Statement
			/*yy:example int f() { switch(i) case 42 ... 56: x(); } */
/*yy:case Range      */ |	"case" ConstantExpression "..." ConstantExpression ':' Statement
			/*yy:example int f() { switch(i) default: x(); } */
/*yy:case Default    */ |	"default" ':' Statement

			/* [0], 6.8.2 Compound statement */
			/*yy:field	scope Scope	*/
			/*yy:example int f() { int i; } */
			CompoundStatement:
				'{' BlockItemList '}'

			/*yy:example int f() { int i; }*/
			BlockItemList:
				BlockItem
			/*yy:example int f() { int i; double j; }*/
			|	BlockItemList BlockItem

			/*yy:example int f() { int i; }*/
/*yy:case Decl       */ BlockItem:
				Declaration
			/*yy:example int f() { g(); }*/
/*yy:case Stmt       */ |	Statement
			/*yy:example int f() { __label__ L; }*/
/*yy:case Label      */ |	LabelDeclaration
			/*yy:example int f() { int g() {} }*/
/*yy:case FuncDef    */ |	DeclarationSpecifiers Declarator CompoundStatement

			/* [0], 6.8.3 Expression and null statements */
			/*yy:example int f() { g(); }*/
			ExpressionStatement:
				Expression AttributeSpecifierList ';'

			/* [0], 6.8.4 Selection statements */
			/*yy:example int f() { if(x) y(); } */
/*yy:case If         */ SelectionStatement:
				"if" '(' Expression ')' Statement %prec BELOW_ELSE
			/*yy:example int f() { if(x) y(); else z(); } */
/*yy:case IfElse     */ |	"if" '(' Expression ')' Statement "else" Statement
			/*yy:example int f() { switch(i) case 42: x(); } */
/*yy:case Switch     */ |	"switch" '(' Expression ')' Statement

			/* [0], 6.8.5 Iteration statements */
			/*yy:example int f() { while(x) y(); } */
/*yy:case While      */ IterationStatement:
				"while" '(' Expression ')' Statement
			/*yy:example int f() { do x(); while(y); } */
/*yy:case Do         */ |	"do" Statement "while" '(' Expression ')' ';'
			/*yy:example int f() { for( i = 0; i < 10; i++) x(); } */
/*yy:case For        */ |	"for" '(' Expression ';' Expression ';' Expression ')' Statement
			/*yy:example int f() { for( int i = 0; i < 10; i++) x(); } */
/*yy:case ForDecl    */ |	"for" '(' Declaration Expression ';' Expression ')' Statement

			/* [0], 6.8.6 Jump statements */
			/*yy:field	lexicalScope	Scope	*/
			/*yy:example int f() { L: goto L; } */
/*yy:case Goto       */ JumpStatement:
				"goto" IDENTIFIER ';'
			/*yy:example int f() { L: x(); void *p = &&L; goto *p; } */
/*yy:case GotoExpr   */ |	"goto" '*' Expression ';'
			/*yy:example int f() { for(;;) if (i) continue; } */
/*yy:case Continue   */ |	"continue" ';'
			/*yy:example int f() { for(;;) if (i) break; } */
/*yy:case Break      */ |	"break" ';'
			/*yy:example int f() { if (i) return x; } */
/*yy:case Return     */ |	"return" Expression ';'

			/* [0], 6.9 External definitions */
			/*yy:list*/
			/*yy:example int i; */
			TranslationUnit:
				ExternalDeclaration
			/*yy:example int i; int j; */
			|	TranslationUnit ExternalDeclaration

			/*yy:example int f() {} */
/*yy:case FuncDef    */ ExternalDeclaration:
				FunctionDefinition
			/*yy:example int i; */
/*yy:case Decl       */ |	Declaration
			/*yy:example int f() asm("nop"); */
/*yy:case Asm        */ |	AsmFunctionDefinition
			/*yy:example asm("nop"); */
/*yy:case AsmStmt    */ |	AsmStatement
			/*yy:example ; */
/*yy:case Empty      */ |	';'

			/* [0], 6.9.1 Function definitions */
			/*yy:example int f() {} */
			FunctionDefinition:
				DeclarationSpecifiers Declarator DeclarationList CompoundStatement

			/*yy:example int f(i) int i; {} */
			DeclarationList:
				Declaration
			/*yy:example int f(i, j) int i; int j; {} */
			|	DeclarationList Declaration

			/* -------------------------------------- Extensions */

			/*yy:example asm("nop": [a] b); */
			AsmIndex:
				'[' Expression ']'

			/*yy:example asm("nop": a); */
			AsmExpressionList:
				AsmIndex AssignmentExpression
			/*yy:example asm("nop": a, b); */
			|	AsmExpressionList ',' AsmIndex AssignmentExpression

			/*yy:example asm("nop": a); */
			AsmArgList:
				':' AsmExpressionList
			/*yy:example asm("nop": a : b); */
			|	AsmArgList ':' AsmExpressionList

			/*yy:example asm("nop"); */
			Asm:
				"asm" AsmQualifierList '(' STRINGLITERAL AsmArgList ')'
 
			/*yy:example void f() { asm("nop"); } */
			AsmStatement:
				Asm AttributeSpecifierList ';'

			/*yy:example int f() asm("nop"); */
			AsmFunctionDefinition:
				DeclarationSpecifiers Declarator AsmStatement

			/*yy:example asm volatile ("nop"); */
/*yy:case Volatile   */ AsmQualifier:
				"volatile"
			/*yy:example asm inline ("nop"); */
/*yy:case Inline     */ |	"inline"
			/*yy:example asm goto ("nop"); */
/*yy:case Goto       */ |	"goto"

			/*yy:example asm inline ("nop"); */
			AsmQualifierList:
				AsmQualifier
			/*yy:example asm inline volatile ("nop"); */
			|	AsmQualifierList AsmQualifier

			/*yy:example int f() { __label__ L; L: x(); } */
			LabelDeclaration:
				"__label__" IdentifierList ';'

			/* [4], 6.37 Attribute Syntax */
			/*yy:example int i __attribute__((a(b))); */
			ExpressionList:
				AssignmentExpression
			/*yy:example int i __attribute__((a(b, c))); */
			|	ExpressionList ',' AssignmentExpression

			/*yy:field	lexicalScope	Scope	*/
			/*yy:example int i __attribute__((a)); */
/*yy:case Ident      */ AttributeValue:
				IDENTIFIER
			/*yy:example int i __attribute__((a(b))); */
/*yy:case Expr       */ |	IDENTIFIER '(' ExpressionList ')'

			/*yy:example int i __attribute__((a)); */
			AttributeValueList:
				AttributeValue
			/*yy:example int i __attribute__((a, b)); */
			|	AttributeValueList ',' AttributeValue

			/*yy:example int i __attribute__((a)); */
			AttributeSpecifier:
				"__attribute__" '(' '(' AttributeValueList ')' ')'

			/*yy:example int i __attribute__((a)); */
			AttributeSpecifierList:
				AttributeSpecifier %prec BELOW_ATTRIBUTE
			/*yy:example int i __attribute__((a)) __attribute((b)); */
			|	AttributeSpecifierList AttributeSpecifier