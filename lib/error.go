package lib

type errorWithContext struct {
	original error
	message  string
}

func (e errorWithContext) Error() string {
	return e.message
}
