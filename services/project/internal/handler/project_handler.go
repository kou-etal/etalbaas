package handler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/kou-etal/etalbaas/pkg/apperror"
	"github.com/kou-etal/etalbaas/pkg/requestctx"
	projectv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/project/v1"
	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/project/v1/projectv1connect"
	"github.com/kou-etal/etalbaas/services/project/internal/service"
)

type ProjectHandler struct {
	projectv1connect.UnimplementedProjectServiceHandler
	svc *service.ProjectService
}

func NewProjectHandler(svc *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

func (h *ProjectHandler) CreateProject(ctx context.Context, req *connect.Request[projectv1.CreateProjectRequest]) (*connect.Response[projectv1.CreateProjectResponse], error) {
	userID, err := requestctx.UserID(ctx)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeUnauthenticated, "user id not found"))
	}

	row, err := h.svc.CreateProject(ctx, service.CreateProjectParams{
		TenantID:           userID,
		DisplayName:        req.Msg.DisplayName,
		Description:        req.Msg.Description,
		PostgresEnabled:    req.Msg.PostgresEnabled,
		PostgresExtensions: req.Msg.PostgresExtensions,
		RedisEnabled:       req.Msg.RedisEnabled,
		PostgrestEnabled:   req.Msg.PostgrestEnabled,
	})
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	return connect.NewResponse(&projectv1.CreateProjectResponse{
		Project: ProjectToProto(row),
	}), nil
}

func (h *ProjectHandler) ListProjects(ctx context.Context, req *connect.Request[projectv1.ListProjectsRequest]) (*connect.Response[projectv1.ListProjectsResponse], error) {
	userID, err := requestctx.UserID(ctx)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeUnauthenticated, "user id not found"))
	}

	pageSize, cursorCreatedAt, cursorID, err := ProjectPaginationFromProto(req.Msg.GetPagination())
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	rows, err := h.svc.ListProjects(ctx, userID, pageSize+1, cursorCreatedAt, cursorID)
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	hasMore := int32(len(rows)) > pageSize
	if hasMore {
		rows = rows[:pageSize]
	}

	projects, nextToken := ProjectsToProto(rows, hasMore)
	return connect.NewResponse(&projectv1.ListProjectsResponse{
		Projects:   projects,
		Pagination: PaginationToProto(nextToken),
	}), nil
}

func (h *ProjectHandler) GetProject(ctx context.Context, req *connect.Request[projectv1.GetProjectRequest]) (*connect.Response[projectv1.GetProjectResponse], error) {
	userID, err := requestctx.UserID(ctx)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeUnauthenticated, "user id not found"))
	}

	row, err := h.svc.GetProject(ctx, userID, req.Msg.ProjectId)
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	return connect.NewResponse(&projectv1.GetProjectResponse{
		Project: ProjectToProto(row),
	}), nil
}

func (h *ProjectHandler) DeleteProject(ctx context.Context, req *connect.Request[projectv1.DeleteProjectRequest]) (*connect.Response[projectv1.DeleteProjectResponse], error) {
	userID, err := requestctx.UserID(ctx)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeUnauthenticated, "user id not found"))
	}

	row, err := h.svc.DeleteProject(ctx, userID, req.Msg.ProjectId)
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	return connect.NewResponse(&projectv1.DeleteProjectResponse{
		Project: ProjectToProto(row),
	}), nil
}

func (h *ProjectHandler) PauseProject(ctx context.Context, req *connect.Request[projectv1.PauseProjectRequest]) (*connect.Response[projectv1.PauseProjectResponse], error) {
	userID, err := requestctx.UserID(ctx)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeUnauthenticated, "user id not found"))
	}

	row, err := h.svc.PauseProject(ctx, userID, req.Msg.ProjectId)
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	return connect.NewResponse(&projectv1.PauseProjectResponse{
		Project: ProjectToProto(row),
	}), nil
}

func (h *ProjectHandler) ResumeProject(ctx context.Context, req *connect.Request[projectv1.ResumeProjectRequest]) (*connect.Response[projectv1.ResumeProjectResponse], error) {
	userID, err := requestctx.UserID(ctx)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeUnauthenticated, "user id not found"))
	}

	row, err := h.svc.ResumeProject(ctx, userID, req.Msg.ProjectId)
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	return connect.NewResponse(&projectv1.ResumeProjectResponse{
		Project: ProjectToProto(row),
	}), nil
}

func (h *ProjectHandler) CreateApiKey(ctx context.Context, req *connect.Request[projectv1.CreateApiKeyRequest]) (*connect.Response[projectv1.CreateApiKeyResponse], error) {
	userID, err := requestctx.UserID(ctx)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeUnauthenticated, "user id not found"))
	}

	result, err := h.svc.CreateApiKey(ctx, service.CreateApiKeyParams{
		ProjectID:     req.Msg.ProjectId,
		TenantID:      userID,
		Name:          req.Msg.Name,
		Role:          req.Msg.Role,
		ExpiresInDays: req.Msg.ExpiresInDays,
	})
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	return connect.NewResponse(&projectv1.CreateApiKeyResponse{
		ApiKey: ApiKeyToProto(result.ApiKey),
		RawKey: result.RawKey,
	}), nil
}

func (h *ProjectHandler) ListApiKeys(ctx context.Context, req *connect.Request[projectv1.ListApiKeysRequest]) (*connect.Response[projectv1.ListApiKeysResponse], error) {
	userID, err := requestctx.UserID(ctx)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeUnauthenticated, "user id not found"))
	}

	pageSize, cursorCreatedAt, cursorID, err := UUIDPaginationFromProto(req.Msg.GetPagination())
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	rows, err := h.svc.ListApiKeys(ctx, userID, req.Msg.ProjectId, pageSize+1, cursorCreatedAt, cursorID)
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	hasMore := int32(len(rows)) > pageSize
	if hasMore {
		rows = rows[:pageSize]
	}

	apiKeys, nextToken := ApiKeysToProto(rows, hasMore)
	return connect.NewResponse(&projectv1.ListApiKeysResponse{
		ApiKeys:    apiKeys,
		Pagination: PaginationToProto(nextToken),
	}), nil
}

func (h *ProjectHandler) RevokeApiKey(ctx context.Context, req *connect.Request[projectv1.RevokeApiKeyRequest]) (*connect.Response[projectv1.RevokeApiKeyResponse], error) {
	userID, err := requestctx.UserID(ctx)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeUnauthenticated, "user id not found"))
	}

	apiKeyID, err := uuid.Parse(req.Msg.ApiKeyId)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeInvalidArgument, "invalid api_key_id"))
	}

	row, err := h.svc.RevokeApiKey(ctx, userID, req.Msg.ProjectId, apiKeyID)
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	return connect.NewResponse(&projectv1.RevokeApiKeyResponse{
		ApiKey: ApiKeyToProto(row),
	}), nil
}
