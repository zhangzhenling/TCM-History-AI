package usecase

import (
	"context"
	"time"

	"tcm-history-ai/backend/ai-service/internal/application/dto"
	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/ai-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

type TokenUseCase struct {
	usageRepo repository.TokenUsageRepository
	quotaRepo repository.TokenQuotaRepository
}

func NewTokenUseCase(
	usageRepo repository.TokenUsageRepository,
	quotaRepo repository.TokenQuotaRepository,
) *TokenUseCase {
	return &TokenUseCase{
		usageRepo: usageRepo,
		quotaRepo: quotaRepo,
	}
}

func (uc *TokenUseCase) RecordUsage(ctx context.Context, userID, conversationID int64, model, provider string, promptTokens, completionTokens int, estimatedCostCents int64) error {
	if userID <= 0 {
		return errno.New(errno.InvalidParams, "user_id is required")
	}
	if promptTokens < 0 || completionTokens < 0 {
		return errno.New(errno.InvalidParams, "tokens cannot be negative")
	}

	totalTokens := promptTokens + completionTokens

	usage := &entity.TokenUsage{
		ID:                 idgen.Next(),
		UserID:             userID,
		ConversationID:     conversationID,
		Model:              model,
		Provider:           provider,
		PromptTokens:       promptTokens,
		CompletionTokens:   completionTokens,
		TotalTokens:        totalTokens,
		EstimatedCostCents: estimatedCostCents,
	}

	if err := uc.usageRepo.Create(ctx, usage); err != nil {
		return err
	}

	month := time.Now().Format("2006-01")
	_ = uc.quotaRepo.IncrementUsed(ctx, userID, month, totalTokens)

	return nil
}

func (uc *TokenUseCase) GetQuota(ctx context.Context, userID int64) (*dto.QuotaResponse, error) {
	if userID <= 0 {
		return nil, errno.New(errno.InvalidParams, "user_id is required")
	}

	month := time.Now().Format("2006-01")
	quota, err := uc.quotaRepo.FindOrCreate(ctx, userID, month)
	if err != nil {
		return nil, err
	}

	return toQuotaResponse(quota), nil
}

func (uc *TokenUseCase) GetUsageSummary(ctx context.Context, userID int64) (*dto.UsageSummaryResponse, error) {
	if userID <= 0 {
		return nil, errno.New(errno.InvalidParams, "user_id is required")
	}

	now := time.Now()

	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := now.AddDate(0, 0, -int(now.Weekday())).Truncate(24 * time.Hour)
	if now.Weekday() == time.Sunday {
		weekStart = now.AddDate(0, 0, -6).Truncate(24 * time.Hour)
	}
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	today, err := uc.sumUsageInRange(ctx, userID, todayStart, now)
	if err != nil {
		return nil, err
	}

	thisWeek, err := uc.sumUsageInRange(ctx, userID, weekStart, now)
	if err != nil {
		return nil, err
	}

	thisMonth, err := uc.sumUsageInRange(ctx, userID, monthStart, now)
	if err != nil {
		return nil, err
	}

	return &dto.UsageSummaryResponse{
		Today:     today,
		ThisWeek:  thisWeek,
		ThisMonth: thisMonth,
	}, nil
}

func (uc *TokenUseCase) CheckQuota(ctx context.Context, userID int64, requiredTokens int) (int64, error) {
	if userID <= 0 {
		return 0, errno.New(errno.InvalidParams, "user_id is required")
	}

	month := time.Now().Format("2006-01")
	available, err := uc.quotaRepo.CheckAvailable(ctx, userID, month)
	if err != nil {
		return 0, err
	}

	if available > 0 && int64(requiredTokens) > available {
		return available, errno.New(errno.RateLimited, "insufficient token quota")
	}

	return available, nil
}

func (uc *TokenUseCase) GetUsageRecords(ctx context.Context, userID int64, p pagination.Params) (dto.ListResponse[dto.TokenUsageResponse], error) {
	if userID <= 0 {
		return dto.ListResponse[dto.TokenUsageResponse]{}, errno.New(errno.InvalidParams, "user_id is required")
	}

	items, total, err := uc.usageRepo.ListByUser(ctx, userID, p)
	if err != nil {
		return dto.ListResponse[dto.TokenUsageResponse]{}, err
	}

	resp := make([]dto.TokenUsageResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toTokenUsageResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

func (uc *TokenUseCase) sumUsageInRange(ctx context.Context, userID int64, start, end time.Time) (dto.UsagePeriodSummary, error) {
	items, _, err := uc.usageRepo.ListByUser(ctx, userID, pagination.Params{Page: 1, PageSize: 10000})
	if err != nil {
		return dto.UsagePeriodSummary{}, err
	}

	var summary dto.UsagePeriodSummary
	for _, item := range items {
		if item.CreatedAt.Before(start) || item.CreatedAt.After(end) {
			continue
		}
		summary.TotalTokens += item.TotalTokens
		summary.PromptTokens += item.PromptTokens
		summary.CompletionTokens += item.CompletionTokens
		summary.Requests++
		summary.EstimatedCostCents += item.EstimatedCostCents
	}

	return summary, nil
}

func toTokenUsageResponse(u *entity.TokenUsage) *dto.TokenUsageResponse {
	return &dto.TokenUsageResponse{
		ID:                 u.ID,
		UserID:             u.UserID,
		ConversationID:     u.ConversationID,
		Model:              u.Model,
		Provider:           u.Provider,
		PromptTokens:       u.PromptTokens,
		CompletionTokens:   u.CompletionTokens,
		TotalTokens:        u.TotalTokens,
		EstimatedCostCents: u.EstimatedCostCents,
		CreatedAt:          u.CreatedAt.Format(time.RFC3339),
	}
}

func toQuotaResponse(q *entity.TokenQuota) *dto.QuotaResponse {
	return &dto.QuotaResponse{
		UserID:          q.UserID,
		Month:           q.Month,
		TotalTokens:     q.TotalTokens,
		UsedTokens:      q.UsedTokens,
		AvailableTokens: q.AvailableTokens,
		UpdatedAt:       q.UpdatedAt.Format(time.RFC3339),
	}
}
