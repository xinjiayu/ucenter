package ucenter

import (
	"crypto/md5"
	"database/sql"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"time"
	// for mysql driver
	_ "github.com/go-sql-driver/mysql"
)

var (
	// Config configure must initialization before call Init()
	// default config not use redis
	Config = Configure{
		UserTableName:         "uc_users",
		TokenTablename:        "uc_user_token",
		TokenExpiresIn:        7 * 24 * 60 * 60, // one week
		SessionExpiresIn:      24 * 60 * 60,     // a day
		PreTokenExpireIn:      2 * 60 * 60,      // two hours
		InMemoryCacheExpireIn: 2 * 60 * 60,      // two hours
	}

	// inner variable
	db                  *sql.DB
	accessTokenCache    *Cache
	preAccessTokenCache *Cache
	sessionCache        *Cache
	redisPool           *redis.Pool
)

var (
	// ErrUserExist user has exits for register
	ErrUserExist = errors.New("user name has exist")

	// ErrUserNotExist user has not exist
	ErrUserNotExist = errors.New("user has not exist")

	// ErrParamInvalid param not valid
	ErrParamInvalid = errors.New("param not valid")

	// ErrPwdInvalid password invalid
	ErrPwdInvalid = errors.New("password  invalid")

	// ErrSetRefreshToken set refresh_token error
	ErrSetRefreshToken = errors.New("set refresh_token error")

	// ErrSetAccessToken set access_token error
	ErrSetAccessToken = errors.New("set access_token error")

	// ErrSetPreAccessToken set pre_access_token error
	ErrSetPreAccessToken = errors.New("set pre_access_token error")

	// ErrRefreshTokenInvalid refresh token is invalid
	ErrRefreshTokenInvalid = errors.New("refresh token is invalid")

	// ErrAccessTokenInvalid access_token is invalid
	ErrAccessTokenInvalid = errors.New("access_token is invalid")

	// ErrTokenNotExist token not exist
	ErrTokenNotExist = errors.New("token not exist")

	// ErrTokenExpired token have expired
	ErrTokenExpired = errors.New("token have expired")

	// ErrTimeParse parse string format to Time error
	ErrTimeParse = errors.New("parse string format to Time error")

	// ErrSetRedis set key/value to reids error
	ErrSetRedis = errors.New("set key/value to reids error")

	// ErrGetRedis get key from reids error
	ErrGetRedis = errors.New("get key from reids error")
)

// Configure configure for data and validation
type Configure struct {
	// MysqlConnStr like root:@/ucenter?charset=utf8
	MysqlConnStr   string
	UserTableName  string
	TokenTablename string
	NodeIdentfy    int
	// access_token expires_in
	TokenExpiresIn   int
	PreTokenExpireIn int
	// session expires_in
	SessionExpiresIn int
	// RedisConnStr connect string for redis, "172.17.0.89:6379"
	RedisConnStr          string
	InMemoryCacheExpireIn int
}

// UserInfo user basic information
type UserInfo struct {
	ID         int64
	UserName   string
	Nickname   string
	Email      string
	Password   string
	Registered string
}

// LoginResult Login result
type LoginResult struct {
	RefreshToken         string
	AccessToken          string
	Session              string
	AccessTokenExpiresIn int
	SessionExpiresIn     int
}

// Init check environment and init settings
// not write in init because of need config
func Init() {
	if len(Config.MysqlConnStr) == 0 {
		fmt.Println("please set config.MysqlConnStr for connect mysql")
		return
	}
	var err error
	db, err = sql.Open("mysql", Config.MysqlConnStr)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = makeSureUserTableExist()
	if err != nil {
		fmt.Println(err)
		return
	}
	if len(Config.RedisConnStr) == 0 {
		accessTokenCache = &Cache{expire: Config.InMemoryCacheExpireIn}
		accessTokenCache.Init()
		preAccessTokenCache = &Cache{expire: Config.InMemoryCacheExpireIn}
		preAccessTokenCache.Init()
		sessionCache = &Cache{expire: Config.SessionExpiresIn}
		sessionCache.Init()
	} else {
		redisPool = &redis.Pool{
			MaxIdle:     3,                 // adjust to your needs
			IdleTimeout: 240 * time.Second, // adjust to your needs
			Dial: func() (redis.Conn, error) {
				c, err := redis.Dial("tcp", Config.RedisConnStr)
				if err != nil {
					return nil, err
				}
				return c, err
			},
		}
	}
}

