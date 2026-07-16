package inifiles

// parse parses data into c. Stubbed until Phase 1 implementation.
func (c *Config) parse(data []byte) error {
	if err := validateCommentChars(c.opts.CommentChar, c.opts.AllowedCommentChars); err != nil {
		return err
	}
	_ = data
	return errNotImplemented
}