package users

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"math"
	"net/http"
	"strings"
)

func Entropy(b []byte) float64 {
	var score float64
	var freqArray [256]float64
	for i := 0; i < len(b); i++ {
		freqArray[b[i]]++
	}
	l := float64(len(b))
	for i := 0; i < 256; i++ {
		if freqArray[i] != 0 {
			freq := freqArray[i] / l
			score -= freq * math.Log2(freq)
		}
	}
	return score / 8
}

func CheckHIBP(password string) bool {
	hash := fmt.Sprintf("%X", sha1.Sum([]byte(password)))
	prefix := hash[:5]
	suffix := hash[5:]
	resp, err := http.DefaultClient.Get(fmt.Sprintf("https://api.pwnedpasswords.com/range/%s", prefix))
	if err != nil {
		// can't do much, fail insecurely to not disrupt functionality
		return false
	}
	defer resp.Body.Close()
	s := bufio.NewScanner(resp.Body)
	for s.Scan() {
		if strings.HasPrefix(s.Text(), suffix) {
			return true
		}
	}
	return false
}