// UserRegister register must have set username and password
func UserRegister(user UserInfo) error {
	if len(user.UserName) == 0 || len(user.Password) == 0 {
		return ErrParamInvalid
	}
	u, _ := getUserByName(user.UserName)
	if u != nil {
		return ErrUserExist
	}
	err := createUser(user)
	if err != nil {
		return err
	}
	return nil
}

// UserLogin  user login, if login succeed will return two token string
// first token : refresh_token
// second token: access_token
func UserLogin(name string, password string) (*LoginResult, error) {
	if len(name) == 0 || len(password) == 0 {
		return nil, ErrParamInvalid
	}
	u, err := getUserByName(name)
	if err != nil {
		return nil, err
	}
	pwd := md5.Sum([]byte(password))
	pwdStr := fmt.Sprintf("%x", pwd)
	if pwdStr != u.Password {
		return nil, ErrPwdInvalid
	}
	refreshToken := GetNewToken()
	err = SetRefreshToken(name, refreshToken)
	if err != nil {
		return nil, ErrSetRefreshToken
	}
	accessToken := GetNewToken()
	err = SetAccessToken(name, accessToken)
	if err != nil {
		return nil, ErrSetAccessToken
	}
	SetPreAccessToken(name, "")

	session := GetNewToken()

	// cache token and session if not use redis
	if redisPool == nil {
		accessTokenCache.Set(name, accessToken)
		preAccessTokenCache.Delete(name)
		sessionCache.Set(name, session)
	}

	return &LoginResult{refreshToken, accessToken, session,
		Config.TokenExpiresIn, Config.SessionExpiresIn}, nil
}

// CheckAccessToken check user is valid?
// because of access_token maybe check every request in app, so
// need save it in cache used to reduce the load
func CheckAccessToken(name string, accessToken string) error {
	// if not use redis, check in-memory cache first
	if redisPool == nil {
		token := accessTokenCache.Get(name)
		if len(token) > 0 { // have load from database
			if token == accessToken {
				return nil
			}
			preToken := preAccessTokenCache.Get(name)
			if preToken == accessToken {
				return nil
			}
			return ErrAccessTokenInvalid
		}
	}
	t, err := GetTokenInfo(name)
	if err != nil {
		return err
	}

	// check redis
	if redisPool != nil {
		if accessToken == t.AccessToken ||
			accessToken == t.PreAccessToken {
			return nil
		}
		return ErrAccessTokenInvalid
	}

	// check database
	now := time.Now()
	tokenCreated, err := time.Parse("2006-01-02 15:04:05",
		t.AccessTokenCreated)
	if err != nil {
		return ErrTimeParse
	}
	if now.Unix()-tokenCreated.Unix() > int64(Config.TokenExpiresIn) ||
		t.AccessToken == "" {
		// expire_in or kill down
		preAccessTokenCache.Set(name, "nil")
		accessTokenCache.Set(name, "nil")
		return ErrTokenExpired
	}
	// database have right value
	if redisPool == nil {
		preAccessTokenCache.Set(name, t.PreAccessToken)
		accessTokenCache.Set(name, t.AccessToken)
	}

	if t.AccessToken != accessToken {
		// pre_access_token is valid in 2 hours
		if now.Unix()-tokenCreated.Unix() <
			int64(Config.PreTokenExpireIn) {
			if accessToken == t.PreAccessToken {
				return nil
			}
		}
		return ErrAccessTokenInvalid
	}
	return nil
}

