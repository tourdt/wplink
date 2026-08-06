package metrics

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"wplink/backend/app/internal/handler/handlerx"
	metricslogic "wplink/backend/app/internal/logic/metrics"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/core/logx"
)

func RecordMerchantMapEventHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	limiter := newMerchantMapEventLimiter(60, time.Minute, 4096, time.Now)
	return func(w http.ResponseWriter, r *http.Request) {
		if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.MerchantMapEventsModel == nil {
			logx.WithContext(r.Context()).Error("地图行为 Handler 存储依赖未配置")
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "地图行为服务暂不可用，请稍后重试"))
			return
		}
		rawBody, err := handlerx.ReadLimitedBody(r, 4096)
		if err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "地图行为请求内容过大"))
			return
		}
		var req types.MerchantMapEventReq
		decoder := json.NewDecoder(bytes.NewReader(rawBody))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "请求参数格式不正确"))
			return
		}
		var trailing interface{}
		if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
			// 地图埋点只接受一个完整 JSON 文档，避免合法首值后的第二对象或垃圾内容被静默忽略。
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "请求参数格式不正确"))
			return
		}
		if !limiter.Allow(handlerx.ClientIP(r), req.VisitorKey) {
			response.JSON(w, nil, errx.New(errx.CodeRateLimited, "操作频繁，请稍后再试"))
			return
		}
		subject, authenticated, err := handlerx.OptionalUser(r, svcCtx.UserTokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		userID := ""
		if authenticated {
			userID = subject.UserID
		}
		resp, err := metricslogic.NewRecordMerchantMapEventLogic(svcCtx.APIStore).RecordMerchantMapEvent(r.Context(), metricslogic.RecordMerchantMapEventReq{
			UserID: userID, MerchantID: req.MerchantId, TargetMerchantID: req.TargetMerchantId,
			VisitorKey: req.VisitorKey, SessionID: req.SessionId, EventType: req.EventType, Source: req.Source,
		})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		response.JSON(w, types.MerchantMapEventResp{Recorded: resp.Recorded}, nil)
	}
}

type merchantMapEventWindow struct {
	startedAt time.Time
	count     int
}

type merchantMapEventLimiter struct {
	mu        sync.Mutex
	windows   map[[sha256.Size]byte]merchantMapEventWindow
	limit     int
	window    time.Duration
	maxKeys   int
	now       func() time.Time
	lastSweep time.Time
}

func newMerchantMapEventLimiter(limit int, window time.Duration, maxKeys int, now func() time.Time) *merchantMapEventLimiter {
	if now == nil {
		now = time.Now
	}
	return &merchantMapEventLimiter{
		windows: make(map[[sha256.Size]byte]merchantMapEventWindow),
		limit:   limit, window: window, maxKeys: maxKeys, now: now,
	}
}

func (l *merchantMapEventLimiter) Allow(clientIP string, visitorKey string) bool {
	if l == nil || l.limit <= 0 || l.window <= 0 || l.maxKeys <= 0 || l.now == nil {
		return false
	}
	now := l.now()
	key := sha256.Sum256([]byte(strings.TrimSpace(clientIP) + "\x00" + strings.TrimSpace(visitorKey)))

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.lastSweep.IsZero() || !now.Before(l.lastSweep.Add(l.window)) || len(l.windows) >= l.maxKeys {
		for currentKey, current := range l.windows {
			if !now.Before(current.startedAt.Add(l.window)) {
				delete(l.windows, currentKey)
			}
		}
		l.lastSweep = now
	}
	current, exists := l.windows[key]
	if exists && !now.Before(current.startedAt.Add(l.window)) {
		delete(l.windows, key)
		exists = false
	}
	if exists {
		if current.count >= l.limit {
			return false
		}
		current.count++
		l.windows[key] = current
		return true
	}
	if len(l.windows) >= l.maxKeys {
		return false
	}
	l.windows[key] = merchantMapEventWindow{startedAt: now, count: 1}
	return true
}
