package server

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/nexsoft-git/nexcommon/dto/in"
	"github.com/nexsoft-git/nexcommon/util/text"
	"github.com/nexsoft-git/nexlogger/log"
)

type MultipartReader interface {
	ParseMultipartForm(
		r *http.Request,
	) error

	ReadMultipartFile(
		request *http.Request,
		key string,
	) (
		file []in.MultipartFile,
		errs error,
	)

	SetFileNameAliasingFunction(
		f filenameAliasing,
	) *multipartReader

	StartRemoveExpiredFileJob(
		routine time.Duration,
		expired time.Duration,
	)

	StopRemovingExpiredFileJob()
}

type filenameAliasing func(int, string) string

func SplitFileNameAndExtension(filename string) (string, string) {

	split := strings.Split(filename, ".")
	if len(split) == 1 {
		return filename, ""
	}

	return strings.Join(split[0:len(split)-2], ""), split[len(split)-1]
}

func DefaultAliasingFunction(idx int, fileName string) string {
	var addition string
	if idx > 0 {
		addition = fmt.Sprintf("_%d", idx)
	}

	withoutExt, ext := SplitFileNameAndExtension(fileName)
	return fmt.Sprintf("%s%s.%s", withoutExt, addition, ext)
}

func UUIDAliasingFunction(idx int, fileName string) string {
	var addition string
	if idx > 0 {
		addition = fmt.Sprintf("_%d", idx)
	}

	_, ext := SplitFileNameAndExtension(fileName)

	return fmt.Sprintf("%s%s.%s", text.GetUUID(), addition, ext)
}

func NewMultipartReader(
	maxMemory int64,
	tempFile string,
) MultipartReader {

	_ = os.MkdirAll(tempFile, 0770)

	return &multipartReader{
		maxMemory:        maxMemory,
		tempFile:         tempFile,
		filenameAliasing: DefaultAliasingFunction,
	}
}

type multipartReader struct {
	maxMemory        int64
	tempFile         string
	filenameAliasing filenameAliasing
	state            chan (struct{})
}

func (m *multipartReader) StopRemovingExpiredFileJob() {
	if m.state != nil {
		m.state <- struct{}{}
	}
}

func (m *multipartReader) StartRemoveExpiredFileJob(
	routine time.Duration,
	expired time.Duration,
) {
	ticker := time.NewTicker(routine)
	m.state = make(chan (struct{}))

	for {
		select {
		case <-ticker.C:

			counter := 0
			log.Info().
				Str("path", m.tempFile).
				Caller().
				Msg("Job Remove Expired File Started")

			var err error
			listFile, err := os.ReadDir(m.tempFile)
			if err != nil {
				log.Error().
					Err(err).
					Caller().
					Msg("Error Found when get file from directory")
			}

			for i := 0; i < len(listFile); i++ {
				if !listFile[i].IsDir() {
					info, err := listFile[i].Info()
					if err != nil {
						log.Error().
							Err(err).
							Caller().
							Msg("Error Found when reading file info")
						continue
					}

					if time.Now().Unix() > info.ModTime().Add(expired).Unix() {
						counter++
						err = os.Remove(fmt.Sprintf("%s/%s", m.tempFile, listFile[i].Name()))
						if err != nil {
							log.Error().
								Err(err).
								Caller().
								Msg("Error Found when deleting file")
							continue
						}
					}
				}
			}

			log.Info().
				Str("path", m.tempFile).
				Int("counter", counter).
				Caller().
				Msg("Job Remove Expired File Ended")

		case <-m.state:
			ticker.Stop()
			return
		}
	}

}

var multipartByReader = &multipart.Form{
	Value: make(map[string][]string),
	File:  make(map[string][]*multipart.FileHeader),
}

func (m *multipartReader) SetFileNameAliasingFunction(f filenameAliasing) *multipartReader {
	m.filenameAliasing = f
	return m
}

func (m multipartReader) ParseMultipartForm(
	r *http.Request,
) error {

	if r.MultipartForm == multipartByReader {
		return errors.New("http: multipart handled by MultipartReader")
	}

	if r.Form == nil {
		err := r.ParseForm()
		if err != nil {
			return err
		}
	}

	if r.MultipartForm != nil {
		return nil
	}

	mr, err := m.readMultipart(r, false)
	if err != nil {
		return err
	}

	f, err := m.readForm(mr)
	if err != nil {
		return err
	}

	if r.PostForm == nil {
		r.PostForm = make(url.Values)
	}

	for k, v := range f.Value {
		r.Form[k] = append(r.Form[k], v...)
		// r.PostForm should also be populated. See Issue 9305.
		r.PostForm[k] = append(r.PostForm[k], v...)
	}

	r.MultipartForm = f

	return nil
}

