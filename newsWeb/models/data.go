package models

import (
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

// Article 与 ArticleType 是一对多关系
// Article 与 User 是多对多关系

type User struct{
	Id int
	Name string
	Pwd string
	Articles []*Article `orm:"rel(m2m)"`
}

type Article struct{
	Id int	`orm:"pk;auto"`
	Title string	`orm:"size(40);unique"`
	Content string	`orm:"size(500)"`
	Img string	`orm:"null"`
	Time time.Time	`orm:"type(datetime);auto_now_add"`
	ReadCount int	`orm:"default(0)"`
	ArticleType *ArticleType `orm:"rel(fk)"`
	Users []*User `orm:"reverse(many)"`
}

type ArticleType struct {
	Id int
	TypeName string `orm:"unique"`
	Articles []*Article `orm:"reverse(many)"`
}

func init(){
	orm.RegisterDataBase("default", "mysql", "root:123456@tcp(127.0.0.1:3306)/newsWeb")
	orm.RegisterModel(new(User), new(Article), new(ArticleType))
	orm.RunSyncdb("default", false, true)
}
