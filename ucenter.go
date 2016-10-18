package ucenter

import (
	"crypto/md5"
	"database/sql"
	"errors"
	"fmt"
	"time"
	// for mysql driver
	_ "github.com/go-sql-driver/mysql"
)

var (
	db *sql.DB
	// Config configure must initialization before call Init()
	Config = Configure{"", "users", 0, 7 * 24 * 60 * 60}
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

	// ErrRefreshTokenInvalid refresh token is invalid
	ErrRefreshTokenInvalid = errors.New("refresh token is invalid")

	// ErrAccessTokenInvalid access_token is invalid
	ErrAccessTokenInvalid = errors.New("access_token is invalid")

	// ErrTokenExpired token have expired
	ErrTokenExpired = errors.New("token have expired")

	// ErrTimeParse parse string format to Time error
	ErrTimeParse = errors.New("parse string format to Time error")
)

// Configure configure for data and validation
type Configure struct {
	// MysqlConnStr like root:@/ucenter?charset=utf8
	MysqlConnStr  string
	UserTableName string
	NodeIdentfy   int
	// access_token expires_in
	TokenExpiresIn int
}

// UserInfo user basic information
type UserInfo struct {
	ID            int64
	UserName      string
	Nickname      string
	Email         string
	Password      string
	Registered    string
	RefreshToken  string
	RTokenCreated string
	AccessToken   string
	ATokenCreated string
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
func UserLogin(name string, password string) (string, string, error) {
	if len(name) == 0 || len(password) == 0 {
		return "", "", ErrParamInvalid
	}
	u, err := getUserByName(name)
	if err != nil {
		return "", "", err
	}
	pwd := md5.Sum([]byte(password))
	pwdStr := fmt.Sprintf("%x", pwd)
	if pwdStr != u.Password {
		return "", "", ErrPwdInvalid
	}
	refreshToken := GetNewToken()
	err = resetRefreshToken(name, refreshToken)
	if err != nil {
		return "", "", ErrSetRefreshToken
	}
	AccessToken := GetNewToken()
	err = resetAccessToken(name, AccessToken)
	if err != nil {
		return "", "", ErrSetAccessToken
	}
	return refreshToken, AccessToken, nil
}

// CheckAccessToken check user is valid?
// because of access_token maybe check every request in app, so
// need save it in cache used to reduce the load
func CheckAccessToken(name string, accessToken string) error {
	u, err := getUserByName(name)
	if err != nil {
		return err
	}
	now := time.Now()
	tokenCreated, err := time.Parse("2006-01-02 15:04:05", u.ATokenCreated)
	if err != nil {
		return ErrTimeParse
	}
	if now.Unix()-tokenCreated.Unix() > int64(Config.TokenExpiresIn) {
		return ErrTokenExpired
	}
	if u.AccessToken != accessToken {
		return ErrAccessTokenInvalid
	}
	return nil
}

// ResetAccessToken reset the access_token by refreshToken
// because of access_token maybe check every request in app, so
// need save it in cache used to reduce the load
func ResetAccessToken(name string, refreshToken string) (string, error) {
	u, err := getUserByName(name)
	if err != nil {
		return "", err
	}
	if u.RefreshToken != refreshToken {
		return "", ErrRefreshTokenInvalid
	}
	AccessToken := GetNewToken()
	err = resetAccessToken(name, AccessToken)
	if err != nil {
		return "", ErrSetAccessToken
	}
	return AccessToken, nil
}

// GetUserInfo get user basic info but not contain authentication information
func GetUserInfo(name string) (*UserInfo, error) {
	u, err := getUserByName(name)
	if err != nil {
		return nil, err
	}
	u.RefreshToken = ""
	u.AccessToken = ""
	return u, nil
}

// KillOffLine will delete user token
func KillOffLine(name string) error {
	_, err := getUserByName(name)
	if err != nil {
		return err
	}
	resetRefreshToken(name, "")
	resetAccessToken(name, "")
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
		err := createUserTable()
		if err != nil {
			return err
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

// now not support user
func createUserTable() error {
	createStr := "create table " + Config.UserTableName + "(" +
		"ID              bigint(20) unsigned NOT NULL AUTO_INCREMENT," +
		"user_name       varchar(60) NOT NULL DEFAULT ''," +
		"user_pass       varchar(255) NOT NULL DEFAULT ''," +
		"user_nicename   varchar(50) NOT NULL DEFAULT ''," +
		"user_email      varchar(100) NOT NULL DEFAULT ''," +
		"user_registered datetime NOT NULL DEFAULT CURRENT_TIMESTAMP," +
		"refresh_token   varchar(255) NOT NULL DEFAULT ‘’," +
		"rtoken_created  datetime NOT NULL DEFAULT CURRENT_TIMESTAMP," +
		"access_token    varchar(255) NOT NULL DEFAULT ‘’," +
		"atoken_created  datetime NOT NULL DEFAULT CURRENT_TIMESTAMP," +
		"PRIMARY KEY (`ID`), " +
		"KEY `user_email` (`user_email`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8"
	_, err := db.Exec(createStr)
	if err != nil {
		return err
	}
	return nil
}
