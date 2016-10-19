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
	user := UserInfo{UserName: "sails", Password: "twtpsu31",
		Email: "sailsxu@qq.com"}
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
	loginRet, err := UserLogin(name, pwd)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("refresh_token:" + loginRet.RefreshToken)
	fmt.Println("access_token:" + loginRet.AccessToken)

	err = CheckAccessToken(name, loginRet.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	preAccessToken := loginRet.AccessToken

	accessToken, err := ResetAccessToken(name, loginRet.RefreshToken)
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
