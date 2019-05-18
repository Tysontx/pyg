package controllers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"pyg/pyg/models"
	"math"
)

type GoodsController struct{
	beego.Controller
}

// 显示主页
func (this *GoodsController)ShowIndex(){
	name := this.GetSession("name")
	if name != nil {
		this.Data["name"] = name.(string)
	} else {
		this.Data["name"] = ""
	}

	// 获取类型信息并传递给前端
	// 获取一级菜单
	o := orm.NewOrm()
	// 接受对象
	var oneClass []models.TpshopCategory
	// 查询
	o.QueryTable("TpshopCategory").Filter("Pid", 0).All(&oneClass)

	// 总容器
	var types []map[string]interface{}

	// 获取二级菜单
	for _, v := range oneClass {
		// 行容器
		var t = make(map[string]interface{})
		var secondClass []models.TpshopCategory
		o.QueryTable("TpshopCategory").Filter("Pid", v.Id).All(&secondClass)
		t["t1"] = v // 一级菜单对象
		t["t2"] = secondClass // 二级菜单集合
		// 把行容器加载到总容器中
		types = append(types, t)
	}

	// 获取三级菜单
	for _, v1 := range types {
		// 定义二级容器
		var erji []map[string]interface{}
		// 循环获取二级菜单
		for _, v2 := range v1["t2"].([]models.TpshopCategory) {
			t := make(map[string]interface{})
			var thirdClass []models.TpshopCategory
			// 查询三级菜单
			o.QueryTable("TpshopCategory").Filter("Pid", v2.Id).All(&thirdClass)
			t["t22"] = v2 // 二级菜单对象
			t["t23"] = thirdClass // 三级菜单集合
			// 加载到二级容器中
			erji = append(erji, t)
		}
		// 把二级容器放到总容器中
		v1["t3"] = erji
	}

	this.Data["types"] = types
	this.TplName = "index.html"
}

// 生鲜模块
func (this *GoodsController)ShowIndexSx(){
	// 获取生鲜首页内容
	// 1 获取商品类型
	o := orm.NewOrm()
	var goodsTypes []models.GoodsType // 商品所有类型
	o.QueryTable("GoodsType").All(&goodsTypes)
	this.Data["types"] = goodsTypes
	// 2 获取轮播图
	var goodsBanners []models.IndexGoodsBanner
	o.QueryTable("IndexGoodsBanner").OrderBy("Index").All(&goodsBanners)
	this.Data["goodsBanners"] = goodsBanners
	// 3 获取促销商品图
	var promotionBanners []models.IndexPromotionBanner
	o.QueryTable("IndexPromotionBanner").OrderBy("Index").All(&promotionBanners)
	this.Data["promotionBanners"] = promotionBanners
	// 4 获取首页商品展示
	var Goods []map[string]interface{} // 总容器
	for _, v := range goodsTypes {
		var imageGoods []models.IndexTypeGoodsBanner
		var textGoods []models.IndexTypeGoodsBanner
		qs := o.QueryTable("IndexTypeGoodsBanner").RelatedSel("GoodsSKU", "GoodsType").Filter("GoodsType__Id", v.Id)
		qs.Filter("DisplayType", 1).All(&imageGoods)
		qs.Filter("DisplayType", 0).All(&textGoods)
		// 定义行容器
		temp := make(map[string]interface{})
		temp["goodsType"] = v
		temp["imageGoods"] = imageGoods
		temp["textGoods"] = textGoods
		Goods = append(Goods, temp)
	}
	this.Data["Goods"] = Goods
	this.TplName = "index_sx.html"
}

