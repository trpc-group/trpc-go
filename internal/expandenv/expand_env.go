// Package expandenv replaces ${key} in byte slices with the env value of key.
package expandenv

import (
	"os"
)

// ExpandEnv looks for ${var} in s and replaces them with value of the corresponding environment variable.
// It's not like os.ExpandEnv which handles both ${var} and $var.
// $var is considered invalid, since configurations like password for redis/mysql may contain $.
func ExpandEnv(s []byte) []byte {
	var buf []byte
	i := 0
	for j := 0; j < len(s); j++ {
		if s[j] == '$' && j+2 < len(s) && s[j+1] == '{' { // only ${var} instead of $var is valid
			if buf == nil {
				buf = make([]byte, 0, 2*len(s))
			}
			buf = append(buf, s[i:j]...)
			name, w := getEnvName(s[j+1:])
			if name == nil && w > 0 {
				// invalid matching, remove the $
			} else if name == nil {
				buf = append(buf, s[j]) // keep the $
			} else {
				buf = append(buf, os.Getenv(string(name))...)
			}
			j += w
			i = j + 1
		}
	}
	if buf == nil {
		return s
	}
	return append(buf, s[i:]...)
}

// getEnvName gets env name, that is, var from ${var}.
// The env name and its len will be returned.
func getEnvName(s []byte) ([]byte, int) {
	// look for right curly bracket '}'
	// it's guaranteed that the first char is '{' and the string has at least two char
	for i := 1; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\n' || s[i] == '"' { // "xx${xxx"
			return nil, 0 // encounter invalid char, keep the $
		}
		if s[i] == '}' {
			if i == 1 { // ${}
				return nil, 2 // remove ${}
			}
			return s[1:i], i + 1
		}
	}
	return nil, 0 // no }，keep the $
}
