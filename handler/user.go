package handler

import (
	"filestore-server/common"
	dblayer "filestore-server/db"
	"filestore-server/util"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

const (
	pwd_salt = "*#890"
)

//func SignupHandler(w http.ResponseWriter, r *http.Request) {
//	if r.Method == http.MethodGet {
//		data, err := os.ReadFile("static/view/signup.html")
//		if err != nil {
//			w.WriteHeader(http.StatusInternalServerError)
//			return
//		}
//		w.Write(data)
//		return
//	}
//r.ParseForm()
//username := r.Form.Get("username")
//passwd := r.Form.Get("password")
//if len(username) < 3 || len(passwd) < 5 {
//	w.Write([]byte("Invalid parameter"))
//	return
//}
//enc_passwd := util.Sha1([]byte(passwd + pwd_salt))
//suc := dblayer.UserSignup(username, enc_passwd)
//if suc {
//	w.Write([]byte("success"))
//} else {
//	w.Write([]byte("FAIlED"))
//
//}

// SignupHandler 处理注册get请求
func SignupHandler(c *gin.Context) {
	c.Redirect(http.StatusFound, "/static/view/signup.html")
}

// DoSignupHandler : 处理注册post请求
func DoSignupHandler(c *gin.Context) {
	username := c.Request.FormValue("username")
	passwd := c.Request.FormValue("password")

	// 校验用户名密码
	if len(username) < 3 || len(passwd) < 5 {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "请求参数无效",
			"code": common.StatusParamInvalid,
		})
		return
	}

	// 对密码进行加盐及取Sha1值加密
	encPasswd := util.Sha1([]byte(passwd + pwd_salt))
	// 将用户信息注册到用户表中
	suc := dblayer.UserSignup(username, encPasswd)
	if suc {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "注册成功",
			"code": common.StatusOK,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "注册失败",
			"code": common.StatusRegisterFailed,
		})
	}
}

// SignInHandler : 响应登录页面
func SignInHandler(c *gin.Context) {
	c.Redirect(http.StatusFound, "/static/view/signin.html")
}

// DoSignInHandler : 处理登录post请求
func DoSignInHandler(c *gin.Context) {
	username := c.Request.FormValue("username")
	password := c.Request.FormValue("password")

	encPasswd := util.Sha1([]byte(password + pwd_salt))

	// 1. 校验用户名及密码
	pwdChecked := dblayer.UserSignin(username, encPasswd)
	if !pwdChecked {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "登录失败",
			"code": common.StatusLoginFailed,
		})
		return
	}

	// 2. 生成访问凭证(token)
	token := GenToken(username)
	upRes := dblayer.UpdateToken(username, token)
	if !upRes {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "登录失败",
			"code": common.StatusLoginFailed,
		})
		return
	}

	// 3. 登录成功，返回用户信息
	resp := util.RespMsg{
		Code: int(common.StatusOK),
		Msg:  "登录成功",
		Data: struct {
			Location string
			Username string
			Token    string
		}{
			Location: "http://" + c.Request.Host + "/static/view/home.html",
			Username: username,
			Token:    token,
		},
	}

	c.Data(http.StatusOK, "application/json", resp.JSONBytes())
	
}

//// SignInHandler：登录接口
//func SignInHandler(w http.ResponseWriter, r *http.Request) {
//	if r.Method == http.MethodGet {
//		data, err := os.ReadFile("static/view/signin.html")
//		if err != nil {
//			w.WriteHeader(http.StatusInternalServerError)
//			return
//		}
//		w.Write(data)
//		return
//	}
//	err := r.ParseForm()
//	if err != nil {
//		return
//	}
//	username := r.Form.Get("username")
//	password := r.Form.Get("password")
//	encPasswd := util.Sha1([]byte(password + pwd_salt))
//	//1.校验用户名及密码
//	pwdChecked := dblayer.UserSignin(username, encPasswd)
//	if !pwdChecked {
//		resp := util.RespMsg{
//			Code: -1, // 错误码
//			Msg:  "用户名或密码错误",
//			Data: nil, // 没有额外的数据
//		}
//		w.Write(resp.JSONBytes()) // 返回 JSON 格式的错误信息给前端success处理
//		return
//	}
//
//	//2.生成访问凭证（token)
//	token := GenToken(username)
//	upRes := dblayer.UpdateToken(username, token)
//	if !upRes {
//		w.Write([]byte("FAILED"))
//		return
//	}
//	//3.登录成功后重定向到首页
//	//http.Redirect(w, r, "http://"+r.Host+"/static/view/home.html", http.StatusFound)
//	resp := util.RespMsg{
//		Code: 0,
//		Msg:  "OK",
//		Data: struct {
//			Location string
//			Username string
//			Token    string
//		}{
//			Location: "http://" + r.Host + "/static/view/home.html",
//			Username: username,
//			Token:    token,
//		},
//	}
//	w.Write(resp.JSONBytes())
//}

// // UserInfoHandler ： 查询用户信息
//
//	func UserInfoHandler(w http.ResponseWriter, r *http.Request) {
//		// 1. 解析请求参数
//		r.ParseForm()
//		username := r.Form.Get("username")
//		//token := r.Form.Get("token")
//		//
//		//// 2. 验证token是否有效
//		////isValidToken := IsTokenValid(token)
//		////if !isValidToken {
//		////	w.WriteHeader(http.StatusForbidden)
//		////	return
//		////}
//
//		// 3. 查询用户信息
//		user, err := dblayer.GetUserInfo(username)
//		if err != nil {
//			w.WriteHeader(http.StatusForbidden)
//			return
//		}
//
//		// 4. 组装并且响应用户数据
//		resp := util.RespMsg{
//			Code: 0,
//			Msg:  "OK",
//			Data: user,
//		}
//		w.Write(resp.JSONBytes())
//	}
//
// UserInfoHandler ： 查询用户信息
func UserInfoHandler(c *gin.Context) {
	// 1. 解析请求参数
	username := c.Request.FormValue("username")
	//	token := c.Request.FormValue("token")

	// 2. 查询用户信息
	user, err := dblayer.GetUserInfo(username)
	if err != nil {
		c.JSON(http.StatusForbidden,
			gin.H{})
		return
	}

	// 3. 组装并且响应用户数据
	resp := util.RespMsg{
		Code: 0,
		Msg:  "OK",
		Data: user,
	}
	c.Data(http.StatusOK, "application/json", resp.JSONBytes())
}

// UserExistsHandler ： 查询用户是否存在
func UserExistsHandler(c *gin.Context) {
	// 1. 解析请求参数
	username := c.Request.FormValue("username")

	// 3. 查询用户信息
	exists, err := dblayer.UserExist(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError,
			gin.H{
				"code": common.StatusServerError,
				"msg":  "server error",
			})
	} else {
		c.JSON(http.StatusOK,
			gin.H{
				"code":   common.StatusOK,
				"msg":    "ok",
				"exists": exists,
			})
	}
}

func IsTokenValid(token string) bool {
	if len(token) != 40 {
		return false
	}
	// TODO: 判断token的时效性，是否过期
	// TODO: 从数据库表tbl_user_token查询username对应的token信息
	// TODO: 对比两个token是否一致
	return true
}

func GenToken(username string) string {
	//40位字符md5(username+timestamp+token_salt)+timestamp[:8]
	ts := fmt.Sprintf("%x", time.Now().Unix())
	tokenPrefix := util.MD5([]byte(username + ts + "_tokensalt"))
	return tokenPrefix + ts[:8]
}
