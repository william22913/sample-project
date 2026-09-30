package basic_validator

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/nexsoft-git/nexcommon/regex"
)

func (bv BasicValidator) CheckIsResourceIDExist(availableResourceID string, resourceIDMustHave string) bool {
	splitDot := strings.Split(availableResourceID, " ")
	return bv.ValidateStringContainInStringArray(splitDot, resourceIDMustHave)
}

func (bv BasicValidator) CheckIsScopeExist(availableScope string, scopeMustHave string) bool {
	splitDot := strings.Split(availableScope, " ")
	return bv.ValidateStringContainInStringArray(splitDot, scopeMustHave)
}

func (bv BasicValidator) ValidateStringContainInStringArray(listString []string, key string) bool {
	for i := 0; i < len(listString); i++ {
		if listString[i] == key {
			return true
		}
	}
	return false
}

func (bv BasicValidator) IsMacAddress(input string) (output bool) {
	temp := strings.Split(input, ":")
	if len(temp) == 6 {
		output = true
		for i := 0; i < len(temp); i++ {
			if len(temp[i]) != 2 {
				output = false
			}
		}
		return
	}
	return false
}

func (bv BasicValidator) IsEmailAddress(input string) (output bool) {
	emailRegexp := regexp.MustCompile(regex.EMAIL_REGEX)
	return emailRegexp.MatchString(input)
}

func (bv BasicValidator) IsPhoneNumber(input string) (number int, isValid bool) {
	number, isValid = bv.IsNumeric(input)
	return number, len(input) <= 13 && isValid
}

func (bv BasicValidator) IsPhoneNumberWithCountryCode(input string) bool {
	phoneNumberRegexp := regexp.MustCompile(regex.PHONE_NUMBER_WITH_COUNTRY_CODE)
	return phoneNumberRegexp.MatchString(input)
}

func (bv BasicValidator) IsCountryCode(input string) bool {
	countryCodeRegexp := regexp.MustCompile(regex.COUNTRY_CODE)
	return countryCodeRegexp.MatchString(input)
}

func (bv BasicValidator) IsIPPrivate(input string) (output bool) {
	ipPrivateRegexp := regexp.MustCompile(regex.IP_PRIVATE_LOCALHOST)
	if ipPrivateRegexp.MatchString(input) {
		return true
	}
	ipPrivateRegexp = regexp.MustCompile(regex.IP_PRIVATE_192)
	if ipPrivateRegexp.MatchString(input) {
		return true
	}

	ipPrivateRegexp = regexp.MustCompile(regex.IP_PRIVATE_OTHER)
	return ipPrivateRegexp.MatchString(input)
}

func (bv BasicValidator) IsNexsoftPasswordStandardValid(password string) (bool, string, string) {
	if len(password) < 8 {
		return false, "NEED_MORE_THAN", "8"
	} else if len(password) >= 8 && len(password) <= 50 {
	next:
		for name, classes := range map[string][]*unicode.RangeTable{
			"UPPERCASE": {unicode.Upper, unicode.Title},
			"LOWERCASE": {unicode.Lower},
			"NUMERIC":   {unicode.Number, unicode.Digit},
			"SPECIAL":   {unicode.Space, unicode.Symbol, unicode.Punct, unicode.Mark},
		} {
			for _, r := range password {
				if unicode.IsOneOf(classes, r) {
					continue next
				}
			}
			return false, name, ""
		}
	} else {
		return false, "NEED_LESS_THAN", "50"
	}
	return true, "", ""
}

func (bv BasicValidator) IsNexsoftUsernameStandardValid(username string) (bool, string, string) {
	if len(username) < 6 {
		return false, "NEED_MORE_THAN", "6"
	} else if len(username) > 20 {
		return false, "NEED_LESS_THAN", "20"
	} else {
		usernameRegex := regexp.MustCompile(regex.USERNAME)
		return usernameRegex.MatchString(username), "USERNAME_REGEX_MESSAGE", ""
	}
}

func (bv BasicValidator) IsNexsoftNameStandardValid(username string) (bool, string, string) {
	usernameRegex := regexp.MustCompile(regex.NAME_STANDARD)
	return usernameRegex.MatchString(username), "NAME_REGEX_MESSAGE", ""
}

func (bv BasicValidator) IsNexsoftAdditionalInformationKeyStandardValid(username string) (bool, string) {
	usernameRegex := regexp.MustCompile(regex.ADDITIONAL_INFO)
	return usernameRegex.MatchString(username), "ADDITIONAL_INFO_REGEX"
}

func (bv BasicValidator) IsOnlyContainLowerCase(username string) (bool, string) {
	usernameRegex := regexp.MustCompile(regex.LOWERCASE)
	return usernameRegex.MatchString(username), "LOWERCASE_REGEX"
}