// ResetAccessToken reset the access_token by refreshToken
// because of access_token maybe check every request in app, so
// need save it in cache used to reduce the load
func ResetAccessToken(name string, refreshToken string) (string, error) {
	t, err := GetTokenInfo(name)
	if err != nil {
		return "", err
	}
	if t.RefreshToken != refreshToken {
		return "", ErrRefreshTokenInvalid
	}
	err = SetPreAccessToken(name, t.AccessToken)
	if err != nil {
		return "", err
	}
	AccessToken := GetNewToken()
	err = SetAccessToken(name, AccessToken)
	if err != nil {
		return "", err
	}
	if redisPool == nil {
		accessTokenCache.Set(name, AccessToken)
		preAccessTokenCache.Set(name, t.AccessToken)
	}

	return AccessToken, nil
}

// CheckSession check session for web site,
// and it will auto refresh session expires_in
func CheckSession(name string, session string) bool {
	if redisPool != nil {
		c := redisPool.Get()
		defer c.Close()
		s, err := redis.String(c.Do("GET", "session@"+name))
		if err != nil {
			fmt.Println("redis get failed:", err)
			return false
		}
		if s == session {
			return true
		}
		return false
	}
	s := sessionCache.Get(name)
	if len(s) == 0 || s != session {
		return false
	}

	sessionCache.Set(name, session)
	return true
}

// GetUserInfo get user basic info but not contain authentication information
func GetUserInfo(name string) (*UserInfo, error) {
	u, err := getUserByName(name)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// KillOffLine will delete user token
func KillOffLine(name string) error {
	_, err := getUserByName(name)
	if err != nil {
		return err
	}
	SetRefreshToken(name, "")
	SetAccessToken(name, "")
	SetPreAccessToken(name, "")
	sessionCache.Set(name, "")

	return nil
}

func makeSureUserTableExist() error {
	// check user table have created
	tables, err := getAllTables()
	if err != nil {
		return err
	}
	findedUserTable := false
	for i := 0; i < len(tables); i++ {
		if Config.UserTableName == tables[i] {
			findedUserTable = true
			break
		}
	}
	if !findedUserTable {
		fmt.Println("not find " + Config.UserTableName)
		err := createUserTable()
		if err != nil {
			return err
		}
	}
	if redisClient == nil {
		findedTokenTable := false
		for i := 0; i < len(tables); i++ {
			if Config.TokenTablename == tables[i] {
				findedTokenTable = true
				break
			}
		}
		if !findedTokenTable {
			err := createUserTokenTable()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func getAllTables() ([]string, error) {
	// 得到所有的分类
	rows, err := db.Query("show tables like '%%'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var table string
		if rows.Scan(&table) == nil {
			tables = append(tables, table)
		}
	}
	return tables, nil
}

// create user table
func createUserTable() error {
	createStr := "create table " + Config.UserTableName + "(" +
		"ID               bigint(20) unsigned NOT NULL AUTO_INCREMENT," +
		"user_name        varchar(60) NOT NULL DEFAULT ''," +
		"user_pass        varchar(255) NOT NULL DEFAULT ''," +
		"user_nicename    varchar(50) NOT NULL DEFAULT ''," +
		"user_email       varchar(100) NOT NULL DEFAULT ''," +
		"user_registered  datetime NOT NULL DEFAULT CURRENT_TIMESTAMP," +
		"PRIMARY KEY (`ID`), " +
		"KEY `user_name` (`user_name`), " +
		"KEY `user_email` (`user_email`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8"
	_, err := db.Exec(createStr)
	if err != nil {
		return err
	}
	return nil
}

// if not use redis, this information need save in database
func createUserTokenTable() error {
	createStr := "create table  " + Config.TokenTablename + " (" +
		"user_name        varchar(255) NOT NULL DEFAULT ''," +
		"refresh_token    varchar(255) NOT NULL DEFAULT ''," +
		"rtoken_created   datetime NOT NULL DEFAULT CURRENT_TIMESTAMP," +
		"access_token     varchar(255) NOT NULL DEFAULT ''," +
		"atoken_created   datetime NOT NULL DEFAULT CURRENT_TIMESTAMP," +
		"pre_access_token varchar(255) NOT NULL DEFAULT ''," +
		"KEY `user_name` (`user_name`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8"
	_, err := db.Exec(createStr)
	if err != nil {
		return err
	}
	return nil
}
