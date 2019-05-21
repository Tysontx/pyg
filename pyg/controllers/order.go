package controllers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"pyg/pyg/models"
	"strconv"
	"github.com/gomodule/redigo/redis"
	"time"
	"strings"
	"github.com/smartwalle/alipay"
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
	o.Read(&user, "Name")

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
	o.Begin() // 开始事物
	_, err := o.Insert(&orderInfo)
	if err != nil {
		resp["errno"] = 2
		resp["errmsg"] = "订单表插入失败"
		o.Rollback() // 事物回滚
		return
	}

	conn, err := redis.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		resp["errno"] = 4
		resp["errmsg"] = "redis 连接失败"
		o.Rollback() // 事物回滚
		return
	}
	defer conn.Close()
	goodsSlice := strings.Split(goodsIds[1:len(goodsIds)-1], " ")
	for _,v := range goodsSlice {
		id, err := strconv.Atoi(v)
		var goodsSku models.GoodsSKU
		goodsSku.Id = id
		o.Read(&goodsSku)

		oldStock := goodsSku.Stock
		beego.Info("原始库存等于：", oldStock)

		if err != nil {
			resp["errno"] = 3
			resp["errmsg"] = "商品 id　获取失败"
			o.Rollback() // 事物回滚
			return
		}
		count, err := redis.Int(conn.Do("hget", "name_"+name.(string),id))
		if err != nil {
			resp["errno"] = 6
			resp["errmsg"] = "获取商品数量失败"
			o.Rollback() // 事物回滚
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
			o.Rollback() // 事物回滚
			return
		}
		// 更新销量和库存
		// goodsSku.Stock -= count
		// goodsSku.Sales += count
		// time.Sleep(time.Second * 5) // 手动加延迟
		o.Read(&goodsSku)

		qs := o.QueryTable("GoodsSKU").Filter("Id",id).Filter("Stock", oldStock)
		_, err = qs.Update(orm.Params{"Stock":goodsSku.Stock-count, "Sales":goodsSku.Stock+count})
		if err != nil {
			resp["errno"] = 11
			resp["errmsg"] = "购买失败，请重新排队！"
			o.Rollback()
			return
		}
		beego.Info("当前库存：", goodsSku.Stock)
		_, err = o.Insert(&orderGoods)
		if err != nil {
			resp["errno"] = 7
			resp["errmsg"] = "订单商品表插入失败"
			o.Rollback() // 事物回滚
			return
		}
		// 插入成功后，清空 redis
		_, err = conn.Do("hdel", "name_"+name.(string), id)
		if err != nil {
			resp["errno"] = 9
			resp["errmsg"] = "redis 清空失败"
			o.Rollback() // 事物回滚
		}
	}
	// 返回数据
	resp["errno"] = 5
	resp["errmsg"] = "OK"
	o.Commit() // 提交事物
}

// 用户中心订单页
func (this *OrderController)ShowUserOrder(){
	// 获取当前用户
	name := this.GetSession("name")
	o := orm.NewOrm()
	// 查询当前用户的所有订单
	var orderInfos []models.OrderInfo
	o.QueryTable("OrderInfo").RelatedSel("User").Filter("User__Name", name.(string)).OrderBy("-Time").All(&orderInfos)
	var orders []map[string]interface{} // 定义总容器
	for _, v := range orderInfos {
		temp := make(map[string]interface{}) // 定义行容器
		var orderGoods []models.OrderGoods
		// 获取当前订单的所有商品
		o.QueryTable("OrderGoods").RelatedSel("OrderInfo", "GoodsSKU").Filter("OrderInfo__Id", v.Id).All(&orderGoods)
		// 添加到行容器中
		temp["orderGoods"] = orderGoods
		temp["orderInfo"] = v
		// 将行容器追加到总容器
		orders = append(orders, temp)
	}
	this.Data["orders"] = orders
	this.Data["tplName"] = "全部订单"
	this.Layout = "userLayout.html"
	this.TplName = "user_center_order.html"
}

