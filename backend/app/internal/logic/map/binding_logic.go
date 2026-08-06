package maplogic

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type BindingStore interface {
	GetMapBindingStatus(ctx context.Context, merchantID string) (model.MapBindingStatus, error)
	ListMapBindCandidates(ctx context.Context, filter model.MapBindCandidateFilter) ([]model.MapBindCandidate, error)
	CreateMapBindRequest(ctx context.Context, input model.MapBindRequestInput) (model.MapBindRequest, error)
	ListMapBindRequests(ctx context.Context, filter model.ListMapBindRequestsFilter) ([]model.MapBindRequest, error)
	ReviewMapBindRequest(ctx context.Context, input model.ReviewMapBindRequestInput) (model.MapBindRequest, error)
}

type BindingLogic struct {
	store BindingStore
}

func NewBindingLogic(store BindingStore) *BindingLogic {
	return &BindingLogic{store: store}
}

type MapBindCandidateItem struct {
	ObjectID     string `json:"objectId"`
	SceneCode    string `json:"sceneCode"`
	SceneName    string `json:"sceneName"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	Address      string `json:"address,omitempty"`
	MerchantID   string `json:"merchantId,omitempty"`
	MerchantName string `json:"merchantName,omitempty"`
	IsBound      bool   `json:"isBound"`
}

type MapBindRequestItem struct {
	ID              string   `json:"id"`
	MerchantID      string   `json:"merchantId"`
	MerchantName    string   `json:"merchantName,omitempty"`
	ObjectID        string   `json:"objectId"`
	SceneCode       string   `json:"sceneCode"`
	SceneName       string   `json:"sceneName,omitempty"`
	ObjectCode      string   `json:"objectCode,omitempty"`
	ObjectName      string   `json:"objectName,omitempty"`
	ApplicantUserID string   `json:"applicantUserId,omitempty"`
	EvidenceImages  []string `json:"evidenceImages"`
	Note            string   `json:"note,omitempty"`
	Status          string   `json:"status"`
	ReviewNote      string   `json:"reviewNote,omitempty"`
	ReviewedBy      string   `json:"reviewedBy,omitempty"`
	ReviewedAt      string   `json:"reviewedAt,omitempty"`
	CreatedAt       string   `json:"createdAt"`
}

type MapBindingStatusResp struct {
	BoundObject   *MapBindCandidateItem `json:"boundObject,omitempty"`
	LatestRequest *MapBindRequestItem   `json:"latestRequest,omitempty"`
}

type ListMapBindCandidatesReq struct {
	MerchantID string
	ObjectID   string
	SceneCode  string
	Keyword    string
	Limit      int64
}

type ListMapBindCandidatesResp struct {
	Items []MapBindCandidateItem `json:"items"`
}

type SubmitMapBindRequestReq struct {
	ObjectID        string
	ApplicantUserID string
	EvidenceImages  []string
	Note            string
}

type SubmitMapBindRequestResp struct {
	Item MapBindRequestItem `json:"item"`
}

type ListAdminMapBindRequestsReq struct {
	Status   string
	Keyword  string
	Page     int64
	PageSize int64
}

type ListAdminMapBindRequestsResp struct {
	Items []MapBindRequestItem `json:"items"`
}

type ReviewMapBindRequestReq struct {
	Action     string
	ReviewerID string
	ReviewNote string
}

type ReviewMapBindRequestResp struct {
	Item MapBindRequestItem `json:"item"`
}

func (l *BindingLogic) GetStatus(ctx context.Context, merchantID string) (MapBindingStatusResp, error) {
	merchantID = strings.TrimSpace(merchantID)
	if merchantID == "" {
		return MapBindingStatusResp{}, errx.New(errx.CodeValidationFailed, "请选择商家")
	}
	status, err := l.store.GetMapBindingStatus(ctx, merchantID)
	if err != nil {
		LogMapDependencyFailure(ctx, "查询商家地图绑定状态失败", "get_binding_status", err,
			logx.Field("merchantId", merchantID))
		return MapBindingStatusResp{}, errx.New(errx.CodeInternalError, "地图档口绑定状态加载失败，请稍后重试")
	}
	resp := MapBindingStatusResp{}
	if status.BoundObject != nil {
		item := mapBindCandidateItem(*status.BoundObject)
		resp.BoundObject = &item
	}
	if status.LatestRequest != nil {
		item := mapBindRequestItem(*status.LatestRequest)
		resp.LatestRequest = &item
	}
	return resp, nil
}

func (l *BindingLogic) ListCandidates(ctx context.Context, req ListMapBindCandidatesReq) (ListMapBindCandidatesResp, error) {
	merchantID := strings.TrimSpace(req.MerchantID)
	if merchantID == "" {
		return ListMapBindCandidatesResp{}, errx.New(errx.CodeValidationFailed, "请选择商家")
	}
	candidates, err := l.store.ListMapBindCandidates(ctx, model.MapBindCandidateFilter{
		MerchantID: merchantID,
		ObjectID:   strings.TrimSpace(req.ObjectID),
		SceneCode:  strings.TrimSpace(req.SceneCode),
		Keyword:    strings.TrimSpace(req.Keyword),
		Limit:      req.Limit,
	})
	if err != nil {
		LogMapDependencyFailure(ctx, "查询地图绑定候选点位失败", "list_binding_candidates", err,
			logx.Field("merchantId", merchantID), logx.Field("objectId", strings.TrimSpace(req.ObjectID)),
			logx.Field("sceneCode", strings.TrimSpace(req.SceneCode)))
		return ListMapBindCandidatesResp{}, errx.New(errx.CodeInternalError, "地图档口加载失败，请稍后重试")
	}
	return ListMapBindCandidatesResp{Items: mapBindCandidateItems(candidates)}, nil
}

func (l *BindingLogic) SubmitRequest(ctx context.Context, merchantID string, req SubmitMapBindRequestReq) (SubmitMapBindRequestResp, error) {
	merchantID = strings.TrimSpace(merchantID)
	objectID := strings.TrimSpace(req.ObjectID)
	if merchantID == "" {
		return SubmitMapBindRequestResp{}, errx.New(errx.CodeValidationFailed, "请选择商家")
	}
	if objectID == "" {
		return SubmitMapBindRequestResp{}, errx.New(errx.CodeValidationFailed, "请选择要绑定的地图档口")
	}
	request, err := l.store.CreateMapBindRequest(ctx, model.MapBindRequestInput{
		MerchantID:      merchantID,
		ObjectID:        objectID,
		ApplicantUserID: strings.TrimSpace(req.ApplicantUserID),
		EvidenceImages:  cleanStringSlice(req.EvidenceImages),
		Note:            strings.TrimSpace(req.Note),
	})
	if err != nil {
		if errors.Is(err, model.ErrMapMerchantAlreadyBound) {
			return SubmitMapBindRequestResp{}, errx.New(errx.CodeStateConflict, "该商家已绑定主档口，如需更换请先提交位置纠错")
		}
		if errors.Is(err, model.ErrMapObjectAlreadyBound) {
			return SubmitMapBindRequestResp{}, errx.New(errx.CodeStateConflict, "该档口已绑定其他商家，请选择其他档口或提交位置纠错")
		}
		if errors.Is(err, model.ErrMapBindRequestPending) {
			return SubmitMapBindRequestResp{}, errx.New(errx.CodeStateConflict, "该档口已有待审核绑定申请，请勿重复提交")
		}
		if errors.Is(err, sql.ErrNoRows) {
			return SubmitMapBindRequestResp{}, errx.New(errx.CodeResourceNotFound, "地图档口不存在或暂不可绑定")
		}
		LogMapDependencyFailure(ctx, "自动绑定地图档口失败", "create_binding_request", err,
			logx.Field("merchantId", merchantID), logx.Field("objectId", objectID))
		return SubmitMapBindRequestResp{}, errx.New(errx.CodeInternalError, "档口绑定失败，请稍后重试")
	}
	logx.Infof("商家地图档口自动绑定成功: merchantId=%s objectId=%s requestId=%s", merchantID, objectID, request.ID)
	return SubmitMapBindRequestResp{Item: mapBindRequestItem(request)}, nil
}

func (l *BindingLogic) ListAdminRequests(ctx context.Context, req ListAdminMapBindRequestsReq) (ListAdminMapBindRequestsResp, error) {
	requests, err := l.store.ListMapBindRequests(ctx, model.ListMapBindRequestsFilter{
		Status:   strings.TrimSpace(req.Status),
		Keyword:  strings.TrimSpace(req.Keyword),
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		LogMapDependencyFailure(ctx, "后台查询地图档口绑定申请失败", "list_admin_binding_requests", err,
			logx.Field("status", strings.TrimSpace(req.Status)))
		return ListAdminMapBindRequestsResp{}, errx.New(errx.CodeInternalError, "绑定申请加载失败，请稍后重试")
	}
	return ListAdminMapBindRequestsResp{Items: mapBindRequestItems(requests)}, nil
}

func (l *BindingLogic) ReviewRequest(ctx context.Context, requestID string, req ReviewMapBindRequestReq) (ReviewMapBindRequestResp, error) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return ReviewMapBindRequestResp{}, errx.New(errx.CodeValidationFailed, "请选择绑定申请")
	}
	status, err := reviewActionStatus(req.Action)
	if err != nil {
		return ReviewMapBindRequestResp{}, err
	}
	reviewNote := strings.TrimSpace(req.ReviewNote)
	if status == model.MapBindRequestStatusRejected && reviewNote == "" {
		return ReviewMapBindRequestResp{}, errx.New(errx.CodeValidationFailed, "请填写驳回原因")
	}
	request, err := l.store.ReviewMapBindRequest(ctx, model.ReviewMapBindRequestInput{
		ID:         requestID,
		Status:     status,
		ReviewNote: reviewNote,
		ReviewerID: strings.TrimSpace(req.ReviewerID),
	})
	if err != nil {
		if errors.Is(err, model.ErrMapMerchantAlreadyBound) {
			return ReviewMapBindRequestResp{}, errx.New(errx.CodeStateConflict, "该商家已绑定其他主档口，请先处理原绑定")
		}
		if errors.Is(err, model.ErrMapObjectAlreadyBound) {
			return ReviewMapBindRequestResp{}, errx.New(errx.CodeStateConflict, "该档口已绑定其他商家，请先处理原绑定")
		}
		if errors.Is(err, sql.ErrNoRows) {
			return ReviewMapBindRequestResp{}, errx.New(errx.CodeResourceNotFound, "绑定申请不存在或已处理")
		}
		LogMapDependencyFailure(ctx, "后台审核地图档口绑定申请失败", "review_binding_request", err,
			logx.Field("requestId", requestID), logx.Field("action", strings.TrimSpace(req.Action)),
			logx.Field("reviewerId", strings.TrimSpace(req.ReviewerID)))
		return ReviewMapBindRequestResp{}, errx.New(errx.CodeInternalError, "绑定申请审核失败，请稍后重试")
	}
	logx.Infof("后台审核地图档口绑定申请成功: requestId=%s status=%s reviewerId=%s", requestID, status, strings.TrimSpace(req.ReviewerID))
	return ReviewMapBindRequestResp{Item: mapBindRequestItem(request)}, nil
}

func reviewActionStatus(action string) (string, error) {
	switch strings.TrimSpace(action) {
	case "approve":
		return model.MapBindRequestStatusApproved, nil
	case "reject":
		return model.MapBindRequestStatusRejected, nil
	default:
		return "", errx.New(errx.CodeValidationFailed, "请选择审核动作")
	}
}

func mapBindCandidateItems(items []model.MapBindCandidate) []MapBindCandidateItem {
	resp := make([]MapBindCandidateItem, 0, len(items))
	for _, item := range items {
		resp = append(resp, mapBindCandidateItem(item))
	}
	return resp
}

func mapBindCandidateItem(item model.MapBindCandidate) MapBindCandidateItem {
	return MapBindCandidateItem{
		ObjectID:     item.ObjectID,
		SceneCode:    item.SceneCode,
		SceneName:    item.SceneName,
		Code:         item.Code,
		Name:         item.Name,
		Address:      item.Address,
		MerchantID:   item.MerchantID,
		MerchantName: item.MerchantName,
		IsBound:      item.IsBound,
	}
}

func mapBindRequestItems(items []model.MapBindRequest) []MapBindRequestItem {
	resp := make([]MapBindRequestItem, 0, len(items))
	for _, item := range items {
		resp = append(resp, mapBindRequestItem(item))
	}
	return resp
}

func mapBindRequestItem(item model.MapBindRequest) MapBindRequestItem {
	return MapBindRequestItem{
		ID:              item.ID,
		MerchantID:      item.MerchantID,
		MerchantName:    item.MerchantName,
		ObjectID:        item.ObjectID,
		SceneCode:       item.SceneCode,
		SceneName:       item.SceneName,
		ObjectCode:      item.ObjectCode,
		ObjectName:      item.ObjectName,
		ApplicantUserID: item.ApplicantUserID,
		EvidenceImages:  nonNilStringSlice(item.EvidenceImages),
		Note:            item.Note,
		Status:          item.Status,
		ReviewNote:      item.ReviewNote,
		ReviewedBy:      item.ReviewedBy,
		ReviewedAt:      item.ReviewedAt,
		CreatedAt:       item.CreatedAt,
	}
}
