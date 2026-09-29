package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/nexsoft-git/nexcommon/constanta"
	srvCtx "github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/model"
	"github.com/nexsoft-git/nexlogger/log"

	"github.com/nexsoft-git/nexcommon/dto/out"
	"github.com/nexsoft-git/nexcommon/server/metrics"
)

func NewAPIConnector(
	metrics metrics.Metrics,
) APIConnector {
	return &apiConnector{
		metrics:      metrics,
		logEveryHit:  true,
		outgoingPath: "/v1/audit/outgoing",
	}
}

type apiConnector struct {
	metrics              metrics.Metrics
	logEveryHit          bool
	resourceID           string
	outgoingPath         string
	externalActivityPath string
	fixedToken           string
}

func (a *apiConnector) EnableExternalActivity(
	resourceID string,
	externalActivityHost string,
	externalActivityPort int,
	fixedToken string,
) {
	a.resourceID = resourceID
	a.externalActivityPath = fmt.Sprintf("http://%s:%d", externalActivityHost, externalActivityPort)
	if externalActivityPort == 0 {
		a.externalActivityPath = fmt.Sprintf("https://%s", externalActivityHost)
	}

	a.fixedToken = fixedToken
}

func (a *apiConnector) DisableApiConnectorLog() {
	a.logEveryHit = false
}

type HTTPClientDTO struct {
	Header        http.Header
	Unsuccessfull out.DefaultErrorResponse
}

func (h *HTTPClientDTO) SetHeader(header http.Header) {
	h.Header = header
}

func (h *HTTPClientDTO) GetHeader() http.Header {
	return h.Header
}

func (h *HTTPClientDTO) Unsuccessfully() interface{} {
	return &h.Unsuccessfull
}

type AbsHTTPClientDTO interface {
	GetHeader() http.Header
	SetHeader(http.Header)
	Unsuccessfully() interface{}
}

type hitAPILog struct {
	FullPath string            `json:"path"`
	Method   string            `json:"method"`
	Header   map[string]string `json:"header"`
	Error    error             `json:"err"`
	Status   int               `json:"status"`
}

func (a apiConnector) HitAPI(
	ctx context.Context,
	method string,
	host string,
	path string,
	header map[string]string,
	body interface{},
	result AbsHTTPClientDTO,
) (
	int,
	error,
) {
	var status int
	var err error
	now := time.Now()

	defer func() {
		a.metrics.GetDefaultMetric().APIConnectorHist.WithLabelValues(
			host,
			path,
			method,
			strconv.Itoa(status),
		).Observe(float64(time.Since(now).Seconds()))

		if a.logEveryHit {

			metaData := hitAPILog{
				FullPath: fmt.Sprintf("%s%s", host, path),
				Method:   method,
				Header:   header,
				Error:    err,
				Status:   status,
			}

			data, _ := json.Marshal(metaData)
			ctxMdl, _ := ctx.Value(constanta.ApplicationContextConstanta).(*srvCtx.ContextModel)
			log := log.Info().
				Str("meta_data", string(data))
			if ctxMdl != nil {
				log = log.LoggerModel(ctxMdl.ClientAccess.Logger)
			}
			log.Msg("Do Hit API Another Service")
		}
	}()

	url := fmt.Sprintf("%s%s", host, path)
	bodyByte, _ := json.Marshal(body)

	req, err := http.NewRequest(method, url, bytes.NewBuffer(bodyByte))
	if err != nil {
		return 0, err
	}

	req.Close = true

	a.checkContext(ctx, header)
	a.addHeader(req, header)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}

	result.SetHeader(resp.Header)
	status = resp.StatusCode

	defer resp.Body.Close()

	bodyResponse, _ := ioutil.ReadAll(resp.Body)

	if len(bodyResponse) > 0 {
		if status == 200 {
			err = json.Unmarshal(bodyResponse, result)
		} else {
			err = json.Unmarshal(bodyResponse, result.Unsuccessfully())
		}

		if err != nil {
			return status, err
		}

	}

	return resp.StatusCode, nil
}

