package in

import "mime/multipart"

type MultipartFile struct {
	File     multipart.File
	Header   *multipart.FileHeader
	FullPath string
	Alias    string
}

type MultipleMultipartFile []MultipartFile
