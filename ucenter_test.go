package ucenter

import (
	"fmt"
	"testing"
)

func TestInit(t *testing.T) {
	Config.MysqlConnStr = "root:@/ucenter?charset=utf8"
	Init()
}

func TestCreateUser(t *testing.T) {
	Config.MysqlConnStr = "root:@/ucenter?charset=utf8"
	Init()
	var user UserInfo
	user.UserName = "sails"
	user.Password = "twtpsu31"
	user.Email = "sailsxu@qq.com"
	err := UserRegister(user)
	if err != nil {
		fmt.Println(err)
	}
}

func TestLogin(t *testing.T) {
	Config.MysqlConnStr = "root:@/ucenter?charset=utf8"
	Init()
	name := "sails"
	pwd := "twtpsu31"
	refreshToken, accessToken, err := UserLogin(name, pwd)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("refresh_token:" + refreshToken)
	fmt.Println("access_token:" + accessToken)

	err = CheckAccessToken(name, accessToken)
	if err != nil {
		t.Fatal(err)
	}
	preAccessToken := accessToken

	accessToken, err = ResetAccessToken(name, refreshToken)
	if err != nil {
		t.Fatal(err)
	}

	// check by access token
	err = CheckAccessToken(name, accessToken)
	if err != nil {
		t.Fatal(err)
	}
	// check by pre access token
	err = CheckAccessToken(name, preAccessToken)
	if err != nil {
		t.Fatal(err)
	}
}
