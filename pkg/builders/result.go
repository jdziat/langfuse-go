package builders

// BuildResult wraps a result with its validation state.
// This pattern forces callers to handle validation by requiring
// explicit unwrapping of the result.
type BuildResult[T any] struct {
	value T
	err   error
}

// Unwrap returns the value and error, forcing error handling.
// This is the recommended way to use BuildResult.
//
// Example:
//
//	trace, err := builder.Create(ctx).Unwrap()
//	if err != nil {
//	    log.Printf("Failed to create trace: %v", err)
//	    return
//	}
func (r BuildResult[T]) Unwrap() (T, error) {
	return r.value, r.err
}

// Must returns the value or panics if there's an error.
// Use only in tests or when validation is guaranteed.
//
// Example:
//
//	// Only use in tests!
//	trace := builder.Create(ctx).Must()
func (r BuildResult[T]) Must() T {
	if r.err != nil {
		panic(r.err)
	}
	return r.value
}

// Ok returns true if there was no error.
func (r BuildResult[T]) Ok() bool {
	return r.err == nil
}

// Err returns the error, if any.
func (r BuildResult[T]) Err() error {
	return r.err
}

// Value returns the value without checking for errors.
// Prefer Unwrap() for safe access.
func (r BuildResult[T]) Value() T {
	return r.value
}

// NewBuildResult creates a new BuildResult with a value.
func NewBuildResult[T any](value T, err error) BuildResult[T] {
	return BuildResult[T]{value: value, err: err}
}

// BuildResultError creates a BuildResult with only an error.
func BuildResultError[T any](err error) BuildResult[T] {
	var zero T
	return BuildResult[T]{value: zero, err: err}
}

// BuildResultOk creates a BuildResult with only a value.
func BuildResultOk[T any](value T) BuildResult[T] {
	return BuildResult[T]{value: value, err: nil}
}
