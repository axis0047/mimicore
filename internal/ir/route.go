package ir

type Route struct {
	Method     string
	Path       PathTemplate
	Validators []Validator
	Response   []ResponseRule
}

type PathTemplate struct {
	Segments   []string
	ParamNames []string
}
