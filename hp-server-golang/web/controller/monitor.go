package controller

import (
	"encoding/json"
	"hp-server-lib/bean"
	"hp-server-lib/service"
	"hp-server-lib/web/middleware"
	"net/http"
	"strconv"
)

type MonitorController struct {
	service.MonitorService
}

func (receiver MonitorController) List(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.Header.Get(middleware.HeaderUserId))
	data := receiver.ListData(id)
	if data != nil {
		json.NewEncoder(w).Encode(bean.ResOk(data))
		return
	} else {
		json.NewEncoder(w).Encode(bean.ResError("登陆失败"))
	}
}

func (receiver MonitorController) Detail(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	id := queryParams.Get("id")
	data := receiver.DetailData(id)
	if data != nil {
		json.NewEncoder(w).Encode(bean.ResOk(data))
		return
	} else {
		json.NewEncoder(w).Encode(bean.ResError("登陆失败"))
	}
}
