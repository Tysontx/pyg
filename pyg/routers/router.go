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
	// 生鲜模块
	beego.Router("/index_sx", &controllers.GoodsController{}, "get:ShowIndexSx")
	// 商品详情页
	beego.Router("/goodsDetail", &controllers.GoodsController{}, "get:ShowGoodsDetail")
	// 商品列表页面
	beego.Router("/goodsType", &controllers.GoodsController{}, "get:ShowList")
	// 搜索商品
	beego.Router("/search", &controllers.GoodsController{},"post:HandleSearch")
	// 添加购物车
	beego.Router("/addCart", &controllers.CartController{}, "post:HandleAddCart")
	// 我的购物车
	beego.Router("/user/showCart", &controllers.CartController{}, "get:ShowCart")
	// 更改购物车数量　－－　加一或减一
	beego.Router("/upOrMinus", &controllers.CartController{}, "post:HandleUpOrMinus")
	// 购物车删除商品
	beego.Router("/deleteCart", &controllers.CartController{}, "post:HandleDeleteCart")
	// 点击“去结算”业务
	beego.Router("/user/addOrder", &controllers.OrderController{}, "post:ShowOrder")
	// 提交订单
	beego.Router("/pushOrder", &controllers.OrderController{}, "post:HandlePushOrder")
	// 展示用户中心订单页
	beego.Router("/user/userOrder", &controllers.OrderController{}, "get:ShowUserOrder")
	// 去支付
	beego.Router("/pay", &controllers.OrderController{},"get:Pay")
	// 支付成功
	beego.Router("/payOk", &controllers.OrderController{}, "get:PayOk")
}

func guolvFunc(ctx *context.Context){
	name := ctx.Input.Session("name")
	if name == nil {
		ctx.Redirect(302, "/login")
		return
	}
}
