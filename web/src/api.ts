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

export type NodeImportResult = {
  created: NodeRecord[];
  skipped: string[];
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

export type RuleSetInput = {
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
};

export type ValidationResult = {
  valid: boolean;
  issues: Array<{
    path: string;
    message: string;
  }>;
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

export type XrayPreview = {
  content: string;
  summary: string;
};

export type RemoteServerRecord = {
  id: string;
  name: string;
  host: string;
  connectionMode: string;
  status: string;
  metadata: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
};

export type RemoteServerEnrollment = {
  server: RemoteServerRecord;
  serverToken: string;
  agentToken: string;
};

export type RemoteTaskRecord = {
  id: string;
  remoteServerId: string;
  taskKind: string;
  status: string;
  payload: Record<string, unknown>;
  outputText: string;
  createdAt: string;
  updatedAt: string;
};

export type ProxyGroupRecord = {
  id: string;
  name: string;
  groupKind: string;
  config: Record<string, unknown>;
  sortOrder: number;
  createdAt: string;
  updatedAt: string;
};

export type DNSProviderRecord = {
  id: string;
  providerKind: string;
  name: string;
  credentials: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
};

export type CertificateRecord = {
  id: string;
  name: string;
  domain: string;
  providerId: string;
  certPem: string;
  keyPem: string;
  autoRenew: boolean;
  autoDeploy: boolean;
  expiresAt: string;
  createdAt: string;
  updatedAt: string;
};

export type NotificationChannelRecord = {
  id: string;
  channelKind: string;
  name: string;
  config: Record<string, unknown>;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
};

export type BackupRecord = {
  id: string;
  backupKind: string;
  filePath: string;
  summary: string;
  createdAt: string;
};

export type SystemSettingRecord = {
  key: string;
  value: Record<string, unknown>;
  updatedAt: string;
};

export type TrafficSampleRecord = {
  id: string;
  sampleScope: string;
  scopeId: string;
  rxBytes: number;
  txBytes: number;
  rate: Record<string, unknown>;
  recordedAt: string;
};

export type AuthUser = {
  id: string;
  username: string;
  role: string;
  status: string;
  displayName: string;
};

export type UserRecord = AuthUser & {
  email: string;
  createdAt: string;
  updatedAt: string;
};

export type LoginResponse = {
  token: string;
  user: AuthUser;
};

export type AppBootstrap = {
  modules: ModuleCard[];
  dashboard: DashboardSummary;
  rules: RulesBootstrap;
  templates: TemplateRecord[];
  nodes: NodeRecord[];
  ruleSets: RuleSetRecord[];
  subscriptions: SubscriptionRecord[];
  remoteServers: RemoteServerRecord[];
  proxyGroups: ProxyGroupRecord[];
  dnsProviders: DNSProviderRecord[];
  certificates: CertificateRecord[];
  notificationChannels: NotificationChannelRecord[];
  backups: BackupRecord[];
  systemSettings: SystemSettingRecord[];
  trafficSamples: TrafficSampleRecord[];
  users: UserRecord[];
};

let authToken = localStorage.getItem("harborx_token") ?? "";

export function setAuthToken(token: string) {
  authToken = token;
  if (token) {
    localStorage.setItem("harborx_token", token);
  } else {
    localStorage.removeItem("harborx_token");
  }
}

export function getAuthToken() {
  return authToken;
}

async function fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  headers.set("Content-Type", "application/json");
  if (authToken) {
    headers.set("Authorization", `Bearer ${authToken}`);
  }

  const response = await fetch(path, {
    ...init,
    headers,
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

export function login(input: { username: string; password: string }) {
  return fetchJSON<LoginResponse>("/api/v1/auth/login", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function loadWorkspace(): Promise<AppBootstrap> {
  const [
    modules,
    dashboard,
    rules,
    templates,
    nodes,
    ruleSets,
    subscriptions,
    remoteServers,
    proxyGroups,
    dnsProviders,
    certificates,
    notificationChannels,
    backups,
    systemSettings,
    trafficSamples,
  ] = await Promise.all([
    fetchJSON<ModuleCard[]>("/api/v1/catalog/modules"),
    fetchJSON<DashboardSummary>("/api/v1/dashboard/summary"),
    fetchJSON<RulesBootstrap>("/api/v1/rules/bootstrap"),
    fetchJSON<TemplateRecord[]>("/api/v1/templates"),
    fetchJSON<NodeRecord[]>("/api/v1/nodes"),
    fetchJSON<RuleSetRecord[]>("/api/v1/rulesets"),
    fetchJSON<SubscriptionRecord[]>("/api/v1/subscriptions"),
    fetchJSON<RemoteServerRecord[]>("/api/v1/remote/servers"),
    fetchJSON<ProxyGroupRecord[]>("/api/v1/proxy-groups"),
    fetchJSON<DNSProviderRecord[]>("/api/v1/dns/providers"),
    fetchJSON<CertificateRecord[]>("/api/v1/certificates"),
    fetchJSON<NotificationChannelRecord[]>("/api/v1/notifications/channels"),
    fetchJSON<BackupRecord[]>("/api/v1/backups"),
    fetchJSON<SystemSettingRecord[]>("/api/v1/system/settings"),
    fetchJSON<TrafficSampleRecord[]>("/api/v1/traffic/samples"),
  ]);
  const users = authToken ? await fetchJSON<UserRecord[]>("/api/v1/users") : [];

  return {
    modules,
    dashboard,
    rules,
    templates,
    nodes,
    ruleSets,
    subscriptions,
    remoteServers,
    proxyGroups,
    dnsProviders,
    certificates,
    notificationChannels,
    backups,
    systemSettings,
    trafficSamples,
    users,
  };
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

export function importNodes(input: {
  content: string;
  sourceKind: string;
  tags: string[];
}) {
  return fetchJSON<NodeImportResult>("/api/v1/nodes/import", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function deleteNode(id: string) {
  return fetchJSON<void>(`/api/v1/nodes/${id}`, {
    method: "DELETE",
  });
}

export function updateNode(
  id: string,
  input: {
    name: string;
    sourceKind: string;
    protocol: string;
    serverHost: string;
    serverPort: number;
    tags: string[];
    metadata: Record<string, unknown>;
    enabled: boolean;
  },
) {
  return fetchJSON<NodeRecord>(`/api/v1/nodes/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export function createRuleSet(input: RuleSetInput) {
  return fetchJSON<RuleSetRecord>("/api/v1/rulesets", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function updateRuleSet(id: string, input: RuleSetInput) {
  return fetchJSON<RuleSetRecord>(`/api/v1/rulesets/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export function deleteRuleSet(id: string) {
  return fetchJSON<void>(`/api/v1/rulesets/${id}`, {
    method: "DELETE",
  });
}

export function validateRuleSet(input: RuleSetInput) {
  return fetchJSON<ValidationResult>("/api/v1/rulesets/validate", {
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

export function updateTemplate(
  id: string,
  input: {
    name: string;
    kind: string;
    description: string;
    variables: string[];
    content: string;
  },
) {
  return fetchJSON<TemplateRecord>(`/api/v1/templates/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export function deleteTemplate(id: string) {
  return fetchJSON<void>(`/api/v1/templates/${id}`, {
    method: "DELETE",
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

export function updateSubscription(
  id: string,
  input: {
    name: string;
    ownerUserId: string;
    outputFormat: string;
    templateId: string;
    sources: string[];
    options: Record<string, unknown>;
  },
) {
  return fetchJSON<SubscriptionRecord>(`/api/v1/subscriptions/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export function deleteSubscription(id: string) {
  return fetchJSON<void>(`/api/v1/subscriptions/${id}`, {
    method: "DELETE",
  });
}

export function previewSubscription(id: string) {
  return fetchJSON<RenderedSubscription>(`/api/v1/subscriptions/${id}/preview`);
}

export function subscriptionDownloadURL(id: string) {
  return `/api/v1/subscriptions/${id}/download`;
}

export function previewXray() {
  return fetchJSON<XrayPreview>("/api/v1/xray/preview");
}

export function createRemoteServer(input: {
  name: string;
  host: string;
  connectionMode: string;
  metadata: Record<string, unknown>;
}) {
  return fetchJSON<RemoteServerEnrollment>("/api/v1/remote/servers", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function updateRemoteServer(
  id: string,
  input: {
    name: string;
    host: string;
    connectionMode: string;
    status: string;
    metadata: Record<string, unknown>;
  },
) {
  return fetchJSON<RemoteServerRecord>(`/api/v1/remote/servers/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export function deleteRemoteServer(id: string) {
  return fetchJSON<void>(`/api/v1/remote/servers/${id}`, {
    method: "DELETE",
  });
}

export function listRemoteTasks(serverId: string) {
  return fetchJSON<RemoteTaskRecord[]>(`/api/v1/remote/servers/${serverId}/tasks`);
}

export function createRemoteTask(
  serverId: string,
  input: {
    taskKind: string;
    payload: Record<string, unknown>;
  },
) {
  return fetchJSON<RemoteTaskRecord>(`/api/v1/remote/servers/${serverId}/tasks`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function createProxyGroup(input: {
  name: string;
  groupKind: string;
  config: Record<string, unknown>;
  sortOrder: number;
}) {
  return fetchJSON<ProxyGroupRecord>("/api/v1/proxy-groups", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function deleteProxyGroup(id: string) {
  return fetchJSON<void>(`/api/v1/proxy-groups/${id}`, {
    method: "DELETE",
  });
}

export function createDNSProvider(input: {
  providerKind: string;
  name: string;
  credentials: Record<string, unknown>;
}) {
  return fetchJSON<DNSProviderRecord>("/api/v1/dns/providers", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function deleteDNSProvider(id: string) {
  return fetchJSON<void>(`/api/v1/dns/providers/${id}`, {
    method: "DELETE",
  });
}

export function createCertificate(input: {
  name: string;
  domain: string;
  providerId: string;
  certPem: string;
  keyPem: string;
  autoRenew: boolean;
  autoDeploy: boolean;
  expiresAt: string;
}) {
  return fetchJSON<CertificateRecord>("/api/v1/certificates", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function deleteCertificate(id: string) {
  return fetchJSON<void>(`/api/v1/certificates/${id}`, {
    method: "DELETE",
  });
}

export function createNotificationChannel(input: {
  channelKind: string;
  name: string;
  config: Record<string, unknown>;
  enabled: boolean;
}) {
  return fetchJSON<NotificationChannelRecord>("/api/v1/notifications/channels", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function deleteNotificationChannel(id: string) {
  return fetchJSON<void>(`/api/v1/notifications/channels/${id}`, {
    method: "DELETE",
  });
}

export function testNotificationChannel(id: string, input: { message: string }) {
  return fetchJSON<{ ok: boolean }>(`/api/v1/notifications/channels/${id}/test`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function createBackup(input: { backupKind: string; filePath: string; summary: string }) {
  return fetchJSON<BackupRecord>("/api/v1/backups", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function exportBackup(input: { backupKind: string; summary: string }) {
  return fetchJSON<BackupRecord>("/api/v1/backups/export", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function deleteBackup(id: string) {
  return fetchJSON<void>(`/api/v1/backups/${id}`, {
    method: "DELETE",
  });
}

export function upsertSystemSetting(key: string, input: { value: Record<string, unknown> }) {
  return fetchJSON<SystemSettingRecord>(`/api/v1/system/settings/${key}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export function deleteSystemSetting(key: string) {
  return fetchJSON<void>(`/api/v1/system/settings/${key}`, {
    method: "DELETE",
  });
}

export function createTrafficSample(input: {
  sampleScope: string;
  scopeId: string;
  rxBytes: number;
  txBytes: number;
  rate: Record<string, unknown>;
}) {
  return fetchJSON<TrafficSampleRecord>("/api/v1/traffic/samples", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function createUser(input: {
  username: string;
  password: string;
  role: string;
  displayName: string;
  email: string;
}) {
  return fetchJSON<UserRecord>("/api/v1/users", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function updateUser(
  id: string,
  input: {
    role: string;
    status: string;
    displayName: string;
    email: string;
    password: string;
  },
) {
  return fetchJSON<UserRecord>(`/api/v1/users/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export function deleteUser(id: string) {
  return fetchJSON<void>(`/api/v1/users/${id}`, {
    method: "DELETE",
  });
}
