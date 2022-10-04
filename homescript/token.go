package homescript

type Token struct {
	Kind          TokenKind
	Value         string
	StartLocation Location
	EndLocation   Location
}

type TokenKind uint8

const (
	Unknown TokenKind = iota
	EOF

	Semicolon // ;
	Comma     // ,
	Dot       // .
	Range     // ..
	Arrow     // =>

	LParen // (
	RParen // )
	LCurly // {
	RCurly // }

	Or               // ||
	And              // &&
	Equal            // ==
	NotEqual         // !=
	LessThan         // <
	LessThanEqual    // <=
	GreaterThan      // >
	GreaterThanEqual // >=
	Not              // !

	// TODO: continue here

	Plus     // +
	Minus    // -
	Multiply // *
	Divide   // /
	Reminder // %
	Power    // **

	Assign         // =
	PlusAssign     // +=
	MinusAssign    // -=
	MultiplyAssign // *=
	DivideAssign   // /=
	PowerAssign    // **=
	ReminderAssign // %=

	Fn       // fn
	If       // if
	Else     // else
	Try      // try
	Catch    // catch
	For      // for
	While    // while
	Loop     // loop
	Break    // break
	Continue // continue
	Return   // return

	StringType  // str
	NumberType  // num
	BooleanType // bool
	NullType    // null

	True  // true
	False // false
	On    // on
	Off   // off

	String     // "foo" (token includes quotes whilst content excludes quotes)
	Number     // 42
	Identifier // foobar
)

func unknownToken(location Location) Token {
	return Token{
		Kind:          Unknown,
		Value:         "unknown",
		StartLocation: location,
		EndLocation:   location,
	}
}

func (self TokenKind) String() string {
	var display string
	switch self {
	case Unknown:
		display = "unknown"
	case EOF:
		display = "EOF"
	case Semicolon:
		display = "semicolon"
	case Comma:
		display = "comma"
	case Dot:
		display = "dot"
	case Range:
		display = "range"
	case Arrow:
		display = "arrow"
	case LParen:
		display = "l-paren"
	case RParen:
		display = "r-paren"
	case LCurly:
		display = "l-curly"
	case RCurly:
		display = "r-curly"
	case Or:
		display = "or"
	case And:
		display = "and"
	case Equal:
		display = "equal"
	case NotEqual:
		display = "not-equal"
	case LessThan:
		display = "less-than"
	case LessThanEqual:
		display = "less-than-equal"
	case GreaterThan:
		display = "greater-than"
	case GreaterThanEqual:
		display = "greater-than-equal"
	case Not:
		display = "not"
	case Plus:
		display = "plus"
	case Minus:
		display = "minus"
	case Multiply:
		display = "multiply"
	case Divide:
		display = "divide"
	case Reminder:
		display = "reminder"
	case Power:
		display = "power"
	case Assign:
		display = "assign"
	case PlusAssign:
		display = "plus-assign"
	case MinusAssign:
		display = "minus-assign"
	case MultiplyAssign:
		display = "multiply-assign"
	case DivideAssign:
		display = "divide-assign"
	case PowerAssign:
		display = "power-assign"
	case ReminderAssign:
		display = "reminder-assign"
	case Fn:
		display = "fn"
	case If:
		display = "if"
	case Else:
		display = "else"
	case Try:
		display = "try"
	case Catch:
		display = "catch"
	case For:
		display = "for"
	case While:
		display = "while"
	case Loop:
		display = "loop"
	case Break:
		display = "break"
	case Continue:
		display = "continue"
	case Return:
		display = "return"
	case StringType:
		display = "type:str"
	case NumberType:
		display = "type:num"
	case BooleanType:
		display = "type:bool"
	case NullType:
		display = "type:NULL"
	case True:
		display = "true"
	case False:
		display = "false"
	case On:
		display = "on"
	case Off:
		display = "off"
	case String:
		display = "string"
	case Number:
		display = "number"
	case Identifier:
		display = "identifier"
	}
	return display
}
