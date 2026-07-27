package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
	"tcm-history-ai/backend/user-service/internal/domain/repository"
)

type ApiKeyUseCase struct {
	apiKeyRepo repository.ApiKeyRepository
}

func NewApiKeyUseCase(apiKeyRepo repository.ApiKeyRepository) *ApiKeyUseCase {
	return &ApiKeyUseCase{
		apiKeyRepo: apiKeyRepo,
	}
}

func (uc *ApiKeyUseCase) Create(ctx context.Context, userID int64, in *dto.CreateApiKeyRequest) (*dto.RegenerateResponse, error) {
	if in == nil || in.Name == "" {
		return nil, errno.New(errno.InvalidParams, "name is required")
	}
	if userID <= 0 {
		return nil, errno.New(errno.InvalidParams, "user_id is required")
	}

	plainKey, err := generateApiKey()
	if err != nil {
		return nil, errno.Wrap(errno.InternalError, "generate api key", err)
	}

	keyHash := sha256Hash(plainKey)
	keyPrefix := plainKey[:8]

	scopesJSON, _ := json.Marshal(in.Scopes)
	if len(scopesJSON) == 0 {
		scopesJSON = []byte("[]")
	}

	var expiresAt *time.Time
	if in.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, in.ExpiresAt)
		if err != nil {
			return nil, errno.New(errno.InvalidParams, "invalid expires_at format")
		}
		expiresAt = &t
	}

	apiKey := &entity.ApiKey{
		ID:                 idgen.Next(),
		UserID:             userID,
		Name:               in.Name,
		KeyHash:            keyHash,
		KeyPrefix:          keyPrefix,
		Scopes:             scopesJSON,
		QuotaDaily:         in.QuotaDaily,
		QuotaMonthly:       in.QuotaMonthly,
		RateLimitPerMinute: in.RateLimitPerMinute,
		Status:             entity.ApiKeyStatusActive,
		ExpiresAt:          expiresAt,
	}

	if err := uc.apiKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, err
	}

	resp := toApiKeyResponse(apiKey)
	return &dto.RegenerateResponse{
		ApiKeyResponse: resp,
		PlainKey:       plainKey,
	}, nil
}

func (uc *ApiKeyUseCase) List(ctx context.Context, userID int64, page, pageSize int) (dto.ApiKeyListResponse, error) {
	if userID <= 0 {
		return dto.ApiKeyListResponse{}, errno.New(errno.InvalidParams, "user_id is required")
	}

	items, total, err := uc.apiKeyRepo.ListByUserID(ctx, userID, page, pageSize)
	if err != nil {
		return dto.ApiKeyListResponse{}, err
	}

	respItems := make([]dto.ApiKeyResponse, 0, len(items))
	for i := range items {
		respItems = append(respItems, toApiKeyResponse(&items[i]))
	}

	p := pagination.NewPage(page, pageSize, total, respItems)
	return dto.ApiKeyListResponse{
		Page:      p.Page,
		PageSize:  p.PageSize,
		Total:     p.Total,
		TotalPage: p.TotalPage,
		Items:     p.Items,
	}, nil
}

func (uc *ApiKeyUseCase) Get(ctx context.Context, userID, id int64) (*dto.ApiKeyResponse, error) {
	key, err := uc.apiKeyRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, errno.New(errno.NotFound, "api key not found: "+strconv.FormatInt(id, 10))
	}
	if key.UserID != userID {
		return nil, errno.New(errno.NotFound, "api key not found")
	}
	resp := toApiKeyResponse(key)
	return &resp, nil
}

