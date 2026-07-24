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

type DomainController struct {
	service.DomainService
}

func (receiver DomainController) GetDomainList(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.Header.Get(middleware.HeaderUserId))
	queryParams := r.URL.Query()
	page := queryParams.Get("current")
	pageSize := queryParams.Get("pageSize")
	keyword := queryParams.Get("keyword")
	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)
	if pageInt == 0 {
		pageInt = 1
	}
	if pageSizeInt == 0 {
		pageSizeInt = 10
	}
	json.NewEncoder(w).Encode(bean.ResOk(receiver.DomainList(id, pageInt, pageSizeInt, keyword)))
}

func (receiver DomainController) RemoveDomain(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	id := queryParams.Get("id")
	idInt, _ := strconv.Atoi(id)
	json.NewEncoder(w).Encode(bean.ResOk(receiver.RemoveData(idInt)))
}

func (receiver DomainController) Query(w http.ResponseWriter, r *http.Request) {
	userId, _ := strconv.Atoi(r.Header.Get(middleware.HeaderUserId))
	queryParams := r.URL.Query()
	keyword := queryParams.Get("keyword")
	json.NewEncoder(w).Encode(bean.ResOk(receiver.DomainListByKey(userId, keyword)))
}

func (receiver DomainController) Gen(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	id := queryParams.Get("id")
	idInt, _ := strconv.Atoi(id)
	json.NewEncoder(w).Encode(bean.ResOk(receiver.GenSsl(false, idInt)))
}

func (receiver DomainController) Add(w http.ResponseWriter, r *http.Request) {
	userId, _ := strconv.Atoi(r.Header.Get(middleware.HeaderUserId))
	var msg entity.UserDomainEntity
	// 解析请求体中的JSON数据
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	msg.UserId = &userId
	err = receiver.AddData(msg)
	if err == nil {
		json.NewEncoder(w).Encode(bean.ResOk(nil))
		return
	}
	json.NewEncoder(w).Encode(bean.ResError(err.Error()))
}
