package functions

// Text returns a multi-line string.
func Text() string {
	// A comment-only line.

	/*
		A block comment.
	*/
	s := `first
second

fourth`
	return s // A trailing comment.
}
