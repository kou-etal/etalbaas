package handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/kou-etal/etalbaas/pkg/apperror"
	"github.com/kou-etal/etalbaas/pkg/requestctx"
	eventv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/event/v1"
	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/event/v1/eventv1connect"
	"github.com/kou-etal/etalbaas/services/event/internal/service"
)

type EventHandler struct {
	eventv1connect.UnimplementedEventServiceHandler
	svc *service.EventService
}

func NewEventHandler(svc *service.EventService) *EventHandler {
	return &EventHandler{svc: svc}
}

func (h *EventHandler) ListEventHistory(ctx context.Context, req *connect.Request[eventv1.ListEventHistoryRequest]) (*connect.Response[eventv1.ListEventHistoryResponse], error) {
	userID, err := requestctx.UserID(ctx)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeUnauthenticated, "user id not found"))
	}

	functionID, err := uuid.Parse(req.Msg.FunctionId)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeInvalidArgument, "invalid function_id"))
	}

	msg := req.Msg
	pageSize, cursorCreatedAt, cursorID, err := UUIDPaginationFromProto(msg.GetPagination())
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	params := service.ListEventHistoryParams{
		TenantID:        userID,
		ProjectID:       msg.ProjectId,
		FunctionID:      functionID,
		StatusFilter:    msg.StatusFilter,
		Limit:           pageSize + 1,
		CursorCreatedAt: cursorCreatedAt,
		CursorID:        cursorID,
	}

	if msg.Since != nil {
		t := msg.Since.AsTime()
		params.Since = &t
	}
	if msg.Until != nil {
		t := msg.Until.AsTime()
		params.Until = &t
	}

	rows, err := h.svc.ListEventHistory(ctx, params)
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	hasMore := int32(len(rows)) > pageSize
	if hasMore {
		rows = rows[:pageSize]
	}

	events, nextToken, err := EventRecordsToProto(rows, hasMore)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.Wrap(apperror.CodeInternal, "convert events", err))
	}
	return connect.NewResponse(&eventv1.ListEventHistoryResponse{
		Events:     events,
		Pagination: PaginationToProto(nextToken),
	}), nil
}

func (h *EventHandler) RetryEvent(_ context.Context, _ *connect.Request[eventv1.RetryEventRequest]) (*connect.Response[eventv1.RetryEventResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("RetryEvent is not implemented in Phase 1"))
}