func (bv BasicValidator) IsOnlyContainLowerCaseAndNumber(username string) (bool, string) {
	usernameRegex := regexp.MustCompile(regex.LOWERCASE_AND_NUMBER)
	return usernameRegex.MatchString(username), "LOWERCASE_AND_NUMBER_REGEX"
}

func (bv BasicValidator) IsStringEmpty(input string) bool {
	return input == ""
}

func (bv BasicValidator) IsTimestampValid(input string) (bool, string) {
	format := "2006-01-02T15:04:05.999999999"
	timestamp, err := time.Parse(format, input)

	if err != nil {
		return false, ""
	} else {
		return true, strings.Replace(timestamp.UTC().Format(time.RFC3339Nano), "Z", "", -1)
	}
}

func (bv BasicValidator) IsNumeric(input string) (int, bool) {
	result, err := strconv.Atoi(input)
	if err != nil {
		return -1, false
	} else {
		return result, true
	}
}

func (bv BasicValidator) IsNexsoftPermissionStandardValid(permission string) (bool, string) {
	permissionRegex := regexp.MustCompile(regex.PERMISSION)
	return permissionRegex.MatchString(permission), "PERMISSION_REGEX_MESSAGE"
}

func (bv BasicValidator) IsNPWPValid(npwp string) bool {
	npwpRegex := regexp.MustCompile(regex.NPWP)
	return npwpRegex.MatchString(npwp)
}

func (bv BasicValidator) IsNIKValid(nik string) bool {
	nikRegex := regexp.MustCompile(regex.NIK)
	return nikRegex.MatchString(nik)
}

func (bv BasicValidator) IsFacsimileValid(fax string) bool {
	facsimileRegex := regexp.MustCompile(regex.FAX)
	return facsimileRegex.MatchString(fax)
}

func (bv BasicValidator) IsNexsoftProfileNameStandardValid(profileName string) (bool, string) {
	NameOrTitle := regexp.MustCompile(regex.PROFILE_NAME)
	return NameOrTitle.MatchString(profileName), "PROFILE_NAME_REGEX_MESSAGE"
}

func (bv BasicValidator) IsNameWithUppercaseValid(input string) bool {
	UppercaseRegex := regexp.MustCompile(regex.UPERCASE)
	return UppercaseRegex.MatchString(input)
}

func (bv BasicValidator) IsLongNumeric(input string) bool {
	longNumericRegex := regexp.MustCompile(regex.LONG_NUMERIC)
	return longNumericRegex.MatchString(input)
}

func (bv BasicValidator) IsDataScopeValid(input string) (bool, string) {
	dataScopeRegexp := regexp.MustCompile(regex.DATA_SCOPE)
	return dataScopeRegexp.MatchString(input), "DATA_SCOPE"
}

func (bv BasicValidator) IsNexsoftDirectoryNameStandardValid(profileName string) (bool, string) {
	NameOrTitle := regexp.MustCompile(regex.DIRECTORY_NAME)
	return NameOrTitle.MatchString(profileName), "DIRECTORY_NAME_REGEX_MESSAGE"
}

func (bv BasicValidator) IsIPContainsInWhiteListIP(ipWhitelist string, ip string) bool {
	prefixIPV6 := "(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))"
	ipv6Regex := regexp.MustCompile(prefixIPV6)

	splitSpace := strings.Split(ipWhitelist, " ")
	var isFound bool
	for i := 0; i < len(splitSpace); i++ {
		if ipv6Regex.MatchString(splitSpace[i]) {
			if splitSpace[i] == ip {
				return true
			}
		} else {
			if strings.Contains(splitSpace[i], "-") {
				isFound = bv.ValidateIPV4WithStrip(splitSpace[i], ip)
				if isFound {
					return true
				}
			} else if strings.Contains(splitSpace[i], "/") {
				isFound = bv.ValidateIPV4WithStripMasking(splitSpace[i], ip)
				if isFound {
					return true
				}
			} else {
				if splitSpace[i] == ip {
					return true
				}
			}
		}
	}
	return false
}

