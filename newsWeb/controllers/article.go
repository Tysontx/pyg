package controllers

import (
	"github.com/astaxie/beego"
	"path"
	"time"
	"github.com/astaxie/beego/orm"
	"newsWeb/models"
	"math"
	"strconv"
	"github.com/gomodule/redigo/redis"
	"bytes"
	"encoding/gob"
)

type ArticleController struct {
	beego.Controller
}

// 展示首页
func (this *ArticleController)ShowIndex() {
	// 校验登录状态
	userName := this.GetSession("userName")
	if userName == nil {
		this.Redirect("/article/login", 302)
		return
	}
	this.Data["userName"] = userName.(string) // 断言
	// 获取所有文章显示到页面
	o := orm.NewOrm()
	// 指定查询表
	qs := o.QueryTable("Article")
	var articles []models.Article
	// 获取选中的类型
	typeName := this.GetString("select")
	var count int64
	if typeName == "" {
		// 获取总记录数
		count, _ = qs.RelatedSel("ArticleType").Count()
	} else {
		count, _ = qs.RelatedSel("ArticleType").Filter("ArticleType__TypeName", typeName).Count()
	}

	// 获取总页数
	pageIndex := 2
	pageCount := math.Ceil(float64(count) / float64(pageIndex))
	// 获取首页和末页数据
	// 获取页码
	pageNum, err := this.GetInt("pagenum")
	if err != nil {
		pageNum = 1
	}
	beego.Info("数据总页数为：", pageNum)
	// 获取对应页的数据
	if typeName == "" {
		qs.Limit(pageIndex, pageIndex*(pageNum - 1)).RelatedSel("ArticleType").All(&articles)
	} else {
		qs.Limit(pageIndex, pageIndex*(pageNum - 1)).RelatedSel("ArticleType").Filter("ArticleType__TypeName", typeName).All(&articles)
	}
	// 查询所有文章类型，并展示
	var articleTypes []models.ArticleType
	// o.QueryTable("ArticleType").All(&articleTypes)


	// 主页下拉框选择文章类型，因为要经常操作，所以将文章类型数据存入 redis 中，而不存入 mysql 中
	conn, err := redis.Dial("tcp", "127.0.0.1:6379") // 连接 redis 数据库
	if err != nil {
		beego.Error("redis 连接失败")
		return
	}
	defer conn.Close() // 操作完成后，关闭 redis 连接
	resp, err := conn.Do("get", "newsWeb") // 从 redis 中获取数据
	result, _ := redis.Bytes(resp, err)
	if len(result) == 0 { // 首次从 redis 中取出来的数据为空时，则从 mysql 中得到的文章类型信息写入 redis
		o.QueryTable("ArticleType").All(&articleTypes)
		// 序列化（也叫编码）
		var buffer bytes.Buffer
		enc := gob.NewEncoder(&buffer)
		enc.Encode(articleTypes)
		// 将编码内容存入 redis 中
		conn.Do("set", "newsWeb", buffer.Bytes())
		beego.Info("成功将文章类型写入 redis")
	} else { // 从 redis 获取到了数据
		// 反序列化（也叫解码）
		dec := gob.NewDecoder(bytes.NewReader(result))
		dec.Decode(&articleTypes)
		beego.Info("反序列化后的 articleTypes：", articleTypes)
	}

	this.Data["articleTypes"] = articleTypes
	this.Data["TypeName"] = typeName
	this.Data["articles"] = articles
	this.Data["count"] = count
	this.Data["pageCount"] = pageCount
	this.Data["pageNum"] = pageNum

	this.Layout = "layout.html"
	this.LayoutSections = make(map[string]string)
	this.LayoutSections["indexJS"] = "indexJS.html"
	this.TplName = "index.html"
}

// 展示添加文章页面
func (this *ArticleController)ShowAddArticle() {
	// 获取所有类型并绑定下拉框
	o := orm.NewOrm()
	var articleTypes []models.ArticleType
	o.QueryTable("ArticleType").All(&articleTypes)
	this.Data["articleTypes"] = articleTypes
	this.Layout = "layout.html"
	this.TplName = "add.html"
}

// 处理添加文章业务
func (this *ArticleController)HandleAddArticle(){
	// 获取文章标题
	title := this.GetString("articleName")
	// 获取文章类型
	typeName := this.GetString("select")
	// 获取文章内容
	content := this.GetString("content")
	// 判断标题和内容是否为空
	if title == "" || content == "" || typeName == "" {
		beego.Error("输入的信息不完整")
		this.Data["errmsg"] = "输入的信息不完整"
		this.Layout = "layout.html"
		this.TplName = "add.html"
		return
	}
	savePath := UploadFile(this, "uploadname", "add.html")
	// 处理数据，将数据写入数据库中
	// 新建 orm 对象
	o := orm.NewOrm()
	// 获取插入对象
	var Article models.Article
	// 给插入对象添加数据
	Article.Title = title
	Article.Content = content
	Article.Img = savePath
	// 获取一个类型对象，并插入文章中
	var articleType models.ArticleType
	articleType.TypeName = typeName
	o.Read(&articleType, "TypeName")
	Article.ArticleType = &articleType
	// 将插入对象写入数据库中
	_, err := o.Insert(&Article)
	if err != nil {
		beego.Error("获取数据错误：", err)
		this.Data["errmsg"] = "插入数据失败"
		this.Layout = "layout.html"
		this.TplName = "add.html"
		return
	}
	// 跳转到主页
	this.Redirect("/article/index", 302)
}

