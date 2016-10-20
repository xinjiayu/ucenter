package ucenter

import (
	"crypto/md5"
	"fmt"
)

func getUserByName(name string) (*UserInfo, error) {
	sql := "select ID, user_name, user_pass, user_nicename, user_email," +
		" user_registered " +
		" from " + Config.UserTableName + " where user_name = ?"
	rows, err := db.Query(sql, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {

		var u UserInfo
		if err = rows.Scan(&u.ID, &u.Nickname, &u.Password,
			&u.Nickname, &u.Email, &u.Registered); err == nil {
			return &u, nil
		}
		fmt.Println(err)
	}
	return nil, ErrUserNotExist
}

func createUser(user UserInfo) error {
	password := md5.Sum([]byte(user.Password))
	passwordstr := fmt.Sprintf("%x", password)
	sql := "insert into " + Config.UserTableName + "(user_name, " +
		"user_pass, user_nicename, user_email, user_registered ) " +
		"values(?, ?, ?, ?, now())"
	_, err := db.Exec(sql, user.UserName, passwordstr, user.Nickname,
		user.Email)
	return err
}
