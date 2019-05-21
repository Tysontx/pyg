package controllers

import (
	"github.com/astaxie/beego"
	"regexp"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"fmt"
	"math/rand"
	"time"
	"encoding/json"
	"github.com/astaxie/beego/orm"
	"pyg/pyg/models"
	"github.com/astaxie/beego/utils"
)

type UserController struct {
	beego.Controller
}



type Message struct {
	Message string
	RequestId string
	BizId string
	Code string
}

// 返回 json 格式数据
func RespFunc(this *beego.Controller, resp map[string]interface{}) {
	// 把容器传给前端
	this.Data["json"] = resp
	// 指定传递方式
	this.ServeJSON()
}

// 展示注册页面
func (this *UserController)ShowRegister(){
	this.TplName = "register.html"
}

// 处理验证码业务
func (this *UserController)HandleSendMsg(){
	// 获取数据
	phone := this.GetString("phone")
	resp := make(map[string]interface{})
	defer RespFunc(&this.Controller, resp)
	// 校验数据
	if phone == "" {
		resp["errno"] = 1
		resp["errmsg"] = "获取电话号码失败"
		return
	}
	// 检查电话号码格式是否正确
	reg, _ := regexp.Compile(`^1[3-9][0-9]{9}$`)
	result := reg.FindString(phone)
	if result == "" {
		beego.Error("电话号码格式错误")
		// 给容器赋值
		resp["errno"] = 2
		resp["errmsg"] = "电话号码格式错误"
		return
	}
	// 发送短信 SDK 调用
	client, err := sdk.NewClientWithAccessKey("cn-hangzhou", "LTAIQTxIAEmpBa6z", "cP8zCcLoaLhivsvpYbps9GasL704Rh")
	if err != nil {
		beego.Error("短信 SDK 调用失败")
		resp["errno"] = 3
		resp["errmsg"] = "短信 SDK 调用失败"
		return
	}
	// 生成六位数随机数
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	vcode := fmt.Sprintf("%06d", rnd.Int31n(1000000))
	request := requests.NewCommonRequest()
	request.Method = "POST"
	request.Scheme = "https" // https | http
	request.Domain = "dysmsapi.aliyuncs.com"
	request.Version = "2017-05-25"
	request.ApiName = "SendSms"
	request.QueryParams["RegionId"] = "cn-hangzhou"
	request.QueryParams["PhoneNumbers"] = phone
	request.QueryParams["SignName"] = "pyg"
	request.QueryParams["TemplateCode"] = "SMS_165118250"
	request.QueryParams["TemplateParam"] = "{\"code\":"+vcode+"}"
	response, err := client.ProcessCommonRequest(request)
	if err != nil {
		beego.Error("短信发送失败4")
		resp["errno"] = 4
		resp["errmsg"] = "短信发送失败4"
		return
	}
	// json 数据解析
	// beego.Info(response)
	var message Message
	json.Unmarshal(response.GetHttpContentBytes(), &message)
	if message.Message != "OK" {
		beego.Error("短信发送失败6")
		resp["errno"] = 6
		resp["errmsg"] = message.Message
		return
	}
	resp["errno"] = 5
	resp["errmsg"] = "发送成功"
	resp["code"] = vcode
}

// 处理注册页面业务
func (this *UserController)HandleRegister(){
	// 获取数据
	phone := this.GetString("phone")
	pwd := this.GetString("password")
	rePwd := this.GetString("repassword")
	// 校验数据
	if phone == "" || pwd == "" || rePwd == "" {
		beego.Error("获取数据错误")
		this.Data["errmsg"] = "获取数据错误"
		this.TplName = "register.html"
		return
	}
	if pwd != rePwd {
		beego.Error("两次输入的密码不一致")
		this.Data["errmsg"] = "两次输入的密码不一致"
		this.TplName = "register.html"
		return
	}
	// 处理数据
	// orm 插入数据
	o := orm.NewOrm()
	var user models.User
	user.Name = phone
	user.Phone = phone
	user.Pwd = pwd
	o.Insert(&user)
	// 返回数据
	this.Ctx.SetCookie("userName", user.Name, 60*10)
	this.Redirect("/register-email", 302)
}