//　支付业务
func (this *OrderController)Pay(){
	// 获取订单 Id
	orderId, err := this.GetInt("orderId")
	// 校验数据
	if err != nil {
		beego.Error("获取支付订单 Id 失败")
		this.Redirect("/user/userOrder", 302)
		return
	}
	// beego.Info(orderId)
	// 处理数据
	o := orm.NewOrm()
	var orderInfo models.OrderInfo
	orderInfo.Id = orderId
	o.Read(&orderInfo)

	// 支付
	publicKey := `MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA1RBCKYygzzsqDMZy9Lj3
				ZEUP5pEjIZjrO0AuiB2WzcploI3m4hgRvpMZDvmpWLm0YOV/nnZ+GknUWGT3RBY6
				iCNEvWkSvKSyhmEBNqRyJfPawkkqTAhlORmD3BRIm0qifoCYrzqLd2JYelcYVBya
				EfNKYiUps3G0UkuC6diVBA6zWmhg/ha+0V2yUTlqXPeOxh9hiM12tfdCNhSD4i91
				24WT2CUFZaSzE1q0mZYRKGngxCQcGWpksqI5xGkd52FpBt9tkkXc3s/XC+/EAXw+
				qtGFNPZ4pW988IcZQhtAbQoYcdyASNAcrPTTAGgFwp5laYZ+EWLH95hYx8xKoVL7
				pQIDAQAB`
	privateKey := `MIIEogIBAAKCAQEA1RBCKYygzzsqDMZy9Lj3ZEUP5pEjIZjrO0AuiB2WzcploI3m
				4hgRvpMZDvmpWLm0YOV/nnZ+GknUWGT3RBY6iCNEvWkSvKSyhmEBNqRyJfPawkkq
				TAhlORmD3BRIm0qifoCYrzqLd2JYelcYVByaEfNKYiUps3G0UkuC6diVBA6zWmhg
				/ha+0V2yUTlqXPeOxh9hiM12tfdCNhSD4i9124WT2CUFZaSzE1q0mZYRKGngxCQc
				GWpksqI5xGkd52FpBt9tkkXc3s/XC+/EAXw+qtGFNPZ4pW988IcZQhtAbQoYcdyA
				SNAcrPTTAGgFwp5laYZ+EWLH95hYx8xKoVL7pQIDAQABAoIBAGE4Gfh7gqUMihNq
				OeoQvFG0cZzzfORHso5GqvTRC467W8P2+/MOqIoc9MIwiWVC11ufXKwhxUiZh5sN
				9wXKXsrfzO3gk/wf6pYGjVcxkiRfMOKWIAaxjf6P9ermFntFgv/WDdVnEVxYM6cf
				NqqqomKucLJ34p9OsskaS5IIkXZXqUtp8Zta2kBeZNGtMFgrvKp3hCFtNJTdUG0h
				LVDRd2n4VriivyxL+4FzhqpeWpx49BbmOXS5bNTwFtperEcbS/NXeFpfGh/qmlhl
				AWWWbPiNCuntQMafaIqwPrdxlt2e925VA0DL0e9auSJrhCSGYJRsFj7hb2xp+E40
				IqB55gECgYEA+RTSbkm4AZ6IRkG2hbGacqrAFjLgLWlZukxLPj7Bh4dMtIUxMj3B
				B+IyWXkAp2aVDQdyfG3EmFm6jIlgYF1u+KCmYc812r/+vatU9Ydm6spk4Mu9kGp1
				cNSxRcJVr4lCALLT38qGlqNSyiS9sFnei5uYQP4x9ijVF/ifoxhKkIkCgYEA2vtR
				wv46ZMF772pxewSz6I1vDtrHKCJpb7eA/6D5b0Ik3ltdFtP3mYtuJFtNL7m4MfmO
				1HKkjVq2RBV05B2JoSxBKgRVjSet8lCVp5+L7FW5LgIDQnYlnIcCQu2e0UXwuucP
				UzELCofP/AHXNEL9/gJrVt5kD7y9L54rRxFZcz0CgYBS6CFa4GLE9zW43OqZ+ZHF
				FRy2xtxjgSuCnR52a4ETUW+wrpy/clqr+xhzO5mCHt0B5zauQAMuCr/TQ2625KKp
				Ux/OcqAkXb+29i5jQ1x4TkHhqS9BwI2yrrkK1TKcKP21KdDoLos53McTzcLtzhwL
				MBEvoOyUWOcFAZZxPQaksQKBgF0NmMPclmHEWm71c32MFQtINp5AV4r1fIptlxKJ
				jBU8LUCT4G3X6wpDVq16YsVaDSynWItsoAI1PuiVmZNp/dcQYCyDpPsTlnY2yjFt
				ud7W2pbzYgE3BWqLcGmSYf+Z0d8KWtfGKmPyLG5xNcrOgPIUgxpp7GlHkbkPZGKR
				u8odAoGAQV1RHpGAv2llE14hC9mOoMvWeu8OniIAiqFdwTNo+aeS4DWI2eT7xj1B
				MpiBDVh63NZjpkyAWE9R0VabQl76WinrOoyX1wJppTRi8RvsUGl3kyz10ZgJ7PaB
				JCiXJ368m73i5+KiO5urRfCasErsiMQg4hUx9KPxyhplIRF1zNA=`
	client := alipay.New("2016093000634410", publicKey, privateKey,false)
	var p = alipay.TradePagePay{}
	p.NotifyURL = "http://192.168.181.156:8080/payOk?orderId=" + strconv.Itoa(orderId)
	p.ReturnURL = "http://192.168.181.156:8080/payOk?orderId=" + strconv.Itoa(orderId)
	p.Subject = "品优购"
	p.OutTradeNo = orderInfo.OrderId
	p.TotalAmount = strconv.Itoa(orderInfo.TotalPrice)
	p.ProductCode = "FAST_INSTANT_TRADE_PAY"
	url, err := client.TradePagePay(p)
	if err != nil {
		beego.Error("支付失败")
		return
	}
	payUrl := url.String()
	// beego.Info(payUrl)
	this.Redirect(payUrl, 302)
}

// 支付成功，返回页面业务处理
func (this *OrderController)PayOk(){
	// 获取支付订单 id
	orderId := this.GetString("orderId")
	if orderId == "" {
		beego.Error("获取订单 id　失败")
		return
	}
	// 更新订单状态
	o := orm.NewOrm()
	var orderInfo models.OrderInfo
	id, _ := strconv.Atoi(orderId)
	orderInfo.Id = id
	o.Read(&orderInfo)
	orderInfo.Orderstatus = 1
	o.Update(&orderInfo, "Orderstatus")

	this.Redirect("/user/userOrder", 302)
}
