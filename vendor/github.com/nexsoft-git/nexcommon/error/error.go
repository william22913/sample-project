package error

func NewUnBundledErrorMessages(
	status int,
	code error,
	f Converter,
) *UnbundledErrorMessages {
	return &UnbundledErrorMessages{
		status:   status,
		code:     code,
		function: f,
		param: []interface{}{
			"UNDEFINED",
			"UNDEFINED",
			"UNDEFINED",
			"UNDEFINED",
			"UNDEFINED",
		},
	}
}
