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
	user := UserInfo{0, "sails", "xu", "sailsxu@qq.com", "twtpsu31", ""}
	err := UserRegister(user)
	if err != nil {
		fmt.Println(err)
	}
}

func TestLogin(t *testing.T) {

}
