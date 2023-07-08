package lib

type errorWithContext struct {
	original error
	message  string
}

func (e errorWithContext) Error() string {
	return e.message + ": " + e.original.Error()
}

func (e errorWithContext) Unwrap() error {
	return e.original
}

// Represents a result that can be cancelled or undone
type IntermediateResult struct {
	cleanup func() error
	err     error
}

// return nil if cleanup was successful
// this includes cases where there was nothing to clenaup because the original operation failed
func (ir *IntermediateResult) Clean() error {
	if ir.err != nil {
		return nil
	}
	return ir.cleanup()
}

func (ir *IntermediateResult) OpError() error {
	return ir.err
}
