# ucenter
ucenter 提供了一套用户认证管理中心，可以很方便的集成到现有的golang web框架中。

## 配置和使用方式
ucenter的验证流程: 当用户登录后，会返回AccessToken,RefreshToken和AccessToken的过期时间，校验时通过AccessToken进行，在AccessToken过期之前，需要用RefreshToken生成一个新的AccessToken；老的AccessToken为了接口的过渡会存在一段时间再失效。

### 配置mysql（必须）
ucenter由于要连接mysql创建用户数据表，所以要配置mysql的连接字符串:
```
Config.MysqlConnStr = "root:@/ucenter?charset=utf8"
```
### 配置redis（可选）
ucenter自带了一个简单的cache，但是如果会运行多个ucenter实例，就不能用自带的cache了，ucenter提供了redis作用统一的token和session的cache的支持
```
Config.RedisConnStr = ":6379"
```

### 使用
+ 初始化
用于初始化一数据表和cache
```
Init()
```
+ 用户注册:
```
user := UserInfo{UserName: "sails", Password: "twtpsu31",
		Email: "sailsxu@qq.com"}
err := UserRegister(user)
```
+ 登录:
```
loginRet, err := UserLogin(name, pwd)
```
+ 用户验证：
```
err := CheckAccessToken(name, accssToken)
```
+ 更新AccessToken
```
accessToken, err := ResetAccessToken(name, RefreshToken)
```
+ 退出:
```
err := KillOffLine(name)
```


## ucenter 将实现的特性
### 用户管理方面
+ 加强用户管理

### 集成oauth2.0支持
+ QQ
+ 微博