// 展示邮箱注册页面
func (this *UserController)ShowEmail(){
	this.TplName = "register-email.html"
}

// 处理邮箱激活业务
func (this *UserController)HandleEmail(){
	// 获取数据
	email := this.GetString("email")
	pwd := this.GetString("password")
	rePwd := this.GetString("repassword")
	// 校验数据
	if email == "" || pwd == "" || rePwd == "" {
		beego.Error("输入数据不完整")
		this.Data["errmsg"] = "输入数据不完整"
		this.TplName = "register-email.html"
		return
	}
	if pwd != rePwd {
		beego.Error("两次输入密码不一致")
		this.Data["errmsg"] = "两次输入密码不一致"
		this.TplName = "register-email.html"
		return
	}
	// 校验邮箱格式
	reg, _ := regexp.Compile(`^\w[\w\.-]*@[0-9a-z][0-9a-z-]*(\.[a-z]+)*\.[a-z]{2,6}$`)
	result := reg.FindString(email)
	if result == "" {
		beego.Error("邮箱格式不正确")
		this.Data["errmsg"] = "邮箱格式不正确"
		this.TplName = "register-email.html"
		return
	}
	// 处理数据（发送邮件）
	config := `{"username":"28645957@qq.com","password":"wezwocoytyllbghe","host":"smtp.qq.com","port":587}`
	emailReg := utils.NewEMail(config)
	emailReg.Subject = "品优购邮箱激活"
	emailReg.From = "28645957@qq.com"
	emailReg.To = []string{email}
	userName := this.Ctx.GetCookie("userName")
	emailReg.HTML = `<a href="http://192.168.181.156:8080/active?userName=`+userName+`"> 点击激活该用户</a>`
	//发送
	beego.Info("发送邮件")
	err := emailReg.Send()
	if err != nil {
		beego.Error("发送邮件失败")
		this.Data["errmsg"] = "发送邮件失败"
		return
	}
	// 更新字段
	o := orm.NewOrm()
	var user models.User
	user.Name = userName
	err = o.Read(&user, "Name")
	if err != nil {
		beego.Error("查询失败")
	}
	user.Email = email
	o.Update(&user)
	// 返回数据
	this.Ctx.WriteString("邮件已发送，请到目标邮件激活！")
}

// 邮箱激活
func (this *UserController)Active(){
	// 获取数据
	userName := this.GetString("userName")
	// 校验数据
	if userName == "" {
		beego.Error("用户名错误")
		this.Redirect("/register-email", 302)
		return
	}
	// 处理数据（更新字段）
	o := orm.NewOrm()
	var user models.User
	user.Name = userName
	err := o.Read(&user, "Name")
	if err != nil {
		beego.Error("用户名不存在")
		this.Redirect("/register-email", 302)
		return
	}
	user.Active = true
	o.Update(&user, "Active")
	// 返回数据
	this.Redirect("/login", 302)
}

// 展示登录页面
func (this *UserController)ShowLogin(){
	// 判断 cookie 是否有值
	name := this.Ctx.GetCookie("loginName")
	if name == "" {
		this.Data["checked"] = ""
	} else {
		this.Data["checked"] = "checked"
	}
	this.Data["name"] = name
	this.TplName = "login.html"
}

