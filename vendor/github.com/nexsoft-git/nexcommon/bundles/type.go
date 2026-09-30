package bundles

type Bundles interface {
	ReadMessageBundle(
		bundleName string,
		messageID string,
		language string,
		param map[string]interface{},
	) (output string)
}
