package ucenter

import (
	"crypto/md5"
	"fmt"
	"github.com/garyburd/redigo/redis"
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

// TokenInfo token信息
type TokenInfo struct {
	UserName            string
	RefreshToken        string
	RefreshTokenCreated string
	AccessToken         string
	AccessTokenCreated  string
	PreAccessToken      string
}

// TokenType type of token
type TokenType string

const (
	refreshToken   TokenType = "refresh_token"
	accessToken    TokenType = "access_token"
	preAccessToken TokenType = "pre_access_token"
)

// SetRefreshToken set refresh token for database or redis
func SetRefreshToken(name string, token string) error {
	if redisClient == nil {
		u, err := GetTokenInfo(name)
		if u == nil {
			sql := "insert into " + Config.TokenTablename +
				"(user_name, refresh_token, rtoken_created)" +
				" values(?,?, now())"
			_, err := db.Exec(sql, name, token)
			if err != nil {
				fmt.Println(err)
				return ErrSetRefreshToken
			}
			return nil
		}
		sql := "update " + Config.TokenTablename +
			" set refresh_token= ?, " +
			" rtoken_created = now() where user_name=?"
		_, err = db.Exec(sql, token, name)
		if err != nil {
			fmt.Println(err)
			return ErrSetRefreshToken
		}
		return nil

	}
	// set redis cache, refresh_token 不设置过期时间
	_, err := (*redisClient).Do("SET", "refresh_token@"+name, token)
	if err != nil {
		fmt.Println(err)
		return ErrSetRefreshToken
	}
	return nil
}

// SetAccessToken set refresh_token for database or redis
func SetAccessToken(name string, token string) error {
	if redisClient == nil {
		u, err := GetTokenInfo(name)
		if u == nil {
			sql := "insert into " + Config.TokenTablename +
				"(user_name, access_token, atoken_created)" +
				" values(?,?, now())"
			_, err := db.Exec(sql, name, token)
			if err != nil {
				fmt.Println(err)
				return ErrSetAccessToken
			}
			return nil
		}
		sql := "update " + Config.TokenTablename +
			" set access_token= ?, " +
			" atoken_created = now() where user_name=?"
		_, err = db.Exec(sql, token, name)
		if err != nil {
			fmt.Println(err)
			return ErrSetAccessToken
		}
		return nil

	}
	// set redis cache, access_token
	_, err := (*redisClient).Do("SET", "access_token@"+name, token,
		"EX", strconv.Itoa(Config.TokenExpiresIn))
	if err != nil {
		fmt.Println(err)
		return ErrSetAccessToken
	}
	return nil
}

// SetPreAccessToken set refresh_token for database or redis
func SetPreAccessToken(name string, token string) error {
	if redisClient == nil {
		u, err := GetTokenInfo(name)
		if u == nil {
			sql := "insert into " + Config.TokenTablename +
				"(user_name, pre_access_token)" +
				" values(?,?)"
			_, err := db.Exec(sql, name, token)
			if err != nil {
				fmt.Println(err)
				return ErrSetPreAccessToken
			}
			return nil
		}
		sql := "update " + Config.TokenTablename +
			" set pre_access_token= ? " +
			"where user_name=?"
		_, err = db.Exec(sql, token, name)
		if err != nil {
			fmt.Println(err)
			return ErrSetPreAccessToken
		}
		return nil

	}
	// set redis cache, pre_access_token
	_, err := (*redisClient).Do("SET", "pre_access_token@"+name, token,
		"EX", strconv.Itoa(Config.PreTokenExpireIn))
	if err != nil {
		fmt.Println(err)
		return ErrSetPreAccessToken
	}
	return nil
}

// GetTokenInfo get token from database or redis
func GetTokenInfo(name string) (*TokenInfo, error) {
	if redisClient == nil {
		sql := "select user_name,refresh_token,rtoken_created," +
			"access_token,atoken_created,pre_access_token from " +
			Config.TokenTablename + " where user_name=?"
		rows, err := db.Query(sql, name)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var t TokenInfo
			if err = rows.Scan(&t.UserName, &t.RefreshToken,
				&t.RefreshTokenCreated, &t.AccessToken,
				&t.AccessTokenCreated,
				&t.PreAccessToken); err == nil {
				return &t, nil
			}
		}
		return nil, ErrTokenNotExist
	}
	refreshToken, err := redis.String((*redisClient).Do("GET",
		"refresh_token@"+name))
	if err != nil {
		fmt.Println("redis get failed:", err)
		return nil, ErrGetRedis
	}
	accessToken, err := redis.String((*redisClient).Do("GET",
		"access_token@"+name))
	if err != nil {
		fmt.Println("redis get failed:", err)
		return nil, ErrGetRedis
	}
	pretoken, err := redis.String((*redisClient).Do("GET",
		"pre_access_token@"+name))
	if err != nil {
		fmt.Println("redis get failed:", err)
		return nil, ErrGetRedis
	}
	var t TokenInfo
	t.UserName = name
	t.AccessToken = accessToken
	t.RefreshToken = refreshToken
	t.PreAccessToken = pretoken
	return &t, nil
}