// 商品详情
func (this *GoodsController)ShowGoodsDetail(){
	// 获取数据
	id, err := this.GetInt("id")
	// 校验数据
	if err != nil {
		beego.Error("未知链接错误")
		this.Redirect("/index_sx", 302)
		return
	}
	// 处理数据
	o := orm.NewOrm()
	// 商品 SKU
	var goodsSKU models.GoodsSKU
	//goodsSKU.Id = id
	//o.Read(&goodsSKU)

	// 获取商品详情
	o.QueryTable("GoodsSKU").RelatedSel("Goods", "GoodsType").Filter("Id", id).One(&goodsSKU)
	// 获取同一类型的新品推荐
	var newGoods []models.GoodsSKU
	qs := o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Name", goodsSKU.GoodsType.Name)
	qs.OrderBy("-Time").Limit(2,0).All(&newGoods)

	// 返回数据
	this.Data["goodsSKU"] = goodsSKU
	this.Data["newGoods"] = newGoods
	this.TplName = "detail.html"
}

// 页码显示	pageCount:页数	pageIndex:当前页码
func PageEdit(pageCount int, pageIndex int) []int {
	var page []int
	if pageCount < 5 { // 小于五页
		for i := 1; i <= 5; i++ {
			page = append(page, i)
		}
	} else if pageIndex <= 3{ // 大于五页，前三页
		for i := 1; i <=5; i++ {
			page = append(page, i)
		}
	} else if pageIndex >= pageCount - 2 { // 大于五页，后三页
		for i := pageIndex; i <= pageCount; i++ {
			page = append(page, i)
		}
	} else { // 中间页
		for i := pageIndex - 2; i <= pageIndex + 2; i++ {
			page = append(page, i)
		}
	}
	return page
}

// 商品列表
func (this *GoodsController)ShowList(){
	// 获取数据
	id, err := this.GetInt("id")
	sort := this.GetString("sort")
	// 校验数据
	if err != nil {
		beego.Error("获取商品列表 Id 失败")
		this.Redirect("inedx_sx", 302)
		return
	}
	// 处理数据
	o := orm.NewOrm()
	var goods []models.GoodsSKU // 同一类型商品
	qs := o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id", id)
	count, _ := qs.Count()	// 总个数
	pageSize := 1 // 每页显示个数
	pageCount := int(math.Ceil(float64(count)/float64(pageSize)))	// 总页数
	// 获取当前页码
	pageIndex, err := this.GetInt("pageIndex")
	if err != nil { // 默认显示第一页
		pageIndex = 1
	}
	pages := PageEdit(pageCount, pageIndex)
	// 获取上一页、下一页的值
	var prePage,nextPage int
	// 设置范围
	if pageIndex - 1 <= 0 {
		prePage = 1
	} else {
		prePage = pageIndex - 1
	}
	if pageIndex + 1 >= pageCount {
		nextPage = pageCount
	} else {
		nextPage = pageIndex + 1
	}
	qs = qs.Limit(pageSize, pageSize*(pageIndex-1))
	if sort == "" {
		qs.All(&goods) // 显示全部
	} else if sort == "price" {
		qs.OrderBy("Price").All(&goods) // 按照价格排序
	} else {
		qs.OrderBy("-Sales").All(&goods) // 按照销量排序
	}

	var goodsSKU []models.GoodsSKU // 新品推荐
	o.QueryTable("GoodsSKU").OrderBy("-Time").Limit(2, 0).All(&goodsSKU)
	// 返回数据
	this.Data["prePage"] = prePage
	this.Data["nextPage"] = nextPage
	this.Data["id"] = id
	this.Data["sort"] = sort
	this.Data["pages"] = pages
	this.Data["goods"] = goods
	this.Data["goodsSKU"] = goodsSKU
	this.TplName = "list.html"
}

// 搜索商品
func (this *GoodsController)HandleSearch(){
	// 获取数据
	goodsName := this.GetString("goodsName")
	// 校验数据
	if goodsName == "" {
		this.Redirect("/index_sx", 302)
		return
	}
	// 处理数据
	o := orm.NewOrm()
	var goods []models.GoodsSKU
	o.QueryTable("GoodsSKU").Filter("Name__icontains", goodsName).All(&goods)
	// 返回数据
	this.Data["goods"] = goods
	this.TplName = "search.html"
}
