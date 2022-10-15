package interpreter

type Result struct {
	ShouldContinue bool
	ReturnValue    *Value
	BreakValue     *Value
	Value          *Value
}