func (m multipartReader) readMultipart(
	r *http.Request,
	allowMixed bool,
) (
	*multipart.Reader,
	error,
) {

	v := r.Header.Get("Content-Type")
	if v == "" {
		return nil, http.ErrNotMultipart
	}

	d, params, err := mime.ParseMediaType(v)
	if err != nil || !(d == "multipart/form-data" || allowMixed && d == "multipart/mixed") {
		return nil, http.ErrNotMultipart
	}

	boundary, ok := params["boundary"]
	if !ok {
		return nil, http.ErrMissingBoundary
	}

	return multipart.NewReader(r.Body, boundary), nil
}

func (m multipartReader) readForm(
	r *multipart.Reader,
) (
	form *multipart.Form,
	err error,
) {
	maxMemory := m.maxMemory
	tempFile := m.tempFile

	form = &multipart.Form{
		Value: make(map[string][]string),
		File:  make(map[string][]*multipart.FileHeader),
	}

	defer func() {
		if err != nil {
			if form != nil {
				_ = form.RemoveAll()
			}
		}
	}()

	maxValueBytes := maxMemory + int64(10<<20)

	var first = true

	for {
		p, err := r.NextPart()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		name := p.FormName()
		if name == "" {
			continue
		}

		filename := p.FileName()

		var b bytes.Buffer

		if filename == "" {
			n, err := io.CopyN(&b, p, maxValueBytes+1)
			if err != nil && err != io.EOF {
				return nil, err
			}

			maxValueBytes -= n
			if maxValueBytes < 0 {
				return nil, multipart.ErrMessageTooLarge
			}

			form.Value[name] = append(form.Value[name], b.String())
			continue
		}

		var idx = 0

		//check form.File[name] is nil (prevent panic)
		if form.File[name] != nil {
			idx = len(form.File[name])
		}

		fileAliasing := m.filenameAliasing(idx, p.FileName())
		p.Header.Add("alias_name", fileAliasing)

		fh := &multipart.FileHeader{
			Filename: filename,
			Header:   p.Header,
		}

		err = func() error {
			var fileTemp *os.File

			defer func() {
				if fileTemp != nil {
					_ = fileTemp.Close()
				}
			}()

			fullPath := fmt.Sprintf("%s/%s", tempFile, fileAliasing)

			if first {
				_ = os.Remove(fullPath)
				first = false
			}

			fileTemp, err = os.OpenFile(fullPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0660)
			if err != nil {
				fileTemp, err = ioutil.TempFile("", "multipart-")
				if err != nil {
					return err
				}
				return err
			}

			size, err := io.Copy(fileTemp, io.MultiReader(&b, p))
			if cerr := fileTemp.Close(); err == nil {
				err = cerr
			}

			if err != nil {
				_ = os.Remove(fileTemp.Name())
				return err
			}

			fh.Size = size

			return nil
		}()

		if err != nil {
			return nil, err
		}

		form.File[name] = append(form.File[name], fh)
	}

	return form, nil
}

func (m multipartReader) ReadMultipartFile(
	request *http.Request,
	key string,
) (
	result []in.MultipartFile,
	err error,
) {

	files, ok := request.MultipartForm.File[key]
	if !ok {
		return
	}

	defer func() {
		if result != nil {
			for i := 0; i < len(result); i++ {
				errs := result[i].File.Close()
				if errs != nil {
					log.Error().
						Err(err).
						Caller().
						Msg("Error Found When Close multipart file")

					if err == nil {
						err = errs
					}
				}
			}
		}
	}()

	for i := 0; i < len(files); i++ {
		file := files[i]
		if file.Header != nil {
			tempResult := in.MultipartFile{
				Header: file,
			}

			tempResult.Alias = tempResult.Header.Header.Get("alias_name")
			tempResult.FullPath = m.tempFile + "/" + tempResult.Alias
			tempResult.File, err = os.OpenFile(tempResult.FullPath, os.O_RDONLY, os.ModeAppend)
			if err != nil {
				return
			}

			result = append(result, tempResult)
		}
	}

	return result, nil
}

func (m multipartReader) getFileTemp(
	tempFile string,
	b bytes.Buffer,
	p *multipart.Part,
	fh *multipart.FileHeader,
) (
	err error,
) {

	var fileTemp *os.File
	defer func() {
		if fileTemp != nil {
			_ = fileTemp.Close()
		}
	}()

	_, _ = os.Create(tempFile)

	fileTemp, err = os.OpenFile(tempFile, os.O_CREATE|os.O_APPEND|os.O_RDWR, os.ModeAppend)
	if err != nil {
		fileTemp, err = ioutil.TempFile("", "multipart-")
		if err != nil {
			return err
		}
		return err
	}

	size, err := io.Copy(fileTemp, io.MultiReader(&b, p))
	if cerr := fileTemp.Close(); err == nil {
		err = cerr
	}

	if err != nil {
		_ = os.Remove(fileTemp.Name())
		return err
	}

	fh.Filename = fileTemp.Name()
	fh.Size = size

	return nil
}
