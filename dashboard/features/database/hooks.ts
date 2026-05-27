"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "@/lib/api/queries";
import {
  pgMetaClient,
  postgrestDataClient,
  type PgTable,
  type PgColumn,
  type PgPolicy,
  type QueryResult,
} from "@/lib/api/clients";

// ─── Schema queries (postgres-meta) ─────────────────────────────────────────

export function useTables(projectId: string) {
  return useQuery<PgTable[]>({
    queryKey: queryKeys.database.tables(projectId),
    queryFn: () => pgMetaClient.listTables(projectId),
    enabled: !!projectId,
  });
}

export function useTable(projectId: string, tableId: number) {
  return useQuery<PgTable>({
    queryKey: queryKeys.database.table(projectId, tableId),
    queryFn: () => pgMetaClient.getTable(projectId, tableId),
    enabled: !!projectId && tableId > 0,
  });
}

export function useColumns(projectId: string, tableId: number) {
  return useQuery<PgColumn[]>({
    queryKey: queryKeys.database.columns(projectId, tableId),
    queryFn: async () => {
      const all = await pgMetaClient.listColumns(projectId, tableId);
      // postgres-meta may ignore table_id filter — filter client-side
      return all.filter((c) => c.table_id === tableId);
    },
    enabled: !!projectId && tableId > 0,
  });
}

export function usePolicies(projectId: string) {
  return useQuery<PgPolicy[]>({
    queryKey: queryKeys.database.policies(projectId),
    queryFn: () => pgMetaClient.listPolicies(projectId),
    enabled: !!projectId,
  });
}

// ─── Schema mutations (postgres-meta) ────────────────────────────────────────

export function useCreateTable(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; schema?: string; comment?: string }) =>
      pgMetaClient.createTable(projectId, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.database.tables(projectId) });
    },
  });
}

export function useUpdateTable(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ tableId, ...body }: { tableId: number; name?: string; rls_enabled?: boolean; comment?: string }) =>
      pgMetaClient.updateTable(projectId, tableId, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.database.tables(projectId) });
    },
  });
}

export function useDeleteTable(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (tableId: number) => pgMetaClient.deleteTable(projectId, tableId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.database.tables(projectId) });
    },
  });
}

export function useCreateColumn(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: { table_id: number; name: string; type: string; default_value?: string; is_nullable?: boolean; is_unique?: boolean; comment?: string }) =>
      pgMetaClient.createColumn(projectId, body),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.database.columns(projectId, variables.table_id) });
      queryClient.invalidateQueries({ queryKey: queryKeys.database.tables(projectId) });
    },
  });
}

export function useUpdateColumn(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ columnId, ...body }: { columnId: string; name?: string; type?: string; default_value?: string; is_nullable?: boolean; comment?: string }) =>
      pgMetaClient.updateColumn(projectId, columnId, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.database.all });
    },
  });
}

export function useDeleteColumn(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (columnId: string) => pgMetaClient.deleteColumn(projectId, columnId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.database.all });
    },
  });
}

// ─── RLS Policy mutations ────────────────────────────────────────────────────

export function useCreatePolicy(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; table: string; schema?: string; definition: string; check?: string; action?: string; command?: string; roles?: string[] }) =>
      pgMetaClient.createPolicy(projectId, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.database.policies(projectId) });
    },
  });
}

export function useUpdatePolicy(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ policyId, ...body }: { policyId: number; name?: string; definition?: string; check?: string; roles?: string[] }) =>
      pgMetaClient.updatePolicy(projectId, policyId, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.database.policies(projectId) });
    },
  });
}

export function useDeletePolicy(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (policyId: number) => pgMetaClient.deletePolicy(projectId, policyId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.database.policies(projectId) });
    },
  });
}

// ─── SQL Editor ──────────────────────────────────────────────────────────────

export function useExecuteQuery(projectId: string) {
  return useMutation<QueryResult, Error, string>({
    mutationFn: (query: string) => pgMetaClient.executeQuery(projectId, query),
  });
}

// ─── Table data queries (PostgREST) ──────────────────────────────────────────

export function useTableData(
  projectId: string,
  table: string,
  limit = 50,
  offset = 0,
) {
  return useQuery<Record<string, unknown>[]>({
    queryKey: [...queryKeys.database.tableData(projectId, table), limit, offset],
    queryFn: () => postgrestDataClient.getTableData(projectId, table, limit, offset),
    enabled: !!projectId && !!table,
  });
}

export function useRowCount(projectId: string, table: string) {
  return useQuery<number>({
    queryKey: queryKeys.database.rowCount(projectId, table),
    queryFn: () => postgrestDataClient.getRowCount(projectId, table),
    enabled: !!projectId && !!table,
  });
}

// ─── Table data mutations (PostgREST) ────────────────────────────────────────

export function useInsertRow(projectId: string, table: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Record<string, unknown>) =>
      postgrestDataClient.insertRow(projectId, table, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.database.tableData(projectId, table) });
      queryClient.invalidateQueries({ queryKey: queryKeys.database.rowCount(projectId, table) });
    },
  });
}

export function useUpdateRow(projectId: string, table: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ pkCol, pkVal, data }: { pkCol: string; pkVal: unknown; data: Record<string, unknown> }) =>
      postgrestDataClient.updateRow(projectId, table, pkCol, pkVal, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.database.tableData(projectId, table) });
    },
  });
}

export function useDeleteRow(projectId: string, table: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ pkCol, pkVal }: { pkCol: string; pkVal: unknown }) =>
      postgrestDataClient.deleteRow(projectId, table, pkCol, pkVal),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.database.tableData(projectId, table) });
      queryClient.invalidateQueries({ queryKey: queryKeys.database.rowCount(projectId, table) });
    },
  });
}
