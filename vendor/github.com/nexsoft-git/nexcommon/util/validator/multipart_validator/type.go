package multipart_validator

type MultipartValidator interface {
	ValidateMultipart(
		param *validateMultipartParam,
	) (
		interface{},
		error,
	)
}
