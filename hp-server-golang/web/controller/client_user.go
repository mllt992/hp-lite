package controller

import (
	"encoding/json"
	"hp-server-lib/bean"
	"hp-server-lib/entity"
	"hp-server-lib/service"
	"net/http"
	"strconv"
)

type ClientUserController struct {
	service.UserCustomService
}

func (receiver ClientUserController) Add(w http.ResponseWriter, r *http.Request) {
	var msg entity.UserCustomEntity
	// 解析请求体中的JSON数据
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	receiver.AddData(msg)
	json.NewEncoder(w).Encode(bean.ResOk(nil))
}

func (receiver ClientUserController) List(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	page := queryParams.Get("current")
	pageSize := queryParams.Get("pageSize")
	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)
	if pageInt == 0 {
		pageInt = 1
	}
	if pageSizeInt == 0 {
		pageSizeInt = 10
	}

	json.NewEncoder(w).Encode(bean.ResOk(receiver.ListData(pageInt, pageSizeInt)))
}

func (receiver ClientUserController) Del(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	id := queryParams.Get("id")
	idInt, _ := strconv.Atoi(id)
	receiver.RemoveData(idInt)
	json.NewEncoder(w).Encode(bean.ResOk(nil))
}