func (a apiConnector) HitAPIMultipart(
	ctx context.Context,
	method string,
	host string,
	path string,
	header map[string]string,
	body model.MultipartRequest,
	result AbsHTTPClientDTO,
) (
	int,
	error,
) {
	var status int
	var err error
	now := time.Now()

	defer func() {
		a.metrics.GetDefaultMetric().APIConnectorHist.WithLabelValues(
			host,
			path,
			method,
			strconv.Itoa(status),
		).Observe(float64(time.Since(now).Seconds()))

		if a.logEveryHit {
			ctxMdl, _ := ctx.Value(constanta.ApplicationContextConstanta).(*srvCtx.ContextModel)

			log.Info().
				Interface("meta_data", hitAPILog{
					FullPath: fmt.Sprintf("%s%s", host, path),
					Method:   method,
					Header:   header,
					Error:    err,
					Status:   status,
				}).
				LoggerModel(ctxMdl.ClientAccess.Logger).
				Msg("Do Hit API Another Service")
		}
	}()

	buffer := &bytes.Buffer{}
	writer := multipart.NewWriter(buffer)

	for keys := range body {
		switch body[keys].(type) {
		case model.FileURLPath:
			data := body[keys].(model.FileURLPath)

			part, err := writer.CreateFormFile(keys, filepath.Base(data.FullPath))
			if err != nil {
				return 0, err
			}

			file, err := os.OpenFile(data.FullPath, os.O_RDONLY, os.ModeAppend)

			if err != nil {
				return 0, err
			}

			defer func() {
				_ = file.Close()
				if err == nil {
					if !data.KeepFile {
						_ = os.Remove(data.FullPath)
					}
				}
			}()

			_, err = io.Copy(part, file)

		default:
			if reflect.ValueOf(body[keys]).Kind() == reflect.Struct {
				bodyByte, _ := json.Marshal(body[keys])
				_ = writer.WriteField(keys, string(bodyByte))
			} else {
				_ = writer.WriteField(keys, fmt.Sprintf("%v", body[keys]))
			}
		}
	}

	_ = writer.Close()

	url := fmt.Sprintf("%s%s", host, path)
	req, err := http.NewRequest(method, url, buffer)

	if err != nil {
		return 0, err
	}

	req.Close = true

	header["Content-Type"] = writer.FormDataContentType()

	a.checkContext(ctx, header)
	a.addHeader(req, header)

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return 0, err
	}

	result.SetHeader(resp.Header)
	status = resp.StatusCode

	defer resp.Body.Close()

	bodyResponse, _ := ioutil.ReadAll(resp.Body)

	if len(bodyResponse) > 0 {
		if status == 200 {
			err = json.Unmarshal(bodyResponse, result)
		} else {
			err = json.Unmarshal(bodyResponse, result.Unsuccessfully())
		}

		if err != nil {
			return 0, err
		}

	}

	return resp.StatusCode, nil
}

func (a apiConnector) addHeader(req *http.Request, header map[string]string) {
	for keys := range header {
		req.Header.Add(keys, header[keys])
	}
}

func (a apiConnector) checkContext(ctx context.Context, header map[string]string) {
	if header == nil {
		header = make(map[string]string)
	}

	requestID := ctx.Value(constanta.RequestIDConstanta)
	if header[constanta.RequestIDConstanta] == "" {
		header[constanta.RequestIDConstanta], _ = requestID.(string)
	}

	ipAddress := ctx.Value(constanta.IPAddressConstanta)
	if header[constanta.IPAddressConstanta] == "" {
		header[constanta.IPAddressConstanta], _ = ipAddress.(string)
	}

	source := ctx.Value(constanta.SourceConstanta)
	if header[constanta.SourceConstanta] == "" {
		header[constanta.SourceConstanta], _ = source.(string)
	}
}

