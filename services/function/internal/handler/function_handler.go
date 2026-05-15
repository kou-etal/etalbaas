package handler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/kou-etal/etalbaas/pkg/apperror"
	"github.com/kou-etal/etalbaas/pkg/requestctx"
	functionv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/function/v1"
	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/function/v1/functionv1connect"
	"github.com/kou-etal/etalbaas/services/function/internal/service"
)

type FunctionHandler struct {
	functionv1connect.UnimplementedFunctionServiceHandler
	svc *service.FunctionService
}

func NewFunctionHandler(svc *service.FunctionService) *FunctionHandler {
	return &FunctionHandler{svc: svc}
}

func (h *FunctionHandler) CreateFunction(ctx context.Context, req *connect.Request[functionv1.CreateFunctionRequest]) (*connect.Response[functionv1.CreateFunctionResponse], error) {
	userID, err := requestctx.UserID(ctx)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeUnauthenticated, "user id not found"))
	}

	msg := req.Msg

	sourceType, sourceConfig, storagePath, err := SourceFromProto(msg.GetGitSource(), msg.GetZipSource(), msg.GetInlineSource())
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeInvalidArgument, err.Error()))
	}

	runtimePreset, runtimeReqs, runtimeDockerfile := RuntimeFromProto(msg.GetPresetRuntime(), msg.GetCustomRuntime())

	triggersJSON, err := TriggersFromProto(msg.Triggers)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeInvalidArgument, err.Error()))
	}

	envVarsJSON, err := EnvVarsFromProto(msg.EnvVars)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeInvalidArgument, err.Error()))
	}

	row, err := h.svc.CreateFunction(ctx, service.CreateFunctionParams{
		TenantID:            userID,
		ProjectID:           msg.ProjectId,
		Name:                msg.Name,
		DisplayName:         msg.DisplayName,
		Kind:                msg.Kind,
		Mode:                msg.Mode,
		SourceType:          sourceType,
		SourceConfig:        sourceConfig,
		SourceStoragePath:   storagePath,
		RuntimePreset:       runtimePreset,
		RuntimeRequirements: runtimeReqs,
		RuntimeDockerfile:   runtimeDockerfile,
		TimeoutSec:          msg.TimeoutSec,
		GpuConfig:           GpuConfigFromProto(msg.GpuConfig),
		Triggers:            triggersJSON,
		EnvVars:             envVarsJSON,
	})
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	f, err := FunctionToProto(row)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.Wrap(apperror.CodeInternal, "convert function", err))
	}
	return connect.NewResponse(&functionv1.CreateFunctionResponse{Function: f}), nil
}

func (h *FunctionHandler) GetFunction(ctx context.Context, req *connect.Request[functionv1.GetFunctionRequest]) (*connect.Response[functionv1.GetFunctionResponse], error) {
	userID, err := requestctx.UserID(ctx)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeUnauthenticated, "user id not found"))
	}

	functionID, err := uuid.Parse(req.Msg.FunctionId)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeInvalidArgument, "invalid function_id"))
	}

	row, err := h.svc.GetFunction(ctx, userID, req.Msg.ProjectId, functionID)
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	f, err := FunctionToProto(row)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.Wrap(apperror.CodeInternal, "convert function", err))
	}
	return connect.NewResponse(&functionv1.GetFunctionResponse{Function: f}), nil
}

func (h *FunctionHandler) ListFunctions(ctx context.Context, req *connect.Request[functionv1.ListFunctionsRequest]) (*connect.Response[functionv1.ListFunctionsResponse], error) {
	userID, err := requestctx.UserID(ctx)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeUnauthenticated, "user id not found"))
	}

	pageSize, cursorCreatedAt, cursorID, err := UUIDPaginationFromProto(req.Msg.GetPagination())
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	rows, err := h.svc.ListFunctions(ctx, userID, req.Msg.ProjectId, pageSize+1, cursorCreatedAt, cursorID)
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	hasMore := int32(len(rows)) > pageSize
	if hasMore {
		rows = rows[:pageSize]
	}

	functions, nextToken, err := FunctionsToProto(rows, hasMore)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.Wrap(apperror.CodeInternal, "convert functions", err))
	}
	return connect.NewResponse(&functionv1.ListFunctionsResponse{
		Functions:  functions,
		Pagination: PaginationToProto(nextToken),
	}), nil
}

