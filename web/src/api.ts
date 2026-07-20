export type ModuleCard = {
  key: string;
  name: string;
  description: string;
  status: string;
  capabilities: string[];
};

export type DashboardSummary = {
  modulesTotal: number;
  modulesInProgress: number;
  focusAreas: string[];
  platformMode: string;
  gatingModel: string;
};

export type RulesBootstrap = {
  ruleTypes: Array<{
    key: string;
    label: string;
    patternHint: string;
    example: string;
    supportsNoArg: boolean;
  }>;
  policies: string[];
  defaultRules: string[];
  templateIds: string[];
  editorFeatures: string[];
};

export type TemplateRecord = {
  id: string;
  name: string;
  kind: string;
  description: string;
  variables: string[];
  content: string;
  locked: boolean;
};

export type NodeRecord = {
  id: string;
  name: string;
  sourceKind: string;
  protocol: string;
  serverHost: string;
  serverPort: number;
  tags: string[];
  metadata: Record<string, unknown>;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
};

export type RuleRecord = {
  id: string;
  ruleType: string;
  pattern: string;
  policy: string;
  sortOrder: number;
  enabled: boolean;
  note: string;
};

export type RuleSetRecord = {
  id: string;
  name: string;
  scope: string;
  description: string;
  createdAt: string;
  updatedAt: string;
  rules: RuleRecord[];
};

export type SubscriptionRecord = {
  id: string;
  name: string;
  ownerUserId: string;
  outputFormat: string;
  templateId: string;
  sources: string[];
  options: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
};

export type RenderedSubscription = {
  subscriptionId: string;
  name: string;
  outputFormat: string;
  templateId: string;
  content: string;
  fileName: string;
  contentType: string;
};

export type AppBootstrap = {
  modules: ModuleCard[];
  dashboard: DashboardSummary;
  rules: RulesBootstrap;
  templates: TemplateRecord[];
  nodes: NodeRecord[];
  ruleSets: RuleSetRecord[];
  subscriptions: SubscriptionRecord[];
};

async function fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    headers: {
      "Content-Type": "application/json",
    },
    ...init,
  });

  if (!response.ok) {
    let message = `Request failed for ${path}: ${response.status}`;
    try {
      const body = (await response.json()) as { error?: string };
      if (body.error) {
        message = body.error;
      }
    } catch {
      // Ignore JSON parse failures for non-JSON error bodies.
    }
    throw new Error(message);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
}

export async function loadWorkspace(): Promise<AppBootstrap> {
  const [modules, dashboard, rules, templates, nodes, ruleSets, subscriptions] = await Promise.all([
    fetchJSON<ModuleCard[]>("/api/v1/catalog/modules"),
    fetchJSON<DashboardSummary>("/api/v1/dashboard/summary"),
    fetchJSON<RulesBootstrap>("/api/v1/rules/bootstrap"),
    fetchJSON<TemplateRecord[]>("/api/v1/templates"),
    fetchJSON<NodeRecord[]>("/api/v1/nodes"),
    fetchJSON<RuleSetRecord[]>("/api/v1/rulesets"),
    fetchJSON<SubscriptionRecord[]>("/api/v1/subscriptions"),
  ]);

  return { modules, dashboard, rules, templates, nodes, ruleSets, subscriptions };
}

export function createNode(input: {
  name: string;
  sourceKind: string;
  protocol: string;
  serverHost: string;
  serverPort: number;
  tags: string[];
  metadata: Record<string, unknown>;
  enabled: boolean;
}) {
  return fetchJSON<NodeRecord>("/api/v1/nodes", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function deleteNode(id: string) {
  return fetchJSON<void>(`/api/v1/nodes/${id}`, {
    method: "DELETE",
  });
}

export function createRuleSet(input: {
  name: string;
  scope: string;
  description: string;
  rules: Array<{
    ruleType: string;
    pattern: string;
    policy: string;
    sortOrder: number;
    enabled: boolean;
    note: string;
  }>;
}) {
  return fetchJSON<RuleSetRecord>("/api/v1/rulesets", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function createTemplate(input: {
  name: string;
  kind: string;
  description: string;
  variables: string[];
  content: string;
}) {
  return fetchJSON<TemplateRecord>("/api/v1/templates", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function createSubscription(input: {
  name: string;
  ownerUserId: string;
  outputFormat: string;
  templateId: string;
  sources: string[];
  options: Record<string, unknown>;
}) {
  return fetchJSON<SubscriptionRecord>("/api/v1/subscriptions", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function previewSubscription(id: string) {
  return fetchJSON<RenderedSubscription>(`/api/v1/subscriptions/${id}/preview`);
}

export function subscriptionDownloadURL(id: string) {
  return `/api/v1/subscriptions/${id}/download`;
}
