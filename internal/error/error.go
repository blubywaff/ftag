package _error

type ErrorWithContext struct {
	Original error
	Message  string
}

func (e ErrorWithContext) Error() string {
	return e.Message + ": " + e.Original.Error()
}

func (e ErrorWithContext) Unwrap() error {
	return e.Original
}

// Represents a result that can be cancelled or undone
type IntermediateResult struct {
	Cleanup func() error
	Err     error
}

func IntermediateResultFromError(err error) IntermediateResult {
	return IntermediateResult{func() error { return nil }, err}
}

// return nil if cleanup was successful
// this includes cases where there was nothing to clenaup because the original operation failed
func (ir *IntermediateResult) Clean() error {
	if ir.Err != nil {
		return nil
	}
	return ir.Cleanup()
}

func (ir *IntermediateResult) OpError() error {
	return ir.Err
}
