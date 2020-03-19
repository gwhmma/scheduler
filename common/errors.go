package common

import "errors"

var (
	ERR_NO_LOCAL_IP_FOUND = errors.New("无法找到本地IP")
	ERR_LOCK_ALREADY_REQUIRED = errors.New("锁已被占用")
)

