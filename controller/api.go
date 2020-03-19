package controller

import (
	"encoding/json"
	"github.com/astaxie/beego"
	"scheduler/common"
	. "scheduler/master"
)

type ApiController struct {
	beego.Controller
}

type Response struct {
	Code    int64       `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

/*
保存新增的job任务

{
"name" : "job1",
"command" : "echo hello",
"cronExpr" : ""
}
*/
func (c *ApiController) Save() {
	var job Job

	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &job); err != nil {
		c.Data["json"] = Response{Code: 500, Message: err.Error()}
		c.ServeJSON()
		return
	}

	if oldJob, err := job.SaveJob(); err != nil {
		c.Data["json"] = Response{Code: 500, Message: err.Error()}
		c.ServeJSON()
		return
	} else {
		c.Data["json"] = Response{Code: 200, Message: "success", Data: oldJob}
		c.ServeJSON()
	}
}

/*
删除任务

{
"name" : "job1"
}
*/
func (c *ApiController) Delete() {
	var job Job

	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &job); err != nil {
		c.Data["json"] = Response{Code: 500, Message: err.Error()}
		c.ServeJSON()
		return
	}

	old, err := job.DeleteJob()
	if err != nil {
		c.Data["json"] = Response{Code: 500, Message: err.Error()}
		c.ServeJSON()
		return
	}

	c.Data["json"] = Response{Code: 200, Message: "success", Data: old}
	c.ServeJSON()
}

//返回所有的任务列表
func (c *ApiController) JobList() {
	var job Job
	jobs, err := job.JobList()
	if err != nil {
		c.Data["json"] = Response{Code: 500, Message: err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = Response{Code: 200, Message: "success", Data: jobs}
	c.ServeJSON()
}

/*
kill任务

{
"name" : "job1"
}
*/
func (c *ApiController) KillJob() {
	var job Job

	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &job); err != nil {
		c.Data["json"] = Response{Code: 500, Message: err.Error()}
		c.ServeJSON()
		return
	}

	if err := job.KillJob(); err != nil {
		c.Data["json"] = Response{Code: 500, Message: err.Error()}
		c.ServeJSON()
		return
	}

	c.Data["json"] = Response{Code: 200, Message: "success"}
	c.ServeJSON()
}

/*
查询任务的执行日志

{
"jobName" : "",
"skip" : 1,
"limit" : 2
}
*/
func (c *ApiController) JobLog() {
	var log common.Log

	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &log); err != nil {
		c.Data["json"] = Response{Code: 500, Message: err.Error()}
		c.ServeJSON()
		return
	}

	if log.Skip == 0 {
		log.Skip = 10
	}
	if log.Limit == 0 {
		log.Limit = 10
	}

	//查询日志
	jobLog, err := JobLogs(&log)
	if err != nil {
		c.Data["json"] = Response{Code: 500, Message: err.Error()}
		c.ServeJSON()
		return
	}

	c.Data["json"] = Response{Code: 200, Message: "success", Data: jobLog}
	c.ServeJSON()
}

// 获取在线的worker节点list
func (c *ApiController) WorkList() {
	list, err := WorkerList()
	if err != nil {
		c.Data["json"] = Response{Code: 500, Message: err.Error()}
		c.ServeJSON()
		return
	}

	c.Data["json"] = Response{Code: 200, Message: "success", Data: list}
	c.ServeJSON()
}
