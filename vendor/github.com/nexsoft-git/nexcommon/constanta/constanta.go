package constanta

import (
	"database/sql"
	"reflect"
	"time"
)

const DefaultTimeFormat = "2006-01-02T15:04:05Z"
const DefaultDtoOutTimeFormat = "2006-01-02T15:04:05.999999999Z"
const DateOnlyTimeFormat = "2006-01-02"
const DefaultDBTimeFormat = "2006-01-02T15:04:05"

const ReadDataAPIMustHave = "read"
const WriteDataAPIMustHave = "write"
const UpdateDataPermissionMustHave = ":update"
const DeleteDataPermissionMustHave = ":delete"
const DuplicateDataPermissionMustHave = ":duplicate"
const SyncDataPermissionMustHave = ":synchronize"
const InsertDataPermissionMustHave = ":insert"
const ViewDataPermissionMustHave = ":view"
const ChangePasswordPermissionMustHave = ":changepassword"
const ExportDataPermissionMustHave = ":export"
const UploadDataPermissionMustHave = ":upload"
const DownloadDataPermissionMustHave = ":download"
const RestoreDataPermissionMustHave = ":restore"
const SharingPermissionMustHave = ":sharing"

const DirectoryLinuxPermission = 0770
const FileLinuxPermission = 0660
const NexAuthResultKey = "X-NEXAUTH-CHECK"
const NexAuthHeaderScopeConstanta = "X-NEXAUTH-SCOPE"
const NexAuthHeaderCheckClientConstanta = "X-NEXAUTH-CLIENT-CHECK"
const NexAuthEmptyTokenResult = "EMPTY"
const NexAuthFixedTokenResult = "FIXED"
const NexAuthInternalTokenResult = "INTERNAL"
const NexAuthUserTokenResult = "USER"

const AuthorizationHeaderConstanta = "Authorization"
const RefreshTokenHeaderConstanta = "Refresh-Token"
const RequestIDConstanta = "X-REQUEST-ID"
const ForceHeaderCodeResponse = "X-FORCE-HEADER-CODE"
const IPAddressConstanta = "X-Forwarded-For"
const SourceConstanta = "X-SOURCE"
const RedirectResponseValidateConstanta = "X-NEXREDIRECT"
const TokenResponseValidateConstanta = "X-NEXTOKEN"
const RefreshResponseValidateConstanta = "X-NEXREFRESH"
const StateResponseValidateConstanta = "X-NEXSTATE"
const CodeResponseValidateConstanta = "X-Nexcode"
const DefaultTokenKeyConstanta = "X-NEXTOKEN"
const TimestampSignatureHeaderNameConstanta = "X-Timestamp"
const SignatureHeaderNameConstanta = "X-Signature"
const DeviceHeaderConstanta = "X-Device"
const ResourceHeaderConstanta = "X-Resource"
const RedirectURINameConstanta = "X-Redirect-Uri"
const RedirectMethodNameConstanta = "X-Redirect-Method"
const ClientRequestTimestamp = "X-Client-Timestamp"
const ApplicationContextConstanta = "application_context"
const IdempotencyHeaderKey = "X-Idempotency-Key"

const UpdateLastUpdateTimeInMinute = 3
const JobProcessErrorStatus = "ERROR"
const JobProcessOnProgressStatus = "ONPROGRESS"
const JobProcessOnProgressErrorStatus = "ONPROGRESS-ERROR"
const JobProcessDoneStatus = "OK"
const JobProcessHousekeepingGroup = "Housekeeping"
const JobProcessSynchronizeGroup = "Synchronize"
const JobProcessElasticType = "Elastic"

const INVALID_TOKEN_REDIS_VALUE = "INVALID"

const NexternalResultValueExternal = "EXTERNALCLIENT"
const NexternalAccountIDKey = "X-NEXTERNAL-ACCOUNT"
const NexternalClientIPKey = "X-NEXCLIENT-IP"
const RESOURCE_CHECKER_DISABLED = "X-RESOURCE_CHECKER_DISABLED"
const DEFAULT_REDIS_VALUE_KEY = "additional_parse"

var IndonesianTimeByTimeOffset = map[int]string{
	28800: "WITA",
	25200: "WIB",
	32400: "WIT",
}

var BoolKind = reflect.TypeOf(sql.NullBool{}).Name()
var StringKind = reflect.TypeOf(sql.NullString{}).Name()
var Float64Kind = reflect.TypeOf(sql.NullFloat64{}).Name()
var Int16Kind = reflect.TypeOf(sql.NullInt16{}).Name()
var Int32Kind = reflect.TypeOf(sql.NullInt32{}).Name()
var Int64Kind = reflect.TypeOf(sql.NullInt64{}).Name()
var TimeKind = reflect.TypeOf(time.Time{}).Name()
var NullTimeKind = reflect.TypeOf(sql.NullTime{}).Name()
