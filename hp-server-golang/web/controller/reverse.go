package controller

import (
	"encoding/json"
	"hp-server-lib/bean"
	"hp-server-lib/entity"
	"hp-server-lib/service"
	"hp-server-lib/web/middleware"
	"net/http"
	"strconv"
)

type ReverseController struct {
	service.ReverseService
}

func (receiver ReverseController) Add(w http.ResponseWriter, r *http.Request) {
	userId, _ := strconv.Atoi(r.Header.Get(middleware.HeaderUserId))
	var msg entity.UserReverseEntity
	// 解析请求体中的JSON数据
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	msg.UserId = &userId
	err = receiver.AddData(msg)
	if err != nil {
		json.NewEncoder(w).Encode(bean.ResError(err.Error()))
		return
	}
	json.NewEncoder(w).Encode(bean.ResOk(nil))
}

func (receiver ReverseController) List(w http.ResponseWriter, r *http.Request) {
	userId, _ := strconv.Atoi(r.Header.Get(middleware.HeaderUserId))
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
	json.NewEncoder(w).Encode(bean.ResOk(receiver.ListData(userId, pageInt, pageSizeInt)))
}

func (receiver ReverseController) Del(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	id := queryParams.Get("id")
	idInt, _ := strconv.Atoi(id)
	receiver.RemoveData(idInt)
	json.NewEncoder(w).Encode(bean.ResOk(nil))
}
