package routers

import (
	"newsWeb/controllers"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
)

func init() {
	// 路由过滤器
	beego.InsertFilter("/article/*", beego.BeforeExec, filterFunc)

    beego.Router("/", &controllers.MainController{})
    // 给注册页面设置路由
    beego.Router("/register", &controllers.UserController{}, "get:ShowRegister;post:HandleRegister")
    // 给登录页面设置路由
    beego.Router("/login", &controllers.UserController{}, "get:ShowLogin;post:HandleLogin")
    // 给主页设置路由
    beego.Router("/article/index", &controllers.ArticleController{}, "get,post:ShowIndex")
    // 给添加文章设置路由
    beego.Router("/article/addArticle", &controllers.ArticleController{}, "get:ShowAddArticle;post:HandleAddArticle")
	// 给查看详情页设置路由
	beego.Router("/article/content", &controllers.ArticleController{}, "get:ShowContent")
	// 给编辑页面设置路由
	beego.Router("/article/update", &controllers.ArticleController{}, "get:ShowUpdate;post:HandleUpdate")
	// 设置删除文章路由
	beego.Router("/article/delete", &controllers.ArticleController{}, "get:HandleDelete")
	// 添加分类页面路由
	beego.Router("/article/addType", &controllers.ArticleController{}, "get:ShowAddType;post:HandleAddType")
	// 添加退出按钮路由
	beego.Router("/article/logout", &controllers.UserController{}, "get:Logout")
	// 设置删除分类
	beego.Router("/article/deleteType", &controllers.ArticleController{}, "get:DeleteType")
}

// 路由过滤器回调函数
func filterFunc(ctx *context.Context) {
	userName := ctx.Input.Session("userName")
	if userName == nil {
		ctx.Redirect(302, "/login")
		return
	}
}
