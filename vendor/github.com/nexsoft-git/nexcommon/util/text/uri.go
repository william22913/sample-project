package text

import (
	"net/url"
	"sort"
	"strings"
)

func LexicographicallySortURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	queryParams := u.Query()

	var keys []string
	for key := range queryParams {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	sortedURL := *u
	sortedURL.RawQuery = ""
	q := url.Values{}
	for _, key := range keys {
		values := queryParams[key]
		sort.Strings(values)
		for _, value := range values {
			q.Add(key, value)
		}
	}
	sortedURL.RawQuery = q.Encode()

	return sortedURL.String(), nil
}

func RemoveQueryParams(
	originalURL string,
) string {
	url := strings.Split(originalURL, "?")
	return url[0]
}

func IsURLWhitelisted(
	url string,
	whitelisted []string,
) bool {
	urlWithoutQuery := RemoveQueryParams(url)
	urlWithoutQuery = strings.Trim(urlWithoutQuery, "/")

	paths := strings.Split(urlWithoutQuery, "/")
	for i := 0; i < len(paths); i++ {
		found := false
		for j := 0; j < len(whitelisted); j++ {
			whitelistedPath := strings.Split(strings.Trim(whitelisted[j], "/"), "/")
			if whitelistedPath[i] == paths[i] {
				found = true
				break
			} else if whitelistedPath[i] == "*" {
				return true
			}
		}

		if i == len(paths)-1 {
			return found
		}

		if !found {
			return false
		}

	}

	return false
}