func (bv BasicValidator) ValidateIPV4WithStrip(ipWhitelist string, ip string) bool {
	splitStrip := strings.Split(ipWhitelist, "-")
	if splitStrip[0] == splitStrip[1] {
		if splitStrip[0] == ip {
			return true
		}
		return false
	}
	splitDot0 := strings.Split(splitStrip[0], ".")
	splitDot1 := strings.Split(splitStrip[1], ".")
	ipSplitDot := strings.Split(ip, ".")
	if len(splitDot0) != 4 && len(splitDot1) != 4 {
		return false
	}

	for i := 0; i < 4; i++ {
		beforeIsBigger := false
		found := false

		splitDot0Int, err := strconv.Atoi(splitDot0[i])
		if err != nil {
			return false
		}
		splitDot1Int, err := strconv.Atoi(splitDot1[i])
		if err != nil {
			return false
		}
		ipSplitDotInt, err := strconv.Atoi(ipSplitDot[i])
		if err != nil {
			return false
		}

		startCheck := splitDot0Int
		if beforeIsBigger {
			startCheck = 1
		}

		if !beforeIsBigger {
			if splitDot1Int > splitDot0Int {
				beforeIsBigger = true
			}
		}

		for j := startCheck; j <= splitDot1Int; j++ {
			if ipSplitDotInt == j {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (bv BasicValidator) ValidateIPV4WithStripMasking(ipWhitelist string, ip string) bool {
	splitStrip := strings.Split(ipWhitelist, "/")
	var ipBinary string

	if len(splitStrip) != 2 {
		return false
	}

	intMask, err := strconv.Atoi(splitStrip[1])
	if err != nil || intMask < 1 {
		return false
	}

	ipBinary, err = bv.ConvertIPToBinary(splitStrip[0])
	if err != nil {
		return false
	}

	firstIP, lastIP, err := bv.GetAvailableIPFromPrefix(ipBinary, intMask)
	if err != nil {
		return false
	}

	return bv.ValidateIPV4WithStrip(firstIP+"-"+lastIP, ip)
}

func (bv BasicValidator) ConvertIPToBinary(ip string) (output string, err error) {
	splitDot := strings.Split(ip, ".")
	if len(splitDot) != 4 {
		return "", errors.New("invalid length")
	}
	for i := 0; i < len(splitDot); i++ {
		var splitDotInt int
		splitDotInt, err = strconv.Atoi(splitDot[i])
		if err != nil {
			return
		}
		intLeft := float64(splitDotInt)
		for j := 7; j >= 0; j-- {
			powResult := math.Pow(float64(2), float64(j))
			if intLeft/powResult >= 1 {
				intLeft -= powResult
				output += "1"
			} else {
				output += "0"
			}
		}
		if i < len(splitDot)-1 {
			output += "."
		}
	}
	return
}

func (bv BasicValidator) ConvertBinaryToIP(ipBinary string) (output string, err error) {
	splitDot := strings.Split(ipBinary, ".")
	if len(splitDot) != 4 {
		return "", errors.New("invalid length")
	}

	for i := 0; i < len(splitDot); i++ {
		var temp float64
		for j := 0; j < len(splitDot[i]); j++ {
			byteData, _ := strconv.Atoi(string(splitDot[i][j]))
			powResult := math.Pow(float64(2), float64(7-j))
			temp += float64(byteData) * powResult
		}
		output += strconv.Itoa(int(temp))
		if i < len(splitDot)-1 {
			output += "."
		}
	}

	return
}

func (bv BasicValidator) GetAvailableIPFromPrefix(ipBinary string, prefix int) (firstIP string, lastIP string, err error) {
	if prefix < 0 && prefix > 31 {
		err = errors.New("unknown prefix")
		return
	}

	temp := ipBinary[0 : prefix+(prefix/8)]

	firstIP = temp
	lastIP = temp

	for i := prefix + (prefix / 8); i < len(ipBinary); i++ {
		if i < len(ipBinary)-1 {
			if string(ipBinary[i]) != "." {
				firstIP += "0"
				lastIP += "1"
			} else {
				firstIP += "."
				lastIP += "."
			}
		} else {
			firstIP += "1"
			lastIP += "0"
		}
	}

	firstIP, _ = bv.ConvertBinaryToIP(firstIP)
	lastIP, _ = bv.ConvertBinaryToIP(lastIP)
	return
}

var pattern = regexp.MustCompile(`^P((?P<year>\d+)Y)?((?P<month>\d+)M)?((?P<week>\d+)W)?((?P<day>\d+)D)?(T((?P<hour>\d+)H)?((?P<minute>\d+)M)?((?P<second>\d+)S)?)?$`)

func (bv BasicValidator) ValidateISO8601Duration(
	duration string,
) (
	time.Duration,
	error,
) {
	var match []string
	var d Duration

	if pattern.MatchString(duration) {
		match = pattern.FindStringSubmatch(duration)
	} else {
		return 0, errors.New("could not parse duration string")
	}

	var timeValid = strings.Contains(duration, "T")
	for i, name := range pattern.SubexpNames() {
		part := match[i]
		if i == 0 || name == "" || part == "" {
			continue
		}

		val, err := strconv.Atoi(part)
		if err != nil {
			return 0, err
		}

		switch name {
		case "year":
			d.Y = val
		case "month":
			d.M = val
		case "week":
			d.W = val
		case "day":
			d.D = val
		case "hour":
			d.TH = val
		case "minute":
			d.TM = val
		case "second":
			d.TS = val
		default:
			return 0, fmt.Errorf("unknown field %s", name)
		}
	}

	if timeValid && (d.TH == 0 && d.TM == 0 && d.TS == 0) {
		return 0, errors.New("time duration must be provided")
	}

	return d.timeDuration(), nil
}
