package router

import (
	"github.com/astaxie/beego"
	"scheduler/controller"
)

func init() {

	beego.Router("/", &controller.MainController{})
	beego.Router("/*", &controller.MainController{})

	beego.Router("/job/save", &controller.ApiController{}, "post:Save")
	beego.Router("/job/delete", &controller.ApiController{}, "post:Delete")
	beego.Router("/job/jobList", &controller.ApiController{}, "get:JobList")
	beego.Router("/job/killJob", &controller.ApiController{}, "post:KillJob")
	beego.Router("/job/log", &controller.ApiController{}, "post:JobLog")
	beego.Router("/worker1/list", &controller.ApiController{}, "get:WorkList")
}
