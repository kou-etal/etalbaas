package handler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/kou-etal/etalbaas/pkg/apperror"
	"github.com/kou-etal/etalbaas/pkg/requestctx"
	secretv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/secret/v1"
	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/secret/v1/secretv1connect"
	"github.com/kou-etal/etalbaas/services/project/internal/service"
)

type SecretHandler struct {
	secretv1connect.UnimplementedSecretServiceHandler
	svc *service.SecretService
}

func NewSecretHandler(svc *service.SecretService) *SecretHandler {
	return &SecretHandler{svc: svc}
}

func (h *SecretHandler) CreateSecret(ctx context.Context, req *connect.Request[secretv1.CreateSecretRequest]) (*connect.Response[secretv1.CreateSecretResponse], error) {
	userID, err := requestctx.UserID(ctx)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeUnauthenticated, "user id not found"))
	}

	row, err := h.svc.CreateSecret(ctx, service.CreateSecretParams{
		TenantID:    userID,
		ProjectID:   req.Msg.ProjectId,
		Name:        req.Msg.Name,
		Value:       req.Msg.Value,
		Description: req.Msg.Description,
	})
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	return connect.NewResponse(&secretv1.CreateSecretResponse{
		Secret: SecretMetadataToProto(row),
	}), nil
}

func (h *SecretHandler) ListSecrets(ctx context.Context, req *connect.Request[secretv1.ListSecretsRequest]) (*connect.Response[secretv1.ListSecretsResponse], error) {
	userID, err := requestctx.UserID(ctx)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeUnauthenticated, "user id not found"))
	}

	pageSize, cursorCreatedAt, cursorID, err := UUIDPaginationFromProto(req.Msg.GetPagination())
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	rows, err := h.svc.ListSecrets(ctx, userID, req.Msg.ProjectId, pageSize+1, cursorCreatedAt, cursorID)
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	hasMore := int32(len(rows)) > pageSize
	if hasMore {
		rows = rows[:pageSize]
	}

	secrets, nextToken := SecretsToProto(rows, hasMore)
	return connect.NewResponse(&secretv1.ListSecretsResponse{
		Secrets:    secrets,
		Pagination: PaginationToProto(nextToken),
	}), nil
}

func (h *SecretHandler) UpdateSecretValue(ctx context.Context, req *connect.Request[secretv1.UpdateSecretValueRequest]) (*connect.Response[secretv1.UpdateSecretValueResponse], error) {
	userID, err := requestctx.UserID(ctx)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeUnauthenticated, "user id not found"))
	}

	secretID, err := uuid.Parse(req.Msg.SecretId)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeInvalidArgument, "invalid secret_id"))
	}

	row, err := h.svc.UpdateSecretValue(ctx, userID, req.Msg.ProjectId, secretID, req.Msg.Value)
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	return connect.NewResponse(&secretv1.UpdateSecretValueResponse{
		Secret: SecretMetadataToProto(row),
	}), nil
}

func (h *SecretHandler) DeleteSecret(ctx context.Context, req *connect.Request[secretv1.DeleteSecretRequest]) (*connect.Response[secretv1.DeleteSecretResponse], error) {
	userID, err := requestctx.UserID(ctx)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeUnauthenticated, "user id not found"))
	}

	secretID, err := uuid.Parse(req.Msg.SecretId)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeInvalidArgument, "invalid secret_id"))
	}

	row, err := h.svc.DeleteSecret(ctx, userID, req.Msg.ProjectId, secretID)
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	return connect.NewResponse(&secretv1.DeleteSecretResponse{
		Secret: SecretMetadataToProto(row),
	}), nil
}
