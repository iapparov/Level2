package app

import(
	"strconv"
	"strings"
)

func StrUnpack(s string) (string, error) {
	var ans strings.Builder
	var n int
	for _, ch := range s{
		if ch >= '0' && ch <= '9'{
			n, err := strconv.Atoi(string(ch))
			if err != nil{
				return "", err
			}
		} else {
			ans.WriteByte(byte(ch))
		}
		if n != 0{
			for range n{
				ans.WriteByte(byte(ch))
			}
			n = 0
		}
	}
	return ans.String(), nil
}