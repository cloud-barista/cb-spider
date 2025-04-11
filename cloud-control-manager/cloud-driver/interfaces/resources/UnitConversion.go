package resources

import (
	"fmt"
	"strconv"
)

// ConvertMBToMiB converts an MB string to a MiB string (int string).
func ConvertMBToMiB(mbStr string) (string, error) {
	mb, err := strconv.Atoi(mbStr)
	if err != nil {
		return "-1", fmt.Errorf("invalid MB value: %v", err)
	}
	return ConvertMBToMiBInt64(int64(mb)), nil
}

// ConvertMBToMiBInt64 converts MB (int64) to a MiB string (int string).
func ConvertMBToMiBInt64(mb int64) string {
	mib := int(mb * 1000 / 1024)
	return strconv.Itoa(mib)
}

// ConvertMiBToGB converts a MiB string to a GB string (int string).
func ConvertMiBToGB(mibStr string) (string, error) {
	mib, err := strconv.Atoi(mibStr)
	if err != nil {
		return "-1", fmt.Errorf("invalid MiB value: %v", err)
	}
	return ConvertMiBToGBInt64(int64(mib)), nil
}

// ConvertMiBToGBInt64 converts MiB (int64) to a GB string (int string).
func ConvertMiBToGBInt64(mib int64) string {
	gb := int(float64(mib) / 1024.0 * 1.073741824)
	return strconv.Itoa(gb)
}

// ConvertGBToMiB converts a GB string to a MiB string (int string).
func ConvertGBToMiB(gbStr string) (string, error) {
	gb, err := strconv.Atoi(gbStr)
	if err != nil {
		return "-1", fmt.Errorf("invalid GB value: %v", err)
	}
	return ConvertGBToMiBInt64(int64(gb)), nil
}

// ConvertGBToMiBInt64 converts GB (int64) to a MiB string (int string).
func ConvertGBToMiBInt64(gb int64) string {
	mib := int(float64(gb) / 1.073741824 * 1024.0)
	return strconv.Itoa(mib)
}

// ConvertGiBToGB converts a GiB string to a GB string (int string).
func ConvertGiBToGB(gibStr string) (string, error) {
	gib, err := strconv.Atoi(gibStr)
	if err != nil {
		return "-1", fmt.Errorf("invalid GiB value: %v", err)
	}
	return ConvertGiBToGBInt64(int64(gib)), nil
}

// ConvertGiBToGBInt64 converts GiB (int64) to a GB string (int string).
func ConvertGiBToGBInt64(gib int64) string {
	gb := int(float64(gib) * 1.073741824)
	return strconv.Itoa(gb)
}

// ConvertGiBToMiB converts a GiB string to a MiB string (int string).
func ConvertGiBToMiB(gibStr string) (string, error) {
	gib, err := strconv.Atoi(gibStr)
	if err != nil {
		return "-1", fmt.Errorf("invalid GiB value: %v", err)
	}
	return ConvertGiBToMiBInt64(int64(gib)), nil
}

// ConvertGiBToMiBInt64 converts GiB (int64) to a MiB string (int string).
func ConvertGiBToMiBInt64(gib int64) string {
	mib := int(gib * 1024)
	return strconv.Itoa(mib)
}

// ConvertByteToMiB converts a Byte string to a MiB string (int string).
func ConvertByteToMiB(byteStr string) (string, error) {
	byteValue, err := strconv.Atoi(byteStr)
	if err != nil {
		return "-1", fmt.Errorf("invalid Byte value: %v", err)
	}
	return ConvertByteToMiBInt64(int64(byteValue)), nil
}

// ConvertByteToMiBInt64 converts Byte (int64) to a MiB string (int string).
func ConvertByteToMiBInt64(byteValue int64) string {
	mib := int(byteValue / (1024 * 1024))
	return strconv.Itoa(mib)
}

// ConvertByteToGB converts a Byte string to a GB string (int string).
func ConvertByteToGB(byteStr string) (string, error) {
	byteValue, err := strconv.Atoi(byteStr)
	if err != nil {
		return "-1", fmt.Errorf("invalid Byte value: %v", err)
	}
	return ConvertByteToGBInt64(int64(byteValue)), nil
}

// ConvertByteToGBInt64 converts Byte (int64) to a GB string (int string).
func ConvertByteToGBInt64(byteValue int64) string {
	gb := int(byteValue / (1000 * 1000 * 1000))
	return strconv.Itoa(gb)
}
