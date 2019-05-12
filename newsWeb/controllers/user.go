package controllers

import (
	"github.com/astaxie/beego"
	"newsWeb/models"
	"github.com/astaxie/beego/orm"
	"encoding/base64"
)

type UserController struct {
	beego.Controller
}

func (this *UserController)ShowRegister() {
	this.TplName = "register.html"
}

func (this *UserController)HandleRegister() {
	username := this.GetString("userName")
	pwd := this.GetString("password")
	if username == "" || pwd == "" {
		beego.Error("输入信息不完整")
		this.TplName = "register.html"
		return
	}
	o := orm.NewOrm()
	var User models.User
	User.Name = username
	User.Pwd = pwd
	id, err := o.Insert(&User)
	if err != nil {
		beego.Error("用户注册失败")
		this.TplName = "register.html"
		return
	}
	beego.Info(id)
	this.Redirect("/login", 302)
}

func (this *UserController)ShowLogin(){
	// 获取 cookie 数据，如果查到了，说明上一次记住了用户名，不然的话，不记住用户名
	userName := this.Ctx.GetCookie("userName")
	// 解密
	dec, _ := base64.StdEncoding.DecodeString(userName)
	if userName != "" {
		this.Data["userName"] = string(dec)
		this.Data["checked"] = "checked"
	} else {
		this.Data["userName"] = ""
		this.Data["checked"] = ""
	}
	this.TplName = "login.html"
}

func (this *UserController)HandleLogin(){
	userName := this.GetString("userName")
	pwd := this.GetString("password")
	if userName == "" || pwd == "" {
		beego.Error("输入数据不完整")
		this.TplName = "login.html"
		return
	}
	o := orm.NewOrm()
	var user models.User
	user.Name = userName
	err := o.Read(&user, "Name")
	if err != nil {
		beego.Error("用户名不正确")
		this.TplName = "login.html"
		return
	}
	if user.Pwd != pwd {
		beego.Error("密码不正确")
		this.TplName = "login.html"
		return
	}
	// 实现记住用户名功能 上一次登录成功以后，点击记住用户名，下一次登录的时候默认显示用户名
	remember := this.GetString("remember")
	// 给 userName 加密
	enc := base64.StdEncoding.EncodeToString([]byte(userName))
	if remember == "on" { // 表示“记住用户名”选项已勾选
		// 设置 cookie
		this.Ctx.SetCookie("userName", enc, 60)
	} else {
		this.Ctx.SetCookie("userName", userName, -1)
	}
	// session 存储
	this.SetSession("userName", userName)
	this.Redirect("/article/index", 302)
}

// 退出登录
func (this *UserController)Logout(){
	this.DelSession("userName")
	this.Redirect("/login", 302)
}
