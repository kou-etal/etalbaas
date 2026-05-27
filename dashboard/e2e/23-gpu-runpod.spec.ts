import { test, expect } from "@playwright/test";
import { execSync } from "node:child_process";
import {
  createTestProject,
  deleteTestProject,
  API_URL,
} from "./helpers/api";

// Allow self-signed certificates in Kind cluster
process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";

const KUBECTL_BIN =
  process.env.KUBECTL_PATH ||
  "C:\\Program Files\\Docker\\Docker\\resources\\bin\\kubectl.exe";

const RUNPOD_API_KEY = process.env.RUNPOD_API_KEY || "";
const RUNPOD_ENDPOINT_ID = process.env.RUNPOD_ENDPOINT_ID || "";

// Skip entire suite if RunPod credentials are not configured.
const skipGPU = !RUNPOD_API_KEY || !RUNPOD_ENDPOINT_ID;

/* ===== Helpers ===== */

function kubectl(args: string, timeoutMs = 30_000): string {
  return execSync(`"${KUBECTL_BIN}" ${args}`, {
    encoding: "utf-8",
    timeout: timeoutMs,
  }).trim();
}

function kubectlSafe(args: string): string {
  try {
    return kubectl(args);
  } catch {
    return "";
  }
}

/** kubectl exec with stdin piped — avoids Windows cmd.exe quoting issues for JSON payloads. */
function kubectlExecWithStdin(
  podSpec: string,
  command: string,
  stdinData: string,
  timeoutMs = 30_000,
): string {
  return execSync(`"${KUBECTL_BIN}" exec -i ${podSpec} -- ${command}`, {
    encoding: "utf-8",
    timeout: timeoutMs,
    input: stdinData,
  }).trim();
}

function waitFor(fn: () => boolean, timeoutMs = 60_000, intervalMs = 3_000): Promise<void> {
  return new Promise((resolve, reject) => {
    const start = Date.now();
    const check = () => {
      if (fn()) return resolve();
      if (Date.now() - start > timeoutMs) return reject(new Error("waitFor timed out"));
      setTimeout(check, intervalMs);
    };
    check();
  });
}

/* ===== State ===== */

let projectId = "";
const GPU_FUNC_NAME = "e2e-gpu-test";

