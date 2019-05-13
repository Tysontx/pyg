package routers

import (
	"pyg/pyg/controllers"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
)

func init() {
	// 路由过滤器
	beego.InsertFilter("/user/*", beego.BeforeExec, guolvFunc)
    beego.Router("/", &controllers.MainController{})
    // 注册页面
    beego.Router("/register", &controllers.UserController{},"get:ShowRegister;post:HandleRegister")
    // 发送验证码
    beego.Router("/sendMsg", &controllers.UserController{}, "post:HandleSendMsg")
    // 邮箱注册页面
    beego.Router("/register-email", &controllers.UserController{}, "get:ShowEmail;post:HandleEmail")
    // 邮箱激活
    beego.Router("/active", &controllers.UserController{}, "get:Active")
    // 登录页面
    beego.Router("/login", &controllers.UserController{}, "get:ShowLogin;post:HandleLogin")
    // 主页
    beego.Router("/index", &controllers.GoodsController{}, "get:ShowIndex")
    // 退出登录
    beego.Router("/user/logout", &controllers.UserController{}, "get:Logout")
    // 展示用户中心页
    beego.Router("/user/userCenterInfo", &controllers.UserController{}, "get:ShowUserCenterInfo")
	// 收货地址页面
	beego.Router("/user/site", &controllers.UserController{}, "get:ShowSite;post:HandleSite")
}

func guolvFunc(ctx *context.Context){
	name := ctx.Input.Session("name")
	if name == nil {
		ctx.Redirect(302, "/login")
		return
	}
}
