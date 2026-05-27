// @generated - stub file for type checking
// Run `npm run proto:gen` to generate real implementation from .proto files

export const ProjectService = {
  typeName: "etalbaas.project.v1.ProjectService",
  methods: {
    createProject: { name: "CreateProject" },
    listProjects: { name: "ListProjects" },
    getProject: { name: "GetProject" },
    deleteProject: { name: "DeleteProject" },
    pauseProject: { name: "PauseProject" },
    resumeProject: { name: "ResumeProject" },
    createApiKey: { name: "CreateApiKey" },
    listApiKeys: { name: "ListApiKeys" },
    revokeApiKey: { name: "RevokeApiKey" },
  },
} as const;
