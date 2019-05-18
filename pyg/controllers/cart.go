package controllers

import (
	"github.com/astaxie/beego"
	"github.com/gomodule/redigo/redis"
	"github.com/astaxie/beego/orm"
	"pyg/pyg/models"
)

type CartController struct{
	beego.Controller
}

// 加入购物车
func (this CartController)HandleAddCart(){
	// 获取数据
	id, err1 := this.GetInt("goodsId") // 商品 Id
	num, err2 := this.GetInt("num") // 数量
	// 定义一个返回　json 容器
	resp := make(map[string]interface{})
	// 以 json 方式传递数据给前端
	defer RespFunc(&this.Controller, resp)
	// 校验数据
	if err1 != nil || err2 != nil {
		resp["errno"] = 1
		beego.Info("执行到这里了１")
		resp["errmsg"] = "输入数据不完整"
		return
	}
	name := this.GetSession("name")
	if name == nil {
		resp["errno"] = 2
		beego.Info("执行到这里了２")
		resp["errmsg"] = "当前用户未登录，不能添加购物车"
		return
	}

	// 处理数据
	conn, err := redis.Dial("tcp", "127.0.0.1:6379") // 连接 redis 数据库

	if err != nil {
		resp["errno"] = 3
		resp["errmsg"] = "redis 连接数据库失败"
		return
	}
	defer conn.Close() // 用完之后，关闭 redis
	oldNum, _ := redis.Int(conn.Do("hget", "name_"+name.(string), id)) // 先读取
	_, err = conn.Do("hset", "name_" + name.(string), id, num + oldNum) // 再写入
	if err != nil {
		resp["errno"] = 4
		resp["errmsg"] = "数据库插入失败"
		return
	}
	resp["errno"] = 5
	resp["errmsg"] = "数据库插入成功"
}

// 我的购物车
func (this *CartController)ShowCart(){
	// 连接 redis
	conn, err := redis.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		beego.Error("redis 连接失败")
		this.Redirect("/index_sx", 302)
		return
	}
	defer conn.Close()
	name := this.GetSession("name") // 从 session 中获取当前用户
	result, err := redis.Ints(conn.Do("hgetall", "name_"+name.(string)))
	if err != nil {
		beego.Error("获取 redis 失败")
		this.Redirect("/index_sx", 302)
		return
	}
	// beego.Info(result)
	// 定义总容器
	var goods []map[string]interface{}
	o := orm.NewOrm()
	var goodsSku models.GoodsSKU
	allPrice := 0
	totalCount := 0
	// 遍历获取到的数据
	for i := 0; i < len(result); i+=2 {
		// 定义行容器
		temp := make(map[string]interface{})
		goodsSku.Id = result[i]
		o.Read(&goodsSku)
		temp["goodsSku"] = goodsSku
		temp["count"] = result[i+1]
		littlePrice := result[i+1] * goodsSku.Price
		temp["littlePrice"] = littlePrice // 小计
		allPrice = allPrice + littlePrice
		totalCount++
		// 将行容器添加到总容器中
		goods = append(goods, temp)
	}
	this.Data["allPrice"] = allPrice
	this.Data["totalCount"] = totalCount
	this.Data["goods"] = goods
	this.TplName = "cart.html"
}

// 更改购物车数量　－－　添加
func (this *CartController)HandleUpOrMinus(){
	// 获取数据
	count, err1 := this.GetInt("count")
	id, err2 := this.GetInt("goodsId")
	// 定义返回容器
	resp := make(map[string]interface{})
	defer RespFunc(&this.Controller, resp)
	// 校验数据
	if err1 != nil || err2 != nil {
		resp["errno"] = 1
		resp["errmsg"] = "数据不完整"
		return
	}
	// 处理数据
	conn, err := redis.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		resp["errno"] = 2
		resp["errmsg"] = "连接 redis 失败"
		return
	}
	defer conn.Close()
	name := this.GetSession("name")
	if name == nil {
		resp["errno"] = 4
		resp["errmsg"] = "当前无用户登录"
		return
	}
	_, err = conn.Do("hset", "name_"+name.(string), id, count)
	if err != nil {
		resp["errno"] = 3
		resp["errmsg"] = "插入失败"
		return
	}
	resp["errno"] = 5
	resp["errmsg"] = "OK"
}

// 购物车删除商品
func (this *CartController)HandleDeleteCart(){
	// 获取数据
	id, err := this.GetInt("goodsId")
	resp := make(map[string]interface{}) // 定义容器
	defer RespFunc(&this.Controller, resp)
	if err != nil {
		resp["errno"] = 1
		resp["errmsg"] = "数据传输不完整"
		return
	}
	// 操作数据库
	conn, err := redis.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		resp["errno"] = 2
		resp["errmsg"] = "数据路连接失败"
		return
	}
	defer conn.Close()
	name := this.GetSession("name")
	if name == nil {
		resp["errno"] = 3
		resp["errmsg"] = "当前无登录用户"
		return
	}
	_, err = conn.Do("hdel", "name_"+name.(string), id)
	if err != nil {
		resp["errno"] = 4
		resp["errmsg"] = "数据库删除失败"
		return
	}
	resp["errno"] = 5
	resp["errmsg"] = "OK"
}