func (a apiConnector) ParseURL(
	rawURL string,
	urlParam interface{},
) (
	parsedURL string,
) {
	if rawURL == "" || urlParam == nil {
		return rawURL
	}

	v := reflect.ValueOf(urlParam)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return rawURL
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return rawURL
	}

	items := collectFields(v)

	outURL := rawURL
	for _, it := range items {
		if it.kind != "path" {
			continue
		}
		ph := "{" + it.name + "}"
		// Use PathEscape on each, but since path placeholder is single-valued by convention,
		// use the first if present.
		val := ""
		if len(it.vals) > 0 {
			val = url.PathEscape(it.vals[0])
		}
		outURL = strings.ReplaceAll(outURL, ph, val)
	}

	// Merge query
	u, err := url.Parse(outURL)
	if err != nil {
		// Fallback: append manually
		q := buildValues(items)
		qstr := q.Encode()
		if qstr == "" {
			return outURL
		}
		sep := "?"
		if strings.Contains(outURL, "?") {
			if strings.HasSuffix(outURL, "?") || strings.HasSuffix(outURL, "&") {
				sep = ""
			} else {
				sep = "&"
			}
		}
		return outURL + sep + qstr
	}

	existing := u.Query()
	additional := buildValues(items)
	for k, vs := range additional {
		for _, v := range vs {
			existing.Add(k, v)
		}
	}
	u.RawQuery = existing.Encode()

	u.Path = cleanPath(u.Path)

	return u.String()
}

func (a apiConnector) ParseHeader(
	headerParam interface{},
) map[string]string {
	if headerParam == nil {
		return map[string]string{}
	}

	v := reflect.ValueOf(headerParam)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return map[string]string{}
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return map[string]string{}
	}

	out := map[string]string{}
	collectHeaders(v, out)
	return out
}
func (a apiConnector) ParseMultipartStruct(
	multipartParam interface{},
) model.MultipartRequest {
	out := make(model.MultipartRequest)

	if multipartParam == nil {
		return out
	}

	v := reflect.ValueOf(multipartParam)
	// Deref pointers to get to the struct
	for v.IsValid() && v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return out
		}
		v = v.Elem()
	}
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return out
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		// Ignore unexported fields
		if sf.PkgPath != "" {
			continue
		}

		jsonKey := jsonTagName(sf.Tag.Get("json"))
		if jsonKey == "" || jsonKey == "-" {
			continue
		}

		fv := v.Field(i)

		// If the field is model.FileURLPath (or *model.FileURLPath) -> set value directly
		if isFileURLPath(sf.Type) {
			out[jsonKey] = derefInterface(fv)
			continue
		}

		// If the field is a struct or interface -> marshal to JSON string
		kind := baseKind(sf.Type)
		switch kind {
		case reflect.Struct, reflect.Interface:
			if isNilValue(fv) {
				out[jsonKey] = "" // keep the key with empty string for nil interface/pointer struct
				continue
			}
			// Use the interface value to preserve proper JSON marshaling
			val := derefInterface(fv)
			if b, err := json.Marshal(val); err == nil {
				out[jsonKey] = string(b)
			} else {
				// Fallback to %v if marshal fails
				out[jsonKey] = fmt.Sprintf("%v", val)
			}
		default:
			// Otherwise -> plain string using %v (with pointer deref)
			out[jsonKey] = fmt.Sprintf("%v", derefInterface(fv))
		}
	}

	return out
}

func jsonTagName(tag string) string {
	if tag == "" {
		return ""
	}
	parts := strings.Split(tag, ",")
	name := strings.TrimSpace(parts[0])
	return name
}

func baseKind(t reflect.Type) reflect.Kind {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Kind()
}

func isFileURLPath(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return t.PkgPath() == "yourpkg/model" && t.Name() == "FileURLPath"
}

func derefInterface(v reflect.Value) interface{} {

	for v.IsValid() && v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return nil
	}
	if v.CanInterface() {
		return v.Interface()
	}

	return fmt.Sprintf("%v", v)
}

func isNilValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Interface, reflect.Func:
		return v.IsNil()
	default:
		return false
	}
}

func collectHeaders(v reflect.Value, out map[string]string) {
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		fv := v.Field(i)

		// Recurse into nested structs / pointers (skip time.Time and other std lib named structs)
		switch fv.Kind() {
		case reflect.Ptr:
			if !fv.IsNil() {
				elem := fv.Elem()
				if elem.IsValid() && elem.Kind() == reflect.Struct && elem.Type().PkgPath() != "time" {
					collectHeaders(elem, out)
				}
			}
		case reflect.Struct:
			if fv.Type().PkgPath() != "time" {
				collectHeaders(fv, out)
			}
		}

		tag := sf.Tag.Get("header")
		if tag == "" {
			continue
		}

		name := tag
		val, ok := headerValueToString(fv)
		if !ok {
			continue
		}
		// Per RFC, multiple values can be comma-separated in a single header line
		// If already present, append.
		if existing, exists := out[name]; exists && existing != "" && val != "" {
			out[name] = existing + ", " + val
		} else {
			out[name] = val
		}
	}
}

func headerValueToString(v reflect.Value) (string, bool) {
	// Deref pointers
	for v.IsValid() && v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "", false
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return "", false
	}

	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		var parts []string
		for i := 0; i < v.Len(); i++ {
			s := fmt.Sprint(valueIface(v.Index(i)))
			if strings.TrimSpace(s) != "" {
				parts = append(parts, s)
			}
		}
		if len(parts) == 0 {
			return "", false
		}
		return strings.Join(parts, ", "), true
	default:
		s := fmt.Sprint(valueIface(v))
		if strings.TrimSpace(s) == "" {
			return "", false
		}
		return s, true
	}
}

func valueIface(v reflect.Value) interface{} {
	for v.IsValid() && v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	if v.CanInterface() {
		return v.Interface()
	}

	return fmt.Sprint(v)
}

type fieldItem struct {
	kind string // "path" or "query"
	name string
	vals []string
}

func collectFields(v reflect.Value) []fieldItem {
	var items []fieldItem

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		fv := v.Field(i)

		switch fv.Kind() {
		case reflect.Ptr:
			if !fv.IsNil() {
				elem := fv.Elem()
				if elem.IsValid() && elem.Kind() == reflect.Struct && elem.Type().PkgPath() != "time" {
					items = append(items, collectFields(elem)...)
				}
			}
		case reflect.Struct:
			if fv.Type().PkgPath() != "time" {
				items = append(items, collectFields(fv)...)
			}
		}

		kind := sf.Tag.Get("field") // "path" | "query"
		if kind == "" {
			continue
		}
		name := sf.Tag.Get("name")
		if name == "" {
			name = sf.Name
		}

		vals := valueToStrings(fv)
		items = append(items, fieldItem{
			kind: strings.ToLower(kind),
			name: name,
			vals: vals,
		})
	}

	return items
}

func valueToStrings(v reflect.Value) []string {

	for v.IsValid() && v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return nil
	}

	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		var out []string
		for i := 0; i < v.Len(); i++ {
			out = append(out, fmt.Sprint(valueInterface(v.Index(i))))
		}
		return out
	default:
		return []string{fmt.Sprint(valueInterface(v))}
	}
}

func valueInterface(v reflect.Value) interface{} {

	for v.IsValid() && v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.CanInterface() {
		return v.Interface()
	}
	return fmt.Sprint(v)
}

func buildValues(items []fieldItem) url.Values {
	q := url.Values{}
	for _, it := range items {
		if it.kind != "query" || len(it.vals) == 0 {
			continue
		}
		for _, val := range it.vals {
			q.Add(it.name, val)
		}
	}
	return q
}

func cleanPath(p string) string {
	hasTrailing := strings.HasSuffix(p, "/") && p != "/"
	clean := path.Clean(p)
	if hasTrailing && !strings.HasSuffix(clean, "/") {
		clean += "/"
	}
	return clean
}
