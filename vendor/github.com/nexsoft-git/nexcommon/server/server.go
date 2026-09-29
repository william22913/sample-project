package server

import (
	"bytes"
	"errors"
	"math"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func GenerateIPAddress(name string) (ip string, err error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	var first string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			if first == "" {
				first = ip.String()
			}
			if name != "" {
				if iface.Name != name {
					continue
				}
			}
			return ip.String(), nil
		}
	}
	if ip == "" {
		return first, nil
	}
	return "", err
}

func GenerateHostname() (string, error) {
	return os.Hostname()
}

func GetMacAddress() (addr string) {
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, i := range interfaces {
			if i.Flags&net.FlagUp != 0 && bytes.Compare(i.HardwareAddr, nil) != 0 {
				addr = i.HardwareAddr.String()
				break
			}
		}
	}
	return
}

func IsIPContainsInWhiteListIP(ipWhitelist string, ip string) bool {
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
				isFound = ValidateIPV4WithStrip(splitSpace[i], ip)
				if isFound {
					return true
				}
			} else if strings.Contains(splitSpace[i], "/") {
				isFound = ValidateIPV4WithStripMasking(splitSpace[i], ip)
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

func ValidateIPV4WithStrip(ipWhitelist string, ip string) bool {
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

	beforeIsBigger := false

	for i := 0; i < 4; i++ {
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

func ValidateIPV4WithStripMasking(ipWhitelist string, ip string) bool {
	splitStrip := strings.Split(ipWhitelist, "/")
	var ipBinary string

	if len(splitStrip) != 2 {
		return false
	}

	intMask, err := strconv.Atoi(splitStrip[1])
	if err != nil || intMask < 1 {
		return false
	}

	ipBinary, err = ConvertIPToBinary(splitStrip[0])
	if err != nil {
		return false
	}

	firstIP, lastIP, err := GetAvailableIPFromPrefix(ipBinary, intMask)
	if err != nil {
		return false
	}

	return ValidateIPV4WithStrip(firstIP+"-"+lastIP, ip)
}

func ConvertIPToBinary(ip string) (output string, err error) {
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

func ConvertBinaryToIP(ipBinary string) (output string, err error) {
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

func GetAvailableIPFromPrefix(ipBinary string, prefix int) (firstIP string, lastIP string, err error) {
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

	firstIP, _ = ConvertBinaryToIP(firstIP)
	lastIP, _ = ConvertBinaryToIP(lastIP)
	return
}