func (uc *ApiKeyUseCase) Update(ctx context.Context, userID, id int64, in *dto.UpdateApiKeyRequest) (*dto.ApiKeyResponse, error) {
	key, err := uc.apiKeyRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, errno.New(errno.NotFound, "api key not found")
	}
	if key.UserID != userID {
		return nil, errno.New(errno.NotFound, "api key not found")
	}

	if in.Name != nil {
		key.Name = *in.Name
	}
	if in.Scopes != nil {
		scopesJSON, _ := json.Marshal(in.Scopes)
		key.Scopes = scopesJSON
	}
	if in.QuotaDaily != nil {
		key.QuotaDaily = *in.QuotaDaily
	}
	if in.QuotaMonthly != nil {
		key.QuotaMonthly = *in.QuotaMonthly
	}
	if in.RateLimitPerMinute != nil {
		key.RateLimitPerMinute = *in.RateLimitPerMinute
	}
	if in.Status != nil {
		if *in.Status != entity.ApiKeyStatusActive && *in.Status != entity.ApiKeyStatusDisabled && *in.Status != entity.ApiKeyStatusRevoked {
			return nil, errno.New(errno.InvalidParams, "invalid status")
		}
		key.Status = *in.Status
	}

	if err := uc.apiKeyRepo.Update(ctx, key); err != nil {
		return nil, err
	}

	resp := toApiKeyResponse(key)
	return &resp, nil
}

func (uc *ApiKeyUseCase) Delete(ctx context.Context, userID, id int64) error {
	key, err := uc.apiKeyRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if key == nil {
		return errno.New(errno.NotFound, "api key not found")
	}
	if key.UserID != userID {
		return errno.New(errno.NotFound, "api key not found")
	}
	return uc.apiKeyRepo.Delete(ctx, id)
}

func (uc *ApiKeyUseCase) Regenerate(ctx context.Context, userID, id int64) (*dto.RegenerateResponse, error) {
	key, err := uc.apiKeyRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, errno.New(errno.NotFound, "api key not found")
	}
	if key.UserID != userID {
		return nil, errno.New(errno.NotFound, "api key not found")
	}

	plainKey, err := generateApiKey()
	if err != nil {
		return nil, errno.Wrap(errno.InternalError, "generate api key", err)
	}

	key.KeyHash = sha256Hash(plainKey)
	key.KeyPrefix = plainKey[:8]
	key.Status = entity.ApiKeyStatusActive

	if err := uc.apiKeyRepo.Update(ctx, key); err != nil {
		return nil, err
	}

	resp := toApiKeyResponse(key)
	return &dto.RegenerateResponse{
		ApiKeyResponse: resp,
		PlainKey:       plainKey,
	}, nil
}

func (uc *ApiKeyUseCase) Validate(ctx context.Context, keyHash string) (*entity.ApiKey, error) {
	key, err := uc.apiKeyRepo.FindByKeyHash(ctx, keyHash)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, errno.New(errno.Unauthorized, "invalid api key")
	}
	if key.Status != entity.ApiKeyStatusActive {
		return nil, errno.New(errno.Unauthorized, "api key is not active")
	}
	if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
		return nil, errno.New(errno.Unauthorized, "api key expired")
	}
	return key, nil
}

func generateApiKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "sk_" + hex.EncodeToString(b), nil
}

func sha256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func toApiKeyResponse(k *entity.ApiKey) dto.ApiKeyResponse {
	var scopes []string
	if len(k.Scopes) > 0 {
		_ = json.Unmarshal(k.Scopes, &scopes)
	}
	if scopes == nil {
		scopes = []string{}
	}

	resp := dto.ApiKeyResponse{
		ID:                 k.ID,
		Name:               k.Name,
		KeyPrefix:          k.KeyPrefix,
		Scopes:             scopes,
		QuotaDaily:         k.QuotaDaily,
		QuotaMonthly:       k.QuotaMonthly,
		RateLimitPerMinute: k.RateLimitPerMinute,
		Status:             k.Status,
		CreatedAt:          k.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          k.UpdatedAt.Format(time.RFC3339),
	}
	if k.LastUsedAt != nil {
		resp.LastUsedAt = k.LastUsedAt.Format(time.RFC3339)
	}
	if k.ExpiresAt != nil {
		resp.ExpiresAt = k.ExpiresAt.Format(time.RFC3339)
	}
	return resp
}
