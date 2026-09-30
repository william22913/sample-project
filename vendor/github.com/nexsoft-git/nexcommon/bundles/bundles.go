package bundles

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/nexsoft-git/nexlogger/log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

func NewBundles(
	rootDir string,
	defaultLanguage string,
) (
	Bundles,
	error,
) {
	bundles := &bundles{
		defaultLanguage: defaultLanguage,
	}
	err := bundles.loadBundleI18N(rootDir)
	return bundles, err
}

type bundles struct {
	bundles         map[string]*i18n.Bundle
	defaultLanguage string
}

/*
[PARAM]

	@bundleName : nama bundle yang ada di struktur i18n
	bundleName yang dapat digunakan pada contoh dibawah adalah ["common.constanta", "common.error", "common_service"]
	|- common
		|-- constanta
		|-- error
	|- common_service

	@messageID : key yang terdapat pada file i18n
	@language : bahasa yang akan digunakan untuk memperoleh message
	@param : param yang akan diisikan untuk melengkapi message yang digenerate
*/
func (b bundles) ReadMessageBundle(
	bundleName string,
	messageID string,
	language string,
	param map[string]interface{},
) (output string) {

	defer func() {
		if r := recover(); r != nil {
			output = messageID
		}
	}()

	if language == "" {
		language = b.defaultLanguage
	}

	bundle := b.bundles[bundleName]
	localize := i18n.NewLocalizer(bundle, language)

	if param == nil {
		output = localize.MustLocalize(&i18n.LocalizeConfig{
			MessageID: messageID,
		})
	} else {
		output = localize.MustLocalize(&i18n.LocalizeConfig{
			MessageID:    messageID,
			TemplateData: param,
		})
	}

	return
}

func (b *bundles) loadBundleI18N(dirName string) error {
	b.bundles = make(map[string]*i18n.Bundle)
	root := fmt.Sprintf("./%s/", dirName)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.Name() == dirName {
			return nil
		}

		if info.IsDir() {
			path = strings.ReplaceAll(path, "\\", "/")
			pathSplit := strings.Split(path, "/")
			name := strings.Join(pathSplit[1:], ".")
			filePath := fmt.Sprintf("./%s", path)

			dir, err := ioutil.ReadDir(filePath)
			if err != nil {
				log.Fatal().
					Err(err).
					Caller().
					Msg("Error When reading i18n bundle.")
			}

			bundle := i18n.NewBundle(language.Indonesian)
			bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

			found := false
			for _, f := range dir {
				if !f.IsDir() {
					found = true

					i18nFilePath := fmt.Sprintf("%s/%s", filePath, f.Name())
					_, err := bundle.LoadMessageFile(i18nFilePath)

					if err != nil {
						return err
					}
				}
			}

			if found {
				b.bundles[name] = bundle
			}
		}
		return err
	})

	return err
}