func (h *FunctionHandler) UpdateFunction(ctx context.Context, req *connect.Request[functionv1.UpdateFunctionRequest]) (*connect.Response[functionv1.UpdateFunctionResponse], error) {
	userID, err := requestctx.UserID(ctx)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeUnauthenticated, "user id not found"))
	}

	functionID, err := uuid.Parse(req.Msg.FunctionId)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeInvalidArgument, "invalid function_id"))
	}

	msg := req.Msg
	params := service.UpdateFunctionParams{
		TenantID:  userID,
		ProjectID: msg.ProjectId,
		ID:        functionID,
	}

	// Optional scalars
	params.DisplayName = msg.DisplayName
	params.Mode = msg.Mode
	params.TimeoutSec = msg.TimeoutSec

	// Source oneof
	if msg.Source != nil {
		switch src := msg.Source.(type) {
		case *functionv1.UpdateFunctionRequest_GitSource:
			st, sc, sp, err := SourceFromProto(src.GitSource, nil, nil)
			if err != nil {
				return nil, apperror.ToConnectError(apperror.New(apperror.CodeInvalidArgument, err.Error()))
			}
			params.SourceType = &st
			params.SourceConfig = sc
			params.SourceStoragePath = sp
		case *functionv1.UpdateFunctionRequest_ZipSource:
			st, sc, sp, err := SourceFromProto(nil, src.ZipSource, nil)
			if err != nil {
				return nil, apperror.ToConnectError(apperror.New(apperror.CodeInvalidArgument, err.Error()))
			}
			params.SourceType = &st
			params.SourceConfig = sc
			params.SourceStoragePath = sp
		case *functionv1.UpdateFunctionRequest_InlineSource:
			st, sc, sp, err := SourceFromProto(nil, nil, src.InlineSource)
			if err != nil {
				return nil, apperror.ToConnectError(apperror.New(apperror.CodeInvalidArgument, err.Error()))
			}
			params.SourceType = &st
			params.SourceConfig = sc
			params.SourceStoragePath = sp
		}
	}

	// Runtime oneof
	if msg.Runtime != nil {
		params.RuntimeSet = true
		switch rt := msg.Runtime.(type) {
		case *functionv1.UpdateFunctionRequest_PresetRuntime:
			p, reqs, _ := RuntimeFromProto(rt.PresetRuntime, nil)
			params.RuntimePreset = p
			params.RuntimeRequirements = reqs
		case *functionv1.UpdateFunctionRequest_CustomRuntime:
			_, _, d := RuntimeFromProto(nil, rt.CustomRuntime)
			params.RuntimeDockerfile = d
		}
	}

	// GpuConfig (optional)
	if msg.GpuConfig != nil {
		params.GpuConfigSet = true
		params.GpuConfig = GpuConfigFromProto(msg.GpuConfig)
	}

	// Repeated fields: only replace when explicitly provided (non-empty).
	// proto3 cannot distinguish "not sent" from "sent empty", so clients
	// cannot clear all triggers/env_vars via update. Use delete+recreate instead.
	if len(msg.Triggers) > 0 {
		triggersJSON, err := TriggersFromProto(msg.Triggers)
		if err != nil {
			return nil, apperror.ToConnectError(apperror.New(apperror.CodeInvalidArgument, err.Error()))
		}
		params.Triggers = triggersJSON
		params.TriggersSet = true
	}

	if len(msg.EnvVars) > 0 {
		envVarsJSON, err := EnvVarsFromProto(msg.EnvVars)
		if err != nil {
			return nil, apperror.ToConnectError(apperror.New(apperror.CodeInvalidArgument, err.Error()))
		}
		params.EnvVars = envVarsJSON
		params.EnvVarsSet = true
	}

	row, err := h.svc.UpdateFunction(ctx, params)
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	f, err := FunctionToProto(row)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.Wrap(apperror.CodeInternal, "convert function", err))
	}
	return connect.NewResponse(&functionv1.UpdateFunctionResponse{Function: f}), nil
}

func (h *FunctionHandler) DeleteFunction(ctx context.Context, req *connect.Request[functionv1.DeleteFunctionRequest]) (*connect.Response[functionv1.DeleteFunctionResponse], error) {
	userID, err := requestctx.UserID(ctx)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeUnauthenticated, "user id not found"))
	}

	functionID, err := uuid.Parse(req.Msg.FunctionId)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.New(apperror.CodeInvalidArgument, "invalid function_id"))
	}

	row, err := h.svc.DeleteFunction(ctx, userID, req.Msg.ProjectId, functionID)
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	f, err := FunctionToProto(row)
	if err != nil {
		return nil, apperror.ToConnectError(apperror.Wrap(apperror.CodeInternal, "convert function", err))
	}
	return connect.NewResponse(&functionv1.DeleteFunctionResponse{Function: f}), nil
}

func (h *FunctionHandler) ListInvocations(ctx context.Context, req *connect.Request[functionv1.ListInvocationsRequest]) (*connect.Response[functionv1.ListInvocationsResponse], error) {
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

	params := service.ListInvocationsParams{
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

	rows, err := h.svc.ListInvocations(ctx, params)
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	hasMore := int32(len(rows)) > pageSize
	if hasMore {
		rows = rows[:pageSize]
	}

	invocations, nextToken := InvocationsToProto(rows, hasMore)
	return connect.NewResponse(&functionv1.ListInvocationsResponse{
		Invocations: invocations,
		Pagination:  PaginationToProto(nextToken),
	}), nil
}
