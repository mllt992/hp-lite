package controller

import (
	"encoding/json"
	"hp-server-lib/bean"
	"hp-server-lib/service"
	"hp-server-lib/web/middleware"
	"net/http"
	"strconv"
)

type DeviceController struct {
	service.DeviceService
}

func (receiver DeviceController) Add(w http.ResponseWriter, r *http.Request) {
	userId, _ := strconv.Atoi(r.Header.Get(middleware.HeaderUserId))
	var msg bean.ReqDeviceInfo
	// 解析请求体中的JSON数据
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = receiver.AddData(userId, msg)
	if err == nil {
		json.NewEncoder(w).Encode(bean.ResOk(nil))
		return
	}
	json.NewEncoder(w).Encode(bean.ResError(err.Error()))
}

func (receiver DeviceController) Update(w http.ResponseWriter, r *http.Request) {
	var msg bean.ReqDeviceInfo
	// 解析请求体中的JSON数据
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = receiver.UpdateData(msg)
	if err == nil {
		json.NewEncoder(w).Encode(bean.ResOk(nil))
		return
	}
	json.NewEncoder(w).Encode(bean.ResError(err.Error()))
}

func (receiver DeviceController) List(w http.ResponseWriter, r *http.Request) {
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

func (receiver DeviceController) Del(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	deviceId := queryParams.Get("deviceId")
	err := receiver.RemoveData(deviceId)
	if err == nil {
		json.NewEncoder(w).Encode(bean.ResOk(nil))
		return
	}
	json.NewEncoder(w).Encode(bean.ResError(err.Error()))
}

func (receiver DeviceController) Stop(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	deviceId := queryParams.Get("deviceId")
	json.NewEncoder(w).Encode(bean.ResOk(receiver.StopData(deviceId)))
}