// 查看文章详情
func (this *ArticleController)ShowContent(){
	// 获取通过 url 传过来的 id
	id, err := this.GetInt("id")
	if err != nil {
		beego.Error("查看详情页的 id 获取失败")
		// 重定向跳转到 index 页面
		this.Redirect("/article/index", 302)
		return
	}
	// 校验数据
	o := orm.NewOrm()
	var article models.Article
	article.Id = id
	o.Read(&article)

	// 多对多查询（方法一）
	// o.LoadRelated(&article, "Users")
	// 多对多高级查询（方法二）
	var users []models.User
	o.QueryTable("User").Filter("Articles__Article__Id", id).Distinct().All(&users)
	this.Data["users"] = users

	// 更新阅读量
	article.ReadCount += 1
	o.Update(&article)

	// 插入多对多关系，根据用户名获取用户对象
	userName := this.GetSession("userName")
	var user models.User
	user.Name = userName.(string)
	o.Read(&user, "Name")
	// 多对多的插入操作
	// 获取 orm 对象
	// 获取被插入数据对象（文章）
	// 获取要插入对象（用户）
	// 获取多对多操作对象
	m2m := o.QueryM2M(&article, "Users")
	// 用多对多操作对象插入
	m2m.Add(user)

	// 返回数据
	this.Data["article"] = article
	this.Layout = "layout.html"
	this.TplName = "content.html"
}

// 展示文章编辑页面
func (this *ArticleController)ShowUpdate(){
	// 获取通过 url 传过来的 id
	id, err := this.GetInt("id")
	// 校验数据
	if err != nil {
		beego.Error("编辑页的 id 获取失败")
		this.Redirect("/article/index", 302)
		return
	}
	// 处理数据
	o := orm.NewOrm()
	var article models.Article
	article.Id = id
	o.Read(&article)
	// 返回数据
	this.Data["article"] = article
	this.Layout = "layout.html"
	this.TplName = "update.html"
}

// 封装上传文件处理函数
func UploadFile(this *ArticleController, filePath string, errHtml string) string {
	// 获取上传的图片
	file, head, err := this.GetFile(filePath)
	if err != nil {
		beego.Error("图片上传失败")
		this.Data["errmsg"] = "图片上传失败"
		this.Layout = "layout.html"
		this.TplName = errHtml
		return ""
	}
	// 关闭文件
	defer file.Close()
	// 检验文件大小
	if head.Size > 5000000 {
		beego.Error("图片大小超过 5MB,上传失败")
		this.Data["errmsg"] = "图片超过 5MB,上传失败"
		this.Layout = "layout.html"
		this.TplName = errHtml
		return ""
	}
	// 校验文件格式，获取文件后缀
	ext := path.Ext(head.Filename)
	if ext != ".jpg" && ext != ".png" && ext != ".jpeg" {
		beego.Error("图片格式错误")
		this.Data["errmsg"] = "图片格式错误"
		this.Layout = "layout.html"
		this.TplName = errHtml
		return ""
	}
	// 给文件重命名
	fileName := time.Now().Format("200601021504051111") + ext
	// 把上传的文件写入存储到项目文件中
	this.SaveToFile(filePath, "./static/img/"+fileName)
	return "/static/img/"+fileName
}

// 处理文章编辑
func (this *ArticleController)HandleUpdate(){
	// 获取前端传来的数据
	title := this.GetString("articleName")
	content := this.GetString("content")
	savePath := UploadFile(this, "uploadname", "update.html")
	id, _ := this.GetInt("id")
	// 校验数据
	if title == "" || content == "" || savePath == "" {
		beego.Error("获取到的数据并不完整")
		this.Redirect("/article/update?id=" + strconv.Itoa(id), 302)
		return
	}
	// 处理数据
	o := orm.NewOrm()
	var article models.Article
	article.Id = id
	o.Read(&article)
	article.Title = title
	article.Content = content
	article.Img = savePath
	o.Update(&article)
	// 返回数据
	this.Redirect("/article/index", 302)
}

// 删除文章
func (this *ArticleController)HandleDelete(){
	// 获取数据
	id, err := this.GetInt("id")
	// 校验数据
	if err != nil {
		beego.Error("删除数据 id 获取失败")
		this.Redirect("/article/index", 302)
		return
	}
	// 处理数据
	o := orm.NewOrm()
	var article models.Article
	article.Id = id
	o.Delete(&article)
	// 返回数据
	this.Redirect("/article/index", 302)
}

// 展示添加分类页面
func (this *ArticleController)ShowAddType(){
	// 获取所有类型，并展示在页面上
	o := orm.NewOrm()
	var articleTypes []models.ArticleType
	o.QueryTable("ArticleType").All(&articleTypes)
	// 返回数据
	this.Data["articleTypes"] = articleTypes
	this.Layout = "layout.html"
	this.TplName="addType.html"
}

// 处理添加类型请求
func (this *ArticleController)HandleAddType(){
	// 获取数据
	typeName := this.GetString("typeName")
	// 校验数据
	if typeName == "" {
		beego.Error("类型名称传输失败")
		this.Redirect("/article/addType", 302)
		return
	}
	// 处理数据（插入操作）
	o := orm.NewOrm()
	var articleType models.ArticleType
	articleType.TypeName = typeName
	o.Insert(&articleType)
	// 返回数据
	this.Redirect("/article/addType", 302)
}

// 删除类型
func (this *ArticleController)DeleteType(){
	// 获取数据
	id, err := this.GetInt("id")
	// 校验数据
	if err != nil {
		beego.Error("获取文章 id 失败")
		this.Redirect("/article/addType", 302)
		return

	}
	// 处理数据（删除数据）
	o := orm.NewOrm()
	var articleType models.ArticleType
	articleType.Id = id
	o.Delete(&articleType, "Id")
	// 返回数据
	this.Redirect("/article/addType", 302)
}