test.describe.serial("GPU RunPod Serverless", () => {
  test.skip(skipGPU, "RUNPOD_API_KEY and RUNPOD_ENDPOINT_ID not set, skipping GPU tests");

  // GPU tests need long timeouts (RunPod cold start, project provisioning, etc.)
  test.setTimeout(300_000);

  test.beforeAll(async () => {
    // Create a test project for GPU functions.
    const project = await createTestProject("e2e-gpu-project");
    projectId = project.id;

    // Wait for project namespace to be ready.
    await waitFor(() => {
      const phase = kubectlSafe(
        `get project ${projectId} -n platform-system -o jsonpath="{.status.phase}"`,
      ).replace(/"/g, "");
      return phase === "Ready";
    }, 120_000);
  });

  test.afterAll(async () => {
    // Cleanup GPU function CRD.
    kubectlSafe(
      `delete function ${GPU_FUNC_NAME} -n project-${projectId} --ignore-not-found`,
    );
    // Cleanup dispatcher jobs.
    kubectlSafe(
      `delete jobs -n platform-system -l etalbaas.io/function=${GPU_FUNC_NAME} --ignore-not-found`,
    );
    // Delete test project.
    if (projectId) {
      await deleteTestProject(projectId);
    }
  });

  test("1: runpod-creds Secret and gpu-dispatcher SA exist", () => {
    const secret = kubectlSafe(
      `get secret runpod-creds -n platform-system -o jsonpath="{.metadata.name}"`,
    ).replace(/"/g, "");
    expect(secret).toBe("runpod-creds");

    const sa = kubectlSafe(
      `get sa gpu-dispatcher -n platform-system -o jsonpath="{.metadata.name}"`,
    ).replace(/"/g, "");
    expect(sa).toBe("gpu-dispatcher");
  });

  test("2: Create GPU function CRD and verify HTTPRoute", async () => {
    // Apply GPU function CR.
    const funcYaml = `
apiVersion: etalbaas.io/v1alpha1
kind: Function
metadata:
  name: ${GPU_FUNC_NAME}
  namespace: project-${projectId}
  labels:
    etalbaas.io/project-id: "${projectId}"
spec:
  displayName: "E2E GPU Test"
  projectRef:
    name: "${projectId}"
  kind: heavy-deployment
  source:
    type: inline
    inline:
      entrypoint: handler.py
      files:
        handler.py: |
          def handler(event):
              return {"output": "hello from gpu"}
  runtime:
    preset: python-3.11
  gpu:
    required: true
    provider: runpod
    product: serverless
    type: any
    providerConfig:
      endpoint_id: "${RUNPOD_ENDPOINT_ID}"
  triggers:
    - type: Http
`;
    // Write to temp file and apply.
    const fs = require("fs");
    const tmpFile = `${process.env.TEMP || "/tmp"}/e2e-gpu-func.yaml`;
    fs.writeFileSync(tmpFile, funcYaml);
    kubectl(`apply -f "${tmpFile}"`);

    // Wait for HTTPRoute to be created.
    await waitFor(() => {
      const routeName = kubectlSafe(
        `get httproute func-${GPU_FUNC_NAME} -n project-${projectId} -o jsonpath="{.metadata.name}"`,
      ).replace(/"/g, "");
      return routeName === `func-${GPU_FUNC_NAME}`;
    }, 120_000);

    // Verify HTTPRoute uses "set" (not "add") for RequestHeaderModifier.
    const routeJSON = kubectl(
      `get httproute func-${GPU_FUNC_NAME} -n project-${projectId} -o json`,
    );
    const route = JSON.parse(routeJSON);
    const filters = route.spec.rules[0].filters;
    const headerModifier = filters.find(
      (f: any) => f.type === "RequestHeaderModifier",
    );
    expect(headerModifier).toBeDefined();
    expect(headerModifier.requestHeaderModifier.set).toBeDefined();
    expect(headerModifier.requestHeaderModifier.add).toBeUndefined();

    // Verify cross-namespace backendRef points to function-ms in platform-system.
    const backend = route.spec.rules[0].backendRefs[0];
    expect(backend.name).toBe("function");
    expect(backend.namespace).toBe("platform-system");

    // Verify ReferenceGrant exists in platform-system.
    const grantName = kubectlSafe(
      `get referencegrant gpu-route-from-project-${projectId} -n platform-system -o jsonpath="{.metadata.name}"`,
    ).replace(/"/g, "");
    expect(grantName).toBe(`gpu-route-from-project-${projectId}`);

    // Verify BackendTrafficPolicy exists.
    const btpName = kubectlSafe(
      `get backendtrafficpolicy gpu-timeout-${GPU_FUNC_NAME} -n project-${projectId} -o jsonpath="{.metadata.name}"`,
    ).replace(/"/g, "");
    expect(btpName).toBe(`gpu-timeout-${GPU_FUNC_NAME}`);
  });

  test("3: HTTP invoke returns RunPod result", async () => {
    // Wait for function build to complete first.
    await waitFor(
      () => {
        const phase = kubectlSafe(
          `get function ${GPU_FUNC_NAME} -n project-${projectId} -o jsonpath="{.status.phase}"`,
        ).replace(/"/g, "");
        return phase === "Ready";
      },
      180_000,
      5_000,
    );

    // Wait for HTTPRoute to be accepted by Envoy.
    await waitFor(
      () => {
        const status = kubectlSafe(
          `get httproute func-${GPU_FUNC_NAME} -n project-${projectId} -o jsonpath="{.status.parents[0].conditions[?(@.type=='Accepted')].status}"`,
        ).replace(/"/g, "");
        return status === "True";
      },
      60_000,
      3_000,
    );

    // Invoke via curl from inside the cluster (nats-box pod) to avoid TLS SNI issues.
    // From within the cluster, CoreDNS resolves *.api.local.etalbaas.dev.
    // Use stdin (-d @-) to pipe JSON payload — avoids Windows cmd.exe quoting issues.
    const invokeURL = `https://${projectId}.api.local.etalbaas.dev/functions/${GPU_FUNC_NAME}/invoke`;
    const payload = JSON.stringify({ input: { prompt: "hello from e2e" } });
    const output = kubectlExecWithStdin(
      "deploy/nats-box -n platform-system",
      `curl -sk -X POST "${invokeURL}" -H "Content-Type: application/json" -d @- -w "\\n__HTTP_CODE__%{http_code}" --max-time 300`,
      payload,
      330_000,
    );
    const lines = output.split("\n");
    const httpCodeLine = lines.find((l: string) => l.includes("__HTTP_CODE__"));
    const httpCode = httpCodeLine ? httpCodeLine.replace("__HTTP_CODE__", "") : "0";
    const body = lines.filter((l: string) => !l.includes("__HTTP_CODE__")).join("\n");

    expect(parseInt(httpCode, 10)).toBe(200);
    const data = JSON.parse(body);
    expect(data.status).toBe("COMPLETED");
    expect(data.output).toBeDefined();
  });

  test("4: Dispatcher Job created and completed", () => {
    // Verify at least one dispatcher job exists for this function.
    const jobs = kubectl(
      `get jobs -n platform-system -l etalbaas.io/function=${GPU_FUNC_NAME} -o jsonpath="{.items[*].metadata.name}"`,
    ).replace(/"/g, "");
    expect(jobs.length).toBeGreaterThan(0);

    // Verify the most recent job succeeded.
    const jobNames = jobs.split(" ").filter(Boolean);
    const lastJob = jobNames[jobNames.length - 1];
    const succeeded = kubectl(
      `get job ${lastJob} -n platform-system -o jsonpath="{.status.succeeded}"`,
    ).replace(/"/g, "");
    expect(succeeded).toBe("1");
  });

  test("5: Invalid endpoint_id returns error gracefully", async () => {
    // Create a function with invalid endpoint.
    const invalidFuncName = "e2e-gpu-invalid";
    const funcYaml = `
apiVersion: etalbaas.io/v1alpha1
kind: Function
metadata:
  name: ${invalidFuncName}
  namespace: project-${projectId}
  labels:
    etalbaas.io/project-id: "${projectId}"
spec:
  displayName: "E2E GPU Invalid"
  projectRef:
    name: "${projectId}"
  kind: heavy-deployment
  source:
    type: inline
    inline:
      entrypoint: handler.py
      files:
        handler.py: |
          def handler(event):
              return {"output": "should not reach"}
  runtime:
    preset: python-3.11
  gpu:
    required: true
    provider: runpod
    product: serverless
    type: any
    providerConfig:
      endpoint_id: "invalid-endpoint-xxx"
  triggers:
    - type: Http
`;
    const fs = require("fs");
    const tmpFile = `${process.env.TEMP || "/tmp"}/e2e-gpu-invalid.yaml`;
    fs.writeFileSync(tmpFile, funcYaml);
    kubectl(`apply -f "${tmpFile}"`);

    // Wait for function to be ready (build must complete first).
    await waitFor(
      () => {
        const phase = kubectlSafe(
          `get function ${invalidFuncName} -n project-${projectId} -o jsonpath="{.status.phase}"`,
        ).replace(/"/g, "");
        return phase === "Ready";
      },
      180_000,
      5_000,
    );

    // Wait for HTTPRoute to be accepted.
    await waitFor(
      () => {
        const status = kubectlSafe(
          `get httproute func-${invalidFuncName} -n project-${projectId} -o jsonpath="{.status.parents[0].conditions[?(@.type=='Accepted')].status}"`,
        ).replace(/"/g, "");
        return status === "True";
      },
      60_000,
      3_000,
    );

    // Invoke from inside the cluster — should fail with error from RunPod (invalid endpoint).
    // Use stdin (-d @-) to pipe JSON payload — avoids Windows cmd.exe quoting issues.
    const invokeURL = `https://${projectId}.api.local.etalbaas.dev/functions/${invalidFuncName}/invoke`;
    const invalidPayload = JSON.stringify({ input: { prompt: "test" } });
    const output = kubectlExecWithStdin(
      "deploy/nats-box -n platform-system",
      `curl -sk -X POST "${invokeURL}" -H "Content-Type: application/json" -d @- -w "\\n__HTTP_CODE__%{http_code}" --max-time 300`,
      invalidPayload,
      330_000,
    );
    const lines = output.split("\n");
    const httpCodeLine = lines.find((l: string) => l.includes("__HTTP_CODE__"));
    const httpCode = parseInt(httpCodeLine ? httpCodeLine.replace("__HTTP_CODE__", "") : "0", 10);
    const body = lines.filter((l: string) => !l.includes("__HTTP_CODE__")).join("\n");

    // Should get an error response (502 from Function MS or dispatcher failure).
    expect(httpCode).toBeGreaterThanOrEqual(400);
    // Response may be JSON or plain text depending on the error path.
    let hasErrorInfo = false;
    try {
      const data = JSON.parse(body);
      hasErrorInfo = !!(data.error || data.status);
    } catch {
      // Plain-text error from Function MS (e.g. "GPU job failed or timed out: ...")
      hasErrorInfo = body.length > 0;
    }
    expect(hasErrorInfo).toBe(true);

    // Cleanup invalid function.
    kubectlSafe(
      `delete function ${invalidFuncName} -n project-${projectId} --ignore-not-found`,
    );
  });
});
