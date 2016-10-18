package ucenter

import (
	"crypto/md5"
	"fmt"
	"strconv"
	"time"
)

// UIDOffset 一毫秒内的偏移量
var UIDOffset = 0

// GetUID 得到全局唯一ID
// 时间毫秒(44) + 节点(5) + 自增长id(15位)
// 时间可以保证400年不重复
// 节点可以用于有多个机器同时产生id时，可以设置63个结点
// 每一毫秒能生成最多65535个id,但是由于机器的限制，
// 一毫秒内光调用GetUID也不能达到60000次(按3GHz算，相当于30个指令周期调用一次)
// 所以综合总论400年内不可能有重复
// 0           45     50          64
// +-----------+------+------------+
// |timestamp  |node  |increment   |
// +-----------+------+------------+
func GetUID(node int) uint64 {
	now := time.Now().UnixNano() / int64(1e6)
	UIDOffset++
	if UIDOffset > 1024 {
		UIDOffset = 0
	}
	value := (now << 20) + int64(node<<15) + int64(UIDOffset)
	return uint64(value)
}

// GetNewToken 产生新的token
func GetNewToken() string {
	UID := GetUID(Config.NodeIdentfy)
	token := md5.Sum([]byte(strconv.FormatInt(int64(UID), 10)))
	tokenStr := fmt.Sprintf("%x", token)
	return tokenStr
}
