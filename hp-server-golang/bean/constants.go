package bean

// Token 相关常量
const (
	// TokenExpireDays token 过期天数
	TokenExpireDays = 3
	// TokenExpireMs token 过期毫秒数 = 3天
	TokenExpireMs = int64(TokenExpireDays * 24 * 60 * 60 * 1000)
)
