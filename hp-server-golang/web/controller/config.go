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

type ConfigController struct {
	service.ConfigService
}

func (receiver ConfigController) GetDeviceKey(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.Header.Get(middleware.HeaderUserId))
	json.NewEncoder(w).Encode(bean.ResOk(receiver.DeviceKey(id)))
}

func (receiver ConfigController) GetConfigList(w http.ResponseWriter, r *http.Request) {
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
	json.NewEncoder(w).Encode(bean.ResOk(receiver.ConfigList(id, pageInt, pageSizeInt, keyword)))
}

func (receiver ConfigController) RemoveConfig(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	configId := queryParams.Get("configId")
	configIdInt, _ := strconv.Atoi(configId)
	json.NewEncoder(w).Encode(bean.ResOk(receiver.RemoveData(configIdInt)))
}

func (receiver ConfigController) Add(w http.ResponseWriter, r *http.Request) {
	var msg entity.UserConfigEntity
	// 解析请求体中的JSON数据
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = receiver.AddData(msg)
	if err == nil {
		json.NewEncoder(w).Encode(bean.ResOk(nil))
		return
	}
	json.NewEncoder(w).Encode(bean.ResError(err.Error()))
}

func (receiver ConfigController) Keyword(w http.ResponseWriter, r *http.Request) {
	userId, _ := strconv.Atoi(r.Header.Get(middleware.HeaderUserId))
	queryParams := r.URL.Query()
	keyword := queryParams.Get("keyword")
	data := receiver.KeywordData(userId, keyword)
	if data != nil {
		json.NewEncoder(w).Encode(bean.ResOk(data))
		return
	}
	json.NewEncoder(w).Encode(bean.ResOk(nil))
}

func (receiver ConfigController) RefConfig(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	configId := queryParams.Get("configId")
	configIdInt, _ := strconv.Atoi(configId)
	err := receiver.RefData(configIdInt)
	if err == nil {
		json.NewEncoder(w).Encode(bean.ResOk(nil))
		return
	}
	json.NewEncoder(w).Encode(bean.ResError(err.Error()))
}

func (receiver ConfigController) ChangeStatus(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	configId := queryParams.Get("configId")
	configIdInt, _ := strconv.Atoi(configId)
	err := receiver.ChangeStatusData(configIdInt)
	if err == nil {
		json.NewEncoder(w).Encode(bean.ResOk(nil))
		return
	}
	json.NewEncoder(w).Encode(bean.ResError(err.Error()))
}
