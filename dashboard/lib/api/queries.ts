export const queryKeys = {
  projects: {
    all: ["projects"] as const,
    list: () => [...queryKeys.projects.all, "list"] as const,
    detail: (id: string) => [...queryKeys.projects.all, id] as const,
  },
  functions: {
    all: ["functions"] as const,
    list: (projectId: string) =>
      [...queryKeys.functions.all, "list", projectId] as const,
    detail: (id: string) => [...queryKeys.functions.all, id] as const,
  },
  invocations: {
    all: ["invocations"] as const,
    list: (projectId: string, functionId: string) =>
      [...queryKeys.invocations.all, "list", projectId, functionId] as const,
  },
  events: {
    all: ["events"] as const,
    list: (projectId: string) =>
      [...queryKeys.events.all, "list", projectId] as const,
  },
  secrets: {
    all: ["secrets"] as const,
    list: (projectId: string) =>
      [...queryKeys.secrets.all, "list", projectId] as const,
  },
  database: {
    all: ["database"] as const,
    tables: (projectId: string) =>
      [...queryKeys.database.all, "tables", projectId] as const,
    table: (projectId: string, tableId: number) =>
      [...queryKeys.database.all, "table", projectId, tableId] as const,
    columns: (projectId: string, tableId: number) =>
      [...queryKeys.database.all, "columns", projectId, tableId] as const,
    policies: (projectId: string) =>
      [...queryKeys.database.all, "policies", projectId] as const,
    tableData: (projectId: string, table: string) =>
      [...queryKeys.database.all, "data", projectId, table] as const,
    rowCount: (projectId: string, table: string) =>
      [...queryKeys.database.all, "count", projectId, table] as const,
  },
  storage: {
    all: ["storage"] as const,
    buckets: (projectId: string) =>
      [...queryKeys.storage.all, "buckets", projectId] as const,
    objects: (projectId: string, bucket: string) =>
      [...queryKeys.storage.all, "objects", projectId, bucket] as const,
  },
  tenant: {
    profile: ["tenant", "profile"] as const,
  },
} as const;
