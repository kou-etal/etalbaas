package handler

import (
	"context"

	"connectrpc.com/connect"

	"github.com/kou-etal/etalbaas/pkg/apperror"
	"github.com/kou-etal/etalbaas/pkg/requestctx"
	tenantv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/tenant/v1"
	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/tenant/v1/tenantv1connect"
	"github.com/kou-etal/etalbaas/services/tenant-user/internal/service"
)

type Handler struct {
	tenantv1connect.UnimplementedTenantServiceHandler
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetMe(ctx context.Context, req *connect.Request[tenantv1.GetMeRequest]) (*connect.Response[tenantv1.GetMeResponse], error) {
	userID, err := requestctx.UserID(ctx)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeUnauthenticated, "user id not found"))
	}

	row, err := h.svc.GetMe(ctx, userID)
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	return connect.NewResponse(&tenantv1.GetMeResponse{
		Tenant: TenantToProto(row),
	}), nil
}

func (h *Handler) UpdateProfile(ctx context.Context, req *connect.Request[tenantv1.UpdateProfileRequest]) (*connect.Response[tenantv1.UpdateProfileResponse], error) {
	userID, err := requestctx.UserID(ctx)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeUnauthenticated, "user id not found"))
	}

	row, err := h.svc.UpdateProfile(ctx, userID, req.Msg.DisplayName, req.Msg.AvatarUrl)
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	return connect.NewResponse(&tenantv1.UpdateProfileResponse{
		Tenant: TenantToProto(row),
	}), nil
}

func (h *Handler) ListTenants(ctx context.Context, req *connect.Request[tenantv1.ListTenantsRequest]) (*connect.Response[tenantv1.ListTenantsResponse], error) {
	pageSize, cursorCreatedAt, cursorID, err := PaginationFromProto(req.Msg.GetPagination())
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	rows, err := h.svc.ListTenants(ctx, pageSize+1, cursorCreatedAt, cursorID)
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	hasMore := int32(len(rows)) > pageSize
	if hasMore {
		rows = rows[:pageSize]
	}

	tenants, nextToken := TenantsToProto(rows, hasMore)
	return connect.NewResponse(&tenantv1.ListTenantsResponse{
		Tenants:    tenants,
		Pagination: PaginationToProto(nextToken),
	}), nil
}
