package centrauthsrv

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

func VerifyTimestamp(challenge string) error {
	// verify challenge timestamp < 30 seconds ago
	if len(strings.Split(challenge, "-")) < 2 {
		return errors.New("VerifyTimestamp: challenge timestamp error")
	}
	unixTimeChallenge, err := strconv.Atoi(strings.Split(challenge, "-")[1])
	if err != nil {
		return errors.New("VerifyTimestamp: challenge timestamp error")
	}
	t := time.Unix(int64(unixTimeChallenge), 10)
	elapsed := time.Since(t)
	if elapsed.Seconds() > 30000 { // 30 seconds to resolve challenge // DEV in development we use more time
		return errors.New("VerifyTimstamp: too much time elapsed sinse the challenge was sent")
	}
	return nil
}
