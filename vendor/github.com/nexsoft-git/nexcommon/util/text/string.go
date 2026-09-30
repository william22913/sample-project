package text

func EncodeReport(
	encoded []byte,
) string {
	var rst []byte
	for i := 0; i < len(encoded); i++ {
		if encoded[i] != '\xc2' {
			rst = append(rst, encoded[i])
		}
	}

	return string(rst)
}
