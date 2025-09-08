package app

import(
	"strconv"
	"strings"
	"errors"
)

func StrUnpack(s string) (string, error) {
	var ans strings.Builder
	var n int
	var chTmp byte
	var ekr bool
	if s == ""{
		return "", nil
	}
	if _, err := strconv.Atoi(s); err == nil {
		return "", errors.New("string contains only numbers")
	}
	for idx, ch := range s{
		if ch >= '0' && ch <= '9' && !ekr{
			if idx == 0{
				continue
			}
			var err error
			n, err = strconv.Atoi(string(ch))
			if err != nil{
				return "", err
			}
			if n == 0{
				return "", errors.New("digits must be grower than 0")
			}
			for i := 0; i<n-1; i++{
				ans.WriteByte(chTmp)
			}
			continue
		}
		ekr = false
		if ch == '\\'{
			ekr = true
			continue
		}
		ans.WriteByte(byte(ch))
		chTmp = byte(ch)
	}
	return ans.String(), nil
}