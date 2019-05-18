package controllers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"pyg/pyg/models"
	"strconv"
	"github.com/gomodule/redigo/redis"
	"time"
	"strings"
)

type OrderController struct {
	beego.Controller
}

// 去结算 业务
func (this *OrderController)ShowOrder(){
	// 获取数据
	goodsIds := this.GetStrings("checkGoods")
	beego.Info(goodsIds)
	// 校验数据
	if len(goodsIds) == 0 {
		beego.Error("传输数据不完整")
		this.Redirect("/user/showCart", 302)
		return
	}
	// 获取当前用户
	name := this.GetSession("name")
	if name == nil {
		beego.Error("当前无用户登录")
		this.Redirect("/login", 302)
		return
	}
	// 获取用户的地址
	o := orm.NewOrm()
	var address []models.Address
	o.QueryTable("Address").RelatedSel("User").Filter("User__Name", name.(string)).All(&address)

	conn, _ := redis.Dial("tcp", "127.0.0.1:6379")
	// 获取商品、获取总价、总件数
	var goods []map[string]interface{} // 定义大容器
	var totalPrice, totalCount int // 定义总价格、总件数
	// 获取商品
	for _, v := range goodsIds {
		temp := make(map[string]interface{}) // 定义行容器
		id, _ := strconv.Atoi(v)
		// 选中的商品对象
		var goodsSku models.GoodsSKU
		goodsSku.Id = id
		o.Read(&goodsSku)
		// 获取商品数量
		count, _ := redis.Int(conn.Do("hget", "name_"+name.(string),id))
		// 计算小计
		littlePrice := goodsSku.Price * count
		temp["goodsSku"] = goodsSku
		temp["count"] = count
		temp["littlePrice"] = littlePrice
		totalPrice += littlePrice
		totalCount += 1
		goods = append(goods,temp) // 行容器添加到总容器中
	}

	this.Data["goods"] = goods
	this.Data["totalPrice"] = totalPrice
	this.Data["totalCount"] = totalCount
	this.Data["truePrice"] = totalPrice + 10
	this.Data["address"] = address
	this.Data["goodsIds"] = goodsIds
	this.TplName = "place_order.html"
}

// 提交订单 业务
func (this *OrderController)HandlePushOrder(){
	// 获取数据
	addrId, err1 := this.GetInt("addrId")
	payId, err2 := this.GetInt("payId")
	goodsIds := this.GetString("goodsIds")
	totalCount, err3 := this.GetInt("totalCount")
	totalPrice, err4 := this.GetInt("totalPrice")
	// 校验数据
	resp := make(map[string]interface{}) // 创建 json 数据容器
	defer RespFunc(&this.Controller, resp)
	if err1 != nil || err2 != nil || goodsIds == "" || err3 != nil || err4 != nil {
		resp["errno"] = 1
		resp["errmsg"] = "输入数据不完整"
		return
	}
	// 处理数据
	name := this.GetSession("name")
	o := orm.NewOrm()
	var user models.User // 创建 user 对象
	user.Name = name.(string)
	o.Read(&user)

	var address models.Address // 创建 address 对象
	address.Id = addrId
	o.Read(&address)

	var orderInfo models.OrderInfo // 创建订单表对象
	orderInfo.OrderId = time.Now().Format("20060102150405"+strconv.Itoa(user.Id)) // 根据时间和用户 id 生成订单号
	orderInfo.User = &user //　赋值对象
	orderInfo.Address = &address // 赋值对象
	orderInfo.PayMethod = payId
	orderInfo.TotalCount = totalCount
	orderInfo.TotalPrice = totalPrice
	orderInfo.TransitPrice = 10
	_, err := o.Insert(&orderInfo)
	if err != nil {
		resp["errno"] = 2
		resp["errmsg"] = "订单表插入失败"
		return
	}

	conn, err := redis.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		resp["errno"] = 4
		resp["errmsg"] = "redis 连接失败"
		return
	}
	defer conn.Close()
	goodsSlice := strings.Split(goodsIds[1:len(goodsIds)-1], " ")
	for _,v := range goodsSlice {
		id, err := strconv.Atoi(v)
		var goodsSku models.GoodsSKU
		goodsSku.Id = id
		o.Read(&goodsSku)
		if err != nil {
			resp["errno"] = 3
			resp["errmsg"] = "商品 id　获取失败"
			return
		}
		count, err := redis.Int(conn.Do("hget", "name_"+name.(string),id))
		if err != nil {
			resp["errno"] = 6
			resp["errmsg"] = "获取商品数量失败"
			return
		}

		littlePrice := goodsSku.Price * count // 获取小计

		var orderGoods models.OrderGoods // 创建订单商品表对象
		orderGoods.OrderInfo = &orderInfo
		orderGoods.GoodsSKU = &goodsSku
		orderGoods.Count = count

		orderGoods.Price = littlePrice
		// 插入之前需要判断库存数量
		if goodsSku.Stock < count {
			resp["errno"] = 8
			resp["errmsg"] = "库存不足"
			return
		}
		// 更新销量和库存
		goodsSku.Stock -= count
		goodsSku.Sales += count
		o.Update(&goodsSku)
		_, err = o.Insert(&orderGoods)
		if err != nil {
			resp["errno"] = 7
			resp["errmsg"] = "订单商品表插入失败"
			return
		}
		// 插入成功后，清空 redis
		_, err = conn.Do("hdel", "name_"+name.(string), id)
		beego.Error(err)
	}
	// 返回数据
	resp["errno"] = 5
	resp["errmsg"] = "OK"
}