// 处理登录页面业务
func (this *UserController)HandleLogin(){
	// 获取数据
	username := this.GetString("name")
	pwd := this.GetString("pwd")
	// 校验数据
	if username == "" || pwd == "" {
		beego.Error("输入数据不完整")
		this.Data["errmsg"] = "输入数据不完整"
		this.TplName = "login.html"
		return
	}
	// 处理数据
	o := orm.NewOrm()
	var user models.User
	reg, _ := regexp.Compile(`^\w[\w\.-]*@[0-9a-z][0-9a-z-]*(\.[a-z]+)*\.[a-z]{2,6}$`)
	result := reg.FindString(username)
	if result != "" { // 邮箱用户名
		user.Email = username
		err := o.Read(&user, "Email")
		if err != nil {
			this.Data["errmsg"] = "邮箱未注册"
			this.TplName = "login.html"
			return
		}
		if user.Pwd != pwd {
			this.Data["errmsg"] = "密码错误"
			this.TplName = "login.html"
			return
		}
	} else { // 普通用户名
		user.Name = username
		err := o.Read(&user,"Name")
		if err != nil {
			this.Data["errmsg"] = "用户名未注册"
			this.TplName = "login.html"
			return
		}
		if user.Pwd != pwd {
			this.Data["errmsg"] = "密码错误"
			this.TplName = "login.html"
			return
		}
	}
	if user.Active == false {
		beego.Error("用户未激活")
		this.Data["errmsg"] = "用户未激活"
		this.TplName = "login.html"
		return
	}
	m1 := this.GetString("m1")
	if m1 == "2" { // 记住账号
		this.Ctx.SetCookie("loginName", user.Name, 60*60)
	} else {
		this.Ctx.SetCookie("loginName", user.Name, -1)
	}
	this.SetSession("name", user.Name)
	// 返回数据
	this.Redirect("/index", 302)
}

// 退出登录
func (this *UserController)Logout(){
	this.DelSession("name")
	this.Redirect("/login", 302)
}

// 展示用户中心页面
func (this *UserController)ShowUserCenterInfo(){
	o := orm.NewOrm()
	var user models.User
	// 获取当前用户
	name := this.GetSession("name")
	user.Name = name.(string)
	o.Read(&user, "Name")
	// 查询当前的默认地址
	var address models.Address
	o.QueryTable("Address").Filter("IsDefault", true).One(&address)

	this.Data["address"] = address
	this.Data["tplName"] = "个人信息"
	this.Layout = "userLayout.html"
	this.TplName = "user_center_info.html"
}

// 展示收货地址页面
func (this *UserController)ShowSite(){
	// 显示默认地址
	o := orm.NewOrm()
	var address models.Address
	name := this.GetSession("name") // 获取当前用户
	qs := o.QueryTable("Address").RelatedSel("User").Filter("User__Name", name.(string))
	qs.Filter("IsDefault", true).One(&address)
	qian := address.Phone[:3]
	hou := address.Phone[7:]
	address.Phone = qian + "****" + hou
	this.Data["address"] = address
	this.Data["tplName"] = "收货地址"
	this.Layout = "userLayout.html"
	this.TplName = "user_center_site.html"
}

// 添加收货地址业务
func (this *UserController)HandleSite(){
	// 获取数据
	receiver := this.GetString("receiver")
	addr := this.GetString("addr")
	postCode := this.GetString("postCode")
	phone := this.GetString("phone")
	// 校验数据
	if receiver == "" || addr == "" || postCode == "" || phone == "" {
		beego.Error("输入数据不完整")
		this.TplName = "user_center_site.html"
		return
	}
	// 处理数据
	o := orm.NewOrm()
	var address models.Address
	address.Receiver = receiver
	address.Addr = addr
	address.PostCode = postCode
	address.Phone = phone
	name := this.GetSession("name") // 获取用户
	var user models.User
	user.Name = name.(string)
	o.Read(&user, "Name")
	address.User = &user // 以对象的方式写入
	// 查询是否有默认地址。如果有，把默认地址修改为非默认。如果没有，直接插入默认地址
	// 查询当前用户是否有默认地址
	var oldAddress models.Address
	qs := o.QueryTable("Address").RelatedSel("User").Filter("User__Name", name.(string))
	err := qs.Filter("IsDefault", true).One(&oldAddress)
	if err == nil { // 查询到了默认地址
		oldAddress.IsDefault = false // 改为非默认
		o.Update(&oldAddress, "IsDefault")
	}
	address.IsDefault = true // 新插入的地址为默认地址
	_, err = o.Insert(&address)
	if err != nil {
		beego.Error("插入失败:",err)
		this.TplName = "user_center_site.html"
		return
	}
	// 返回数据
	this.Redirect("/user/site", 302)
}

