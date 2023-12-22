package ast

type AnalyzedSingleton struct {
	Type                Type
	ImplementsTemplates []Template
	// Methods are unlike methods in languages like Rust or Java.
	// Here, a method is just a function that is implemented in a template block.
	Methods []AnalyzedFunctionDefinition
}

func NewSingleton(typ Type, implementsTemplates []Template, methods []AnalyzedFunctionDefinition) AnalyzedSingleton {
	return AnalyzedSingleton{
		Type:                typ,
		ImplementsTemplates: implementsTemplates,
		Methods:             methods,
	}
}

// A template is comparable to a trait in Rust.
// It describes which methods (and their signatures) need to be implemented on a singleton.
type Template struct {
}
