package middleware

import (
	"encoding/json"
	"hp-server-lib/bean"
	"hp-server-lib/util"
	"net/http"
	"strconv"
	"time"
)

// Header 键名——controller 通过 r.Header.Get("X-User-Id") 获取当前用户 ID
const (
	HeaderUserId = "X-User-Id"
	HeaderRole   = "X-Role"
)

// Auth 认证中间件：校验 Token（含过期检查），通过后注入 X-User-Id / X-Role 到请求头
// controller 中无需导入本包，直接从请求头读取即可
func Auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("token")
		if token == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(bean.ResErrorCode(-2, "用户权限校验失败"))
			return
		}

		userId, role, timestamp, err := util.DecodeToken(token)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(bean.ResErrorCode(-2, "用户权限校验失败"))
			return
		}

		// Token 时效性校验：timestamp 必须在过去且未超过有效期
		now := time.Now().UnixMilli()
		age := now - timestamp
		if age < 0 || age > bean.TokenExpireMs {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(bean.ResErrorCode(-2, "token已过期，请重新登录"))
			return
		}

		// 将认证信息写入请求头，下游 handler 可直接读取
		r.Header.Set(HeaderUserId, strconv.Itoa(userId))
		r.Header.Set(HeaderRole, role)
		next(w, r)
	}
}

// Admin 管理员权限中间件：在 Auth 基础上额外校验角色是否为 ADMIN
func Admin(next http.HandlerFunc) http.HandlerFunc {
	return Auth(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(HeaderRole) != "ADMIN" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(bean.ResErrorCode(-2, "用户权限校验失败"))
			return
		}
		next(w, r)
	})
}
