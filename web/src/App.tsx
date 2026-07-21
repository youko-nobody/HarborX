import { useState, type FormEvent, type ReactNode } from "react";
import {
  listAgentLogs,
  listRemoteTaskLogs,
  listRemoteTasks,
  previewSubscription,
  previewXray,
  getAuthToken,
  login,
  setAuthToken,
  subscriptionDownloadURL,
  validateRuleSet,
  type AgentLogRecord,
  type AuthUser,
  type RemoteServerEnrollment,
  type RemoteTaskLogRecord,
  type RemoteTaskRecord,
  type RenderedSubscription,
  type RuleRecord,
  type RuleSetInput,
  type XrayPreview,
} from "./api";
import { useWorkspaceData } from "./useWorkspaceData";

const outputFormats = [
  "clash-meta",
  "surge",
  "loon",
  "quantumult-x",
  "shadowrocket",
  "sing-box",
  "stash",
  "surfboard",
  "v2ray",
];

const opsResourceKinds = [
  "xray-inbound",
  "traffic-collector",
  "certificate-automation",
  "nginx-fallback",
  "vps-maintenance",
  "external-subscription",
  "security-policy",
  "notification-automation",
];

const opsDefaultConfig: Record<string, string> = {
  "xray-inbound": '{"protocol":"vless","port":443,"network":"tcp","security":"reality","tag":"vless-reality-in"}',
  "traffic-collector": '{"statsEndpoint":"127.0.0.1:10085","scope":"server","scopeId":"local"}',
  "certificate-automation": '{"domain":"example.com","email":"","webroot":"/var/www/html"}',
  "nginx-fallback": '{"serverName":"example.com","listen":"80","root":"/var/www/html"}',
  "vps-maintenance": '{"maintenanceAction":"health-check"}',
  "external-subscription": '{"url":"https://example.com/sub.txt","outputPath":"/var/lib/harborx/external-subscription.txt"}',
  "security-policy": '{"disablePasswordSSH":false,"loginRateLimit":true,"auditSensitiveActions":true}',
  "notification-automation": '{"event":"daily-summary","thresholdPercent":80}',
};

type DraftRule = Omit<RuleRecord, "id"> & { id: string };

type ConsoleSection =
  | "dashboard"
  | "nodes"
  | "subscriptions"
  | "rules"
  | "templates"
  | "users"
  | "traffic"
  | "remote"
  | "xray"
  | "system";

const consoleNavItems: Array<{ key: ConsoleSection; label: string; helper: string }> = [
  { key: "dashboard", label: "总览", helper: "状态、模块与快捷入口" },
  { key: "nodes", label: "节点", helper: "录入、导入与开关管理" },
  { key: "subscriptions", label: "订阅", helper: "生成、下载与套餐绑定" },
  { key: "rules", label: "规则", helper: "Clash 可视化规则编辑" },
  { key: "templates", label: "模板", helper: "内置模板和私有模板库" },
  { key: "users", label: "用户", helper: "成员账号与会话权限" },
  { key: "traffic", label: "流量", helper: "采样记录与汇总视图" },
  { key: "remote", label: "远程", helper: "VPS 纳管、任务与日志" },
  { key: "xray", label: "Xray", helper: "预览、快照与运行模式" },
  { key: "system", label: "系统", helper: "证书、通知、备份与自动化" },
];

const emptyDraftRule = (): DraftRule => ({
  id: createClientId(),
  ruleType: "DOMAIN-SUFFIX",
  pattern: "google.com",
  policy: "Proxy",
  sortOrder: 1,
  enabled: true,
  note: "",
});

export function App() {
  const {
    data,
    loading,
    error,
    busy,
    refresh,
    createNode,
    importNodes,
    updateNode,
    createRuleSet,
    updateRuleSet,
    deleteRuleSet,
    createSubscription,
    createPackage,
    createEntitlement,
    deleteSubscription,
    deletePackage,
    deleteEntitlement,
    createTemplate,
    deleteTemplate,
    deleteNode,
    createRemoteServer,
    updateRemoteServer,
    deleteRemoteServer,
    createRemoteTask,
    saveXraySnapshot,
    restoreXraySnapshot,
    createXrayProfile,
    deleteXrayProfile,
    applyXrayProfile,
    createProxyGroup,
    deleteProxyGroup,
    createDNSProvider,
    deleteDNSProvider,
    createCertificate,
    deleteCertificate,
    createNotificationChannel,
    deleteNotificationChannel,
    testNotificationChannel,
    exportBackup,
    deleteBackup,
    upsertSystemSetting,
    deleteSystemSetting,
    createTrafficSample,
    createOpsResource,
    deleteOpsResource,
    executeOpsResource,
    createUser,
    updateUser,
    deleteUser,
  } = useWorkspaceData();

  const modules = data?.modules ?? [];
  const starterRules = data?.rules.defaultRules ?? [];
  const policyOptions = data?.rules.policies ?? [];
  const templates = data?.templates ?? [];
  const nodes = data?.nodes ?? [];
  const ruleSets = data?.ruleSets ?? [];
  const subscriptions = data?.subscriptions ?? [];
  const packages = data?.packages ?? [];
  const entitlements = data?.entitlements ?? [];
  const remoteServers = data?.remoteServers ?? [];
  const proxyGroups = data?.proxyGroups ?? [];
  const dnsProviders = data?.dnsProviders ?? [];
  const certificates = data?.certificates ?? [];
  const notificationChannels = data?.notificationChannels ?? [];
  const backups = data?.backups ?? [];
  const systemSettings = data?.systemSettings ?? [];
  const trafficSamples = data?.trafficSamples ?? [];
  const trafficRollups = data?.trafficRollups ?? [];
  const opsResources = data?.opsResources ?? [];
  const xraySnapshots = data?.xraySnapshots ?? [];
  const xrayProfiles = data?.xrayProfiles ?? [];
  const users = data?.users ?? [];
  const ruleTypes = data?.rules.ruleTypes ?? [];

  const [nodeName, setNodeName] = useState("");
  const [nodeHost, setNodeHost] = useState("");
  const [nodePort, setNodePort] = useState("443");
  const [nodeProtocol, setNodeProtocol] = useState("vless");
  const [nodeTags, setNodeTags] = useState("");
  const [nodeImportContent, setNodeImportContent] = useState("");
  const [nodeImportStatus, setNodeImportStatus] = useState<string | null>(null);

  const [ruleSetName, setRuleSetName] = useState("Default Route Set");
  const [ruleSetDescription, setRuleSetDescription] = useState("Created from the HarborX operator console.");
  const [editingRuleSetId, setEditingRuleSetId] = useState<string | null>(null);
  const [draftRules, setDraftRules] = useState<DraftRule[]>([emptyDraftRule()]);
  const [ruleValidation, setRuleValidation] = useState<string | null>(null);

  const [templateName, setTemplateName] = useState("Private Mobile Template");
  const [templateDescription, setTemplateDescription] = useState("Your own working template variant.");
  const [templateVariables, setTemplateVariables] = useState("subscription_name,rules,dns,proxy_groups");
  const [templateContent, setTemplateContent] = useState(
    "# HarborX template\nsubscription-name: {{ .SubscriptionName }}\nrules:\n{{ .Rules }}\n",
  );

  const [subscriptionName, setSubscriptionName] = useState("My Main Subscription");
  const [subscriptionFormat, setSubscriptionFormat] = useState("clash-meta");
  const [subscriptionTemplateId, setSubscriptionTemplateId] = useState("private-base-template");
  const [subscriptionSources, setSubscriptionSources] = useState("manual");
  const [packageName, setPackageName] = useState("Personal Unlimited");
  const [packageDescription, setPackageDescription] = useState("Private package without pro or license gates.");
  const [packageBandwidth, setPackageBandwidth] = useState("1099511627776");
  const [packageDevices, setPackageDevices] = useState("5");
  const [packageDuration, setPackageDuration] = useState("30");
  const [packageFeatures, setPackageFeatures] = useState("all-features,xray,subscriptions,remote-agent");
  const [entitlementUserId, setEntitlementUserId] = useState("local-admin");
  const [entitlementPackageId, setEntitlementPackageId] = useState("");
  const [entitlementExpiresAt, setEntitlementExpiresAt] = useState("");
  const [renderedSubscription, setRenderedSubscription] = useState<RenderedSubscription | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [xrayPreview, setXrayPreview] = useState<XrayPreview | null>(null);
  const [xrayError, setXrayError] = useState<string | null>(null);
  const [authUser, setAuthUser] = useState<AuthUser | null>(null);
  const [authUsername, setAuthUsername] = useState("admin");
  const [authPassword, setAuthPassword] = useState("");
  const [authError, setAuthError] = useState<string | null>(null);
  const [remoteName, setRemoteName] = useState("Tokyo Edge VPS");
  const [remoteHost, setRemoteHost] = useState("");
  const [remoteConnectionMode, setRemoteConnectionMode] = useState("pull");
  const [remoteTags, setRemoteTags] = useState("edge,xray");
  const [remoteEnrollment, setRemoteEnrollment] = useState<RemoteServerEnrollment | null>(null);
  const [remoteTasks, setRemoteTasks] = useState<Record<string, RemoteTaskRecord[]>>({});
  const [remoteAgentLogs, setRemoteAgentLogs] = useState<Record<string, AgentLogRecord[]>>({});
  const [remoteTaskLogs, setRemoteTaskLogs] = useState<Record<string, RemoteTaskLogRecord[]>>({});
  const [remoteTaskKind, setRemoteTaskKind] = useState("reload-config");
  const [remoteTaskPayload, setRemoteTaskPayload] = useState('{"service":"xray"}');
  const [remoteError, setRemoteError] = useState<string | null>(null);
  const [proxyGroupName, setProxyGroupName] = useState("Auto");
  const [proxyGroupKind, setProxyGroupKind] = useState("url-test");
  const [proxyGroupConfig, setProxyGroupConfig] = useState('{"interval":300,"url":"https://www.gstatic.com/generate_204"}');
  const [dnsProviderName, setDNSProviderName] = useState("Cloudflare Main");
  const [dnsProviderKind, setDNSProviderKind] = useState("cloudflare");
  const [dnsCredentials, setDNSCredentials] = useState('{"token":"replace-me"}');
  const [certificateName, setCertificateName] = useState("Wildcard Cert");
  const [certificateDomain, setCertificateDomain] = useState("*.example.com");
  const [notificationName, setNotificationName] = useState("Telegram Ops");
  const [notificationConfig, setNotificationConfig] = useState('{"botToken":"replace-me","chatId":"replace-me"}');
  const [backupPath, setBackupPath] = useState("data/backups/manual.harborx.json");
  const [settingKey, setSettingKey] = useState("ui.theme");
  const [settingValue, setSettingValue] = useState('{"theme":"sand","refreshSeconds":30}');
  const [trafficScope, setTrafficScope] = useState("server");
  const [trafficScopeID, setTrafficScopeID] = useState("local");
  const [trafficRX, setTrafficRX] = useState("0");
  const [trafficTX, setTrafficTX] = useState("0");
  const [opsError, setOpsError] = useState<string | null>(null);
  const [opsResourceKind, setOpsResourceKind] = useState("xray-inbound");
  const [opsResourceName, setOpsResourceName] = useState("VLESS Reality Inbound");
  const [opsRemoteServerId, setOpsRemoteServerId] = useState("");
  const [opsConfig, setOpsConfig] = useState(opsDefaultConfig["xray-inbound"]);
  const [opsAction, setOpsAction] = useState("");
  const [opsStatus, setOpsStatus] = useState<string | null>(null);
  const [restoredSnapshot, setRestoredSnapshot] = useState<string | null>(null);
  const [xrayProfileName, setXrayProfileName] = useState("Default External Xray");
  const [xrayProfileRemoteServerId, setXrayProfileRemoteServerId] = useState("");
  const [xrayProfileMode, setXrayProfileMode] = useState("external");
  const [xrayProfileBinary, setXrayProfileBinary] = useState("xray");
  const [xrayProfileConfigPath, setXrayProfileConfigPath] = useState("/usr/local/etc/xray/config.json");
  const [xrayProfileService, setXrayProfileService] = useState("xray");
  const [xrayApplyStatus, setXrayApplyStatus] = useState<string | null>(null);
  const [newUsername, setNewUsername] = useState("member");
  const [newUserPassword, setNewUserPassword] = useState("");
  const [newUserDisplayName, setNewUserDisplayName] = useState("Member");
  const [newUserEmail, setNewUserEmail] = useState("");
  const [userError, setUserError] = useState<string | null>(null);
  const [activeSection, setActiveSection] = useState<ConsoleSection>("dashboard");

  const isAuthenticated = Boolean(getAuthToken());
  const activeNavItem = consoleNavItems.find((item) => item.key === activeSection) ?? consoleNavItems[0];

  async function handleLogin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setAuthError(null);
    try {
      const response = await login({ username: authUsername, password: authPassword });
      setAuthToken(response.token);
      setAuthUser(response.user);
      setAuthPassword("");
      await refresh();
    } catch (loginError) {
      setAuthError(loginError instanceof Error ? loginError.message : "Login failed");
    }
  }

  function handleLogout() {
    setAuthToken("");
    setAuthUser(null);
    void refresh();
  }

  async function handleCreateNode(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await createNode({
      name: nodeName,
      sourceKind: "manual",
      protocol: nodeProtocol,
      serverHost: nodeHost,
      serverPort: Number(nodePort),
      tags: splitCSV(nodeTags),
      metadata: {},
      enabled: true,
    });
    setNodeName("");
    setNodeHost("");
    setNodePort("443");
    setNodeTags("");
  }

  async function handleImportNodes(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setNodeImportStatus(null);
    try {
      const result = await importNodes({
        content: nodeImportContent,
        sourceKind: "share-link-import",
        tags: splitCSV(nodeTags),
      });
      setNodeImportStatus(`已导入 ${result.created.length} 个节点，跳过 ${result.skipped.length} 条。`);
      setNodeImportContent("");
    } catch (importError) {
      setNodeImportStatus(importError instanceof Error ? importError.message : "Node import failed");
    }
  }

  async function handleCreateRuleSet(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const input = buildRuleSetInput();
    const validation = await validateRuleSet(input);
    if (!validation.valid) {
      setRuleValidation(validation.issues.map((issue) => `${issue.path}: ${issue.message}`).join("\n"));
      return;
    }
    setRuleValidation(null);
    if (editingRuleSetId) {
      await updateRuleSet(editingRuleSetId, input);
      resetRuleEditor();
      return;
    }
    await createRuleSet(input);
  }

  function buildRuleSetInput(): RuleSetInput {
    return {
      name: ruleSetName,
      scope: "global",
      description: ruleSetDescription,
      rules: draftRules.map((rule, index) => ({
        ruleType: rule.ruleType,
        pattern: rule.ruleType === "MATCH" ? "" : rule.pattern,
        policy: rule.policy,
        sortOrder: index + 1,
        enabled: rule.enabled,
        note: rule.note,
      })),
    };
  }

  function resetRuleEditor() {
    setEditingRuleSetId(null);
    setRuleSetName("Default Route Set");
    setRuleSetDescription("Created from the HarborX operator console.");
    setDraftRules([emptyDraftRule()]);
    setRuleValidation(null);
  }

  function editRuleSet(item: { id: string; name: string; description: string; rules: RuleRecord[] }) {
    setEditingRuleSetId(item.id);
    setRuleSetName(item.name);
    setRuleSetDescription(item.description);
    setDraftRules(
      item.rules.length ? item.rules.map((rule) => ({ ...rule, id: rule.id || createClientId() })) : [emptyDraftRule()],
    );
    setRuleValidation(null);
    setActiveSection("rules");
  }

  function updateDraftRule(id: string, patch: Partial<DraftRule>) {
    setDraftRules((current) => current.map((rule) => (rule.id === id ? { ...rule, ...patch } : rule)));
  }

  function moveDraftRule(id: string, direction: -1 | 1) {
    setDraftRules((current) => {
      const index = current.findIndex((rule) => rule.id === id);
      const nextIndex = index + direction;
      if (index < 0 || nextIndex < 0 || nextIndex >= current.length) {
        return current;
      }
      const copied = [...current];
      const [item] = copied.splice(index, 1);
      copied.splice(nextIndex, 0, item);
      return copied.map((rule, ruleIndex) => ({ ...rule, sortOrder: ruleIndex + 1 }));
    });
  }

  function removeDraftRule(id: string) {
    setDraftRules((current) => {
      const filtered = current.filter((rule) => rule.id !== id);
      return (filtered.length ? filtered : [emptyDraftRule()]).map((rule, index) => ({ ...rule, sortOrder: index + 1 }));
    });
  }

  async function handleCreateTemplate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await createTemplate({
      name: templateName,
      kind: "private",
      description: templateDescription,
      variables: splitCSV(templateVariables),
      content: templateContent,
    });
  }

  async function handleCreateSubscription(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await createSubscription({
      name: subscriptionName,
      ownerUserId: "local-admin",
      outputFormat: subscriptionFormat,
      templateId: subscriptionTemplateId,
      sources: splitCSV(subscriptionSources),
      options: {},
    });
  }

  async function handleCreatePackage(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await runOps(() =>
      createPackage({
        name: packageName,
        description: packageDescription,
        bandwidthBytes: Number(packageBandwidth),
        deviceLimit: Number(packageDevices),
        durationDays: Number(packageDuration),
        features: splitCSV(packageFeatures),
        enabled: true,
      }),
    );
  }

  async function handleCreateEntitlement(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await runOps(() =>
      createEntitlement({
        userId: entitlementUserId,
        packageId: entitlementPackageId || packages[0]?.id || "",
        status: "active",
        expiresAt: entitlementExpiresAt,
        metadata: {},
      }),
    );
  }

  async function handlePreviewSubscription(id: string) {
    setPreviewError(null);
    try {
      setRenderedSubscription(await previewSubscription(id));
    } catch (previewSubscriptionError) {
      setPreviewError(previewSubscriptionError instanceof Error ? previewSubscriptionError.message : "Failed to render subscription");
    }
  }

  async function handlePreviewXray() {
    setXrayError(null);
    try {
      setXrayPreview(await previewXray());
    } catch (previewXrayError) {
      setXrayError(previewXrayError instanceof Error ? previewXrayError.message : "Failed to preview Xray config");
    }
  }

  async function handleSaveXraySnapshot() {
    setXrayError(null);
    try {
      await saveXraySnapshot({ targetKind: "local", targetId: "default" });
      await refresh();
    } catch (snapshotError) {
      setXrayError(snapshotError instanceof Error ? snapshotError.message : "Failed to save Xray snapshot");
    }
  }

  async function handleRestoreXraySnapshot(id: string) {
    setXrayError(null);
    try {
      const snapshot = await restoreXraySnapshot(id);
      setRestoredSnapshot(snapshot.config);
    } catch (restoreError) {
      setXrayError(restoreError instanceof Error ? restoreError.message : "Failed to restore Xray snapshot");
    }
  }

  async function handleCreateXrayProfile(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setXrayError(null);
    try {
      await createXrayProfile({
        name: xrayProfileName,
        remoteServerId: xrayProfileRemoteServerId,
        runtimeMode: xrayProfileMode,
        binaryPath: xrayProfileBinary,
        configPath: xrayProfileConfigPath,
        serviceName: xrayProfileService,
        metadata: {},
        enabled: true,
      });
    } catch (createProfileError) {
      setXrayError(createProfileError instanceof Error ? createProfileError.message : "Failed to create Xray profile");
    }
  }

  async function handleApplyXrayProfile(id: string, dryRun: boolean) {
    setXrayError(null);
    setXrayApplyStatus(null);
    try {
      const result = await applyXrayProfile(id, {
        dryRun,
        targetKind: "profile",
        targetId: id,
      });
      setXrayApplyStatus(dryRun ? `Dry-run rendered ${result.summary}.` : `Queued ${result.runtimeMode} Xray apply task ${result.taskId}.`);
      setXrayPreview({ content: result.config, summary: result.summary });
    } catch (applyError) {
      setXrayError(applyError instanceof Error ? applyError.message : "Failed to apply Xray profile");
    }
  }

  async function handleCreateRemoteServer(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setRemoteError(null);
    try {
      const enrollment = await createRemoteServer({
        name: remoteName,
        host: remoteHost,
        connectionMode: remoteConnectionMode,
        metadata: { tags: splitCSV(remoteTags) },
      });
      setRemoteEnrollment(enrollment);
      setRemoteName("");
      setRemoteHost("");
      setRemoteTags("");
    } catch (createRemoteError) {
      setRemoteError(createRemoteError instanceof Error ? createRemoteError.message : "Failed to create remote server");
    }
  }

  async function handleSetRemoteStatus(id: string, status: string) {
    const item = remoteServers.find((server) => server.id === id);
    if (!item) {
      return;
    }
    await updateRemoteServer(id, {
      name: item.name,
      host: item.host,
      connectionMode: item.connectionMode,
      status,
      metadata: item.metadata,
    });
  }

  async function handleLoadRemoteTasks(serverId: string) {
    setRemoteError(null);
    try {
      const items = await listRemoteTasks(serverId);
      setRemoteTasks((current) => ({ ...current, [serverId]: items }));
    } catch (tasksError) {
      setRemoteError(tasksError instanceof Error ? tasksError.message : "Failed to load remote tasks");
    }
  }

  async function handleLoadRemoteLogs(serverId: string) {
    setRemoteError(null);
    try {
      const agentLogs = await listAgentLogs(serverId);
      setRemoteAgentLogs((current) => ({ ...current, [serverId]: agentLogs }));
    } catch (logsError) {
      setRemoteError(logsError instanceof Error ? logsError.message : "Failed to load remote logs");
    }
  }

  async function handleLoadRemoteTaskLogs(serverId: string, taskId: string) {
    setRemoteError(null);
    try {
      const items = await listRemoteTaskLogs(serverId, taskId);
      setRemoteTaskLogs((current) => ({ ...current, [`${serverId}:${taskId}`]: items }));
    } catch (taskLogsError) {
      setRemoteError(taskLogsError instanceof Error ? taskLogsError.message : "Failed to load remote task logs");
    }
  }

  async function handleCreateRemoteTask(serverId: string) {
    setRemoteError(null);
    try {
      const payload = parseJSONObject(remoteTaskPayload);
      await createRemoteTask(serverId, {
        taskKind: remoteTaskKind,
        payload,
      });
      await handleLoadRemoteTasks(serverId);
    } catch (createTaskError) {
      setRemoteError(createTaskError instanceof Error ? createTaskError.message : "Failed to create remote task");
    }
  }

  async function handleCreateProxyGroup(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await runOps(() =>
      createProxyGroup({
        name: proxyGroupName,
        groupKind: proxyGroupKind,
        config: parseJSONObject(proxyGroupConfig),
        sortOrder: proxyGroups.length + 1,
      }),
    );
  }

  async function handleCreateDNSProvider(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await runOps(() =>
      createDNSProvider({
        name: dnsProviderName,
        providerKind: dnsProviderKind,
        credentials: parseJSONObject(dnsCredentials),
      }),
    );
  }

  async function handleCreateCertificate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await runOps(() =>
      createCertificate({
        name: certificateName,
        domain: certificateDomain,
        providerId: dnsProviders[0]?.id ?? "",
        certPem: "",
        keyPem: "",
        autoRenew: true,
        autoDeploy: true,
        expiresAt: "",
      }),
    );
  }

  async function handleCreateNotification(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await runOps(() =>
      createNotificationChannel({
        name: notificationName,
        channelKind: "telegram",
        config: parseJSONObject(notificationConfig),
        enabled: true,
      }),
    );
  }

  async function handleCreateBackup(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await runOps(() =>
      exportBackup({
        backupKind: "database",
        summary: "Exported from the HarborX console.",
      }),
    );
  }

  async function handleUpsertSetting(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await runOps(() => upsertSystemSetting(settingKey, { value: parseJSONObject(settingValue) }));
  }

  async function handleCreateTrafficSample(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await runOps(() =>
      createTrafficSample({
        sampleScope: trafficScope,
        scopeId: trafficScopeID,
        rxBytes: Number(trafficRX),
        txBytes: Number(trafficTX),
        rate: {},
      }),
    );
  }

  function handleOpsKindChange(kind: string) {
    setOpsResourceKind(kind);
    setOpsConfig(opsDefaultConfig[kind] ?? "{}");
    setOpsResourceName(kind.replaceAll("-", " "));
  }

  async function handleCreateOpsResource(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await runOps(async () => {
      await createOpsResource({
        resourceKind: opsResourceKind,
        name: opsResourceName,
        remoteServerId: opsRemoteServerId,
        status: "active",
        config: parseJSONObject(opsConfig),
        enabled: true,
      });
      setOpsStatus("Advanced resource saved.");
    });
  }

  async function handleExecuteOpsResource(id: string, dryRun: boolean) {
    await runOps(async () => {
      const result = await executeOpsResource(id, {
        action: opsAction,
        dryRun,
        config: {},
      });
      setOpsStatus(dryRun ? `Dry-run prepared ${result.taskKind}.` : `Queued ${result.taskKind} task ${result.taskId}.`);
    });
  }

  async function runOps(action: () => Promise<unknown>) {
    setOpsError(null);
    setOpsStatus(null);
    try {
      await action();
    } catch (operationError) {
      setOpsError(operationError instanceof Error ? operationError.message : "Operation failed");
    }
  }

  async function handleCreateUser(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setUserError(null);
    try {
      await createUser({
        username: newUsername,
        password: newUserPassword,
        role: "member",
        displayName: newUserDisplayName,
        email: newUserEmail,
      });
      setNewUsername("member");
      setNewUserPassword("");
      setNewUserDisplayName("Member");
      setNewUserEmail("");
    } catch (createUserError) {
      setUserError(createUserError instanceof Error ? createUserError.message : "Failed to create user");
    }
  }

  async function handleToggleUserStatus(id: string) {
    const item = users.find((user) => user.id === id);
    if (!item) {
      return;
    }
    setUserError(null);
    try {
      await updateUser(id, {
        role: item.role,
        status: item.status === "active" ? "disabled" : "active",
        displayName: item.displayName,
        email: item.email,
        password: "",
      });
    } catch (toggleUserError) {
      setUserError(toggleUserError instanceof Error ? toggleUserError.message : "Failed to update user");
    }
  }

  const totalTrafficBytes = trafficRollups.reduce((sum, item) => sum + item.rxBytes + item.txBytes, 0);
  const enabledNodes = nodes.filter((item) => item.enabled).length;
  const onlineRemoteServers = remoteServers.filter((item) => item.status === "online").length;
  const activeUsers = users.filter((item) => item.status === "active").length;
  const latestBackup = backups[0]?.filePath ?? "暂无备份";
  const latestSnapshot = xraySnapshots[0]?.createdAt ?? "暂无快照";
  const taskQueueCount = Object.values(remoteTasks).reduce((sum, items) => sum + items.length, 0);

  const overviewMetrics = [
    { label: "功能域", value: String(data?.dashboard.modulesTotal ?? modules.length), detail: "当前控制台覆盖模块" },
    { label: "节点库存", value: String(nodes.length), detail: `${enabledNodes} 个启用中` },
    { label: "远程主机", value: String(remoteServers.length), detail: `${onlineRemoteServers} 台在线` },
    { label: "流量汇总", value: formatBytes(totalTrafficBytes), detail: `${trafficRollups.length} 个汇总视图` },
  ];

  const dashboardShortcuts: Array<{ key: ConsoleSection; title: string; detail: string; meta: string }> = [
    { key: "nodes", title: "节点管理", detail: `${nodes.length} 个节点`, meta: "导入、启停、标签与协议" },
    { key: "subscriptions", title: "订阅发布", detail: `${subscriptions.length} 条订阅`, meta: "预览、下载、套餐和绑定" },
    { key: "remote", title: "远程运维", detail: `${remoteServers.length} 台 VPS`, meta: "任务队列、Agent 日志与批量执行" },
    { key: "xray", title: "Xray 工作区", detail: `${xrayProfiles.length} 套运行配置`, meta: "预览、快照、外置与内联模式" },
  ];

  function renderDashboardSection() {
    return (
      <>
        <section className="overview-grid">
          {overviewMetrics.map((metric) => (
            <article className="metric-card" key={metric.label}>
              <span className="metric-label">{metric.label}</span>
              <strong>{metric.value}</strong>
              <p>{metric.detail}</p>
            </article>
          ))}
        </section>

        <section className="page-grid dashboard-grid">
          <article className="section-panel span-2">
            <SectionHeader
              kicker="模块状态"
              title="当前功能覆盖与能力清单"
              note="优先把高频运维信息做成密集而可扫描的控制台，不再使用大横幅首页。"
            />
            <div className="module-grid">
              {modules.length ? (
                modules.map((module) => (
                  <article className="module-card" key={module.key}>
                    <div className="entity-head">
                      <strong>{module.name}</strong>
                      <span>{module.status}</span>
                    </div>
                    <p>{module.description}</p>
                    <ul>
                      {module.capabilities.map((capability) => (
                        <li key={capability}>{capability}</li>
                      ))}
                    </ul>
                  </article>
                ))
              ) : (
                <EmptyState title="模块数据还没返回" note="工作区加载完成后，这里会展示后端暴露的功能域和能力清单。" />
              )}
            </div>
          </article>

          <article className="section-panel">
            <SectionHeader kicker="快速入口" title="常用工作流" />
            <div className="quick-nav-grid">
              {dashboardShortcuts.map((item) => (
                <button type="button" className="quick-nav-card" key={item.key} onClick={() => setActiveSection(item.key)}>
                  <strong>{item.title}</strong>
                  <span>{item.detail}</span>
                  <small>{item.meta}</small>
                </button>
              ))}
            </div>
          </article>

          <article className="section-panel">
            <SectionHeader kicker="运行观察" title="核心运行信号" />
            <div className="status-list">
              <StatusLine label="平台模式" value={data?.dashboard.platformMode ?? "selfhost"} />
              <StatusLine label="功能门禁" value={data?.dashboard.gatingModel ?? "local-auth"} />
              <StatusLine label="最近备份" value={latestBackup} />
              <StatusLine label="最近快照" value={latestSnapshot} />
              <StatusLine label="通知通道" value={`${notificationChannels.length} 个`} />
              <StatusLine label="缓存任务" value={`${taskQueueCount} 条`} />
            </div>
          </article>

          <article className="section-panel">
            <SectionHeader kicker="业务摘要" title="模板、订阅与账号" />
            <MiniList
              items={[
                { id: "templates", title: `模板 ${templates.length} 个`, subtitle: "内置模板和私有模板统一维护" },
                { id: "subscriptions", title: `订阅 ${subscriptions.length} 条`, subtitle: "多客户端格式生成与下载" },
                { id: "packages", title: `套餐 ${packages.length} 个`, subtitle: `${entitlements.length} 条用户绑定记录` },
                { id: "users", title: `用户 ${users.length} 人`, subtitle: `${activeUsers} 人处于活跃状态` },
              ]}
            />
          </article>
        </section>
      </>
    );
  }

  function renderNodesSection() {
    return (
      <section className="page-grid two-column">
        <article className="section-panel">
          <SectionHeader kicker="节点录入" title="新增节点与批量导入" note="同一页完成手动录入和分享链接导入，适合快速整理节点库存。" />
          <div className="form-grid">
            <form className="stack-form" onSubmit={(event) => void handleCreateNode(event)}>
              <input placeholder="节点名称" value={nodeName} onChange={(event) => setNodeName(event.target.value)} />
              <input placeholder="服务器地址" value={nodeHost} onChange={(event) => setNodeHost(event.target.value)} />
              <div className="inline-form">
                <select value={nodeProtocol} onChange={(event) => setNodeProtocol(event.target.value)}>
                  {["vmess", "vless", "trojan", "shadowsocks", "hysteria2", "tuic", "snell", "socks5"].map((value) => (
                    <option key={value} value={value}>
                      {value}
                    </option>
                  ))}
                </select>
                <input value={nodePort} onChange={(event) => setNodePort(event.target.value)} placeholder="端口" />
              </div>
              <input placeholder="标签，逗号分隔" value={nodeTags} onChange={(event) => setNodeTags(event.target.value)} />
              <button type="submit">创建节点</button>
            </form>

            <form className="stack-form" onSubmit={(event) => void handleImportNodes(event)}>
              <textarea
                placeholder="每行一条链接，支持 vmess://、vless://、trojan://、ss://"
                value={nodeImportContent}
                onChange={(event) => setNodeImportContent(event.target.value)}
              />
              <button type="submit">导入分享链接</button>
              {nodeImportStatus ? <p className="status">{nodeImportStatus}</p> : null}
            </form>
          </div>
        </article>

        <article className="section-panel">
          <SectionHeader kicker="节点列表" title="库存与开关" />
          <div className="entity-list">
            {nodes.length ? (
              nodes.map((item) => (
                <div className="entity-card" key={item.id}>
                  <div className="entity-head">
                    <strong>{item.name}</strong>
                    <span>{item.enabled ? "启用中" : "已停用"}</span>
                  </div>
                  <p>
                    {item.protocol} · {item.serverHost}:{item.serverPort}
                  </p>
                  <div className="chip-row">
                    {item.tags.map((tag) => (
                      <span className="chip" key={tag}>
                        {tag}
                      </span>
                    ))}
                  </div>
                  <div className="action-row">
                    <button
                      type="button"
                      className="ghost-button"
                      onClick={() =>
                        void updateNode(item.id, {
                          name: item.name,
                          sourceKind: item.sourceKind,
                          protocol: item.protocol,
                          serverHost: item.serverHost,
                          serverPort: item.serverPort,
                          tags: item.tags,
                          metadata: item.metadata,
                          enabled: !item.enabled,
                        })
                      }
                    >
                      {item.enabled ? "停用" : "启用"}
                    </button>
                  <button type="button" className="ghost-button danger-button" onClick={() => void deleteNode(item.id)}>
                    删除
                  </button>
                  </div>
                </div>
              ))
            ) : (
              <EmptyState title="还没有节点" note="先手动创建一个节点，或把分享链接粘贴到左侧导入框。" />
            )}
          </div>
        </article>
      </section>
    );
  }

  function renderRulesSection() {
    return (
      <section className="page-grid two-column">
        <article className="section-panel">
          <SectionHeader kicker="规则仓库" title="已保存规则集" note="保留默认规则预置，也支持把自定义规则集持久化归档。" />
          <div className="rule-list">
            {ruleSets.length > 0
              ? ruleSets.map((item) => (
                  <div className="entity-card" key={item.id}>
                    <div className="entity-head">
                      <strong>{item.name}</strong>
                      <span>{item.scope}</span>
                    </div>
                    <p>{item.description || "暂无说明"}</p>
                    <div className="rule-list compact">
                      {item.rules.map((rule) => (
                        <div className="rule-row" key={rule.id}>
                          <span>{String(rule.sortOrder).padStart(2, "0")}</span>
                          <code>
                            {rule.ruleType}
                            {rule.pattern ? `,${rule.pattern}` : ""}
                            ,{rule.policy}
                          </code>
                        </div>
                      ))}
                    </div>
                    <div className="action-row">
                      <button type="button" className="ghost-button" onClick={() => editRuleSet(item)}>
                        编辑
                      </button>
                      <button type="button" className="ghost-button danger-button" onClick={() => void deleteRuleSet(item.id)}>
                        删除
                      </button>
                    </div>
                  </div>
                ))
              : starterRules.map((rule, index) => (
                  <div className="rule-row" key={rule}>
                    <span>{String(index + 1).padStart(2, "0")}</span>
                    <code>{rule}</code>
                  </div>
                ))}
          </div>
        </article>

        <article className="section-panel">
          <SectionHeader kicker="可视化编辑器" title={editingRuleSetId ? "编辑规则集" : "创建规则集"} />
          <form className="rule-form" onSubmit={(event) => void handleCreateRuleSet(event)}>
            <label>
              规则集名称
              <input value={ruleSetName} onChange={(event) => setRuleSetName(event.target.value)} />
            </label>
            <label>
              说明
              <textarea value={ruleSetDescription} onChange={(event) => setRuleSetDescription(event.target.value)} />
            </label>

            <div className="draft-rule-list">
              {draftRules.map((rule, index) => (
                <div className="draft-rule" key={rule.id}>
                  <div className="entity-head">
                    <strong>规则 {index + 1}</strong>
                    <span>{rule.enabled ? "启用" : "停用"}</span>
                  </div>
                  <select value={rule.ruleType} onChange={(event) => updateDraftRule(rule.id, { ruleType: event.target.value })}>
                    {ruleTypes.map((value) => (
                      <option key={value.key} value={value.key}>
                        {value.key}
                      </option>
                    ))}
                  </select>
                  <input
                    value={rule.pattern}
                    disabled={rule.ruleType === "MATCH"}
                    onChange={(event) => updateDraftRule(rule.id, { pattern: event.target.value })}
                    placeholder={ruleTypes.find((item) => item.key === rule.ruleType)?.patternHint ?? ""}
                  />
                  <select value={rule.policy} onChange={(event) => updateDraftRule(rule.id, { policy: event.target.value })}>
                    {policyOptions.map((value) => (
                      <option key={value} value={value}>
                        {value}
                      </option>
                    ))}
                  </select>
                  <input value={rule.note} onChange={(event) => updateDraftRule(rule.id, { note: event.target.value })} placeholder="备注" />
                  <label className="check-row">
                    <input
                      type="checkbox"
                      checked={rule.enabled}
                      onChange={(event) => updateDraftRule(rule.id, { enabled: event.target.checked })}
                    />
                    启用此规则
                  </label>
                  <div className="action-row">
                    <button type="button" className="ghost-button" onClick={() => moveDraftRule(rule.id, -1)}>
                      上移
                    </button>
                    <button type="button" className="ghost-button" onClick={() => moveDraftRule(rule.id, 1)}>
                      下移
                    </button>
                      <button type="button" className="ghost-button danger-button" onClick={() => removeDraftRule(rule.id)}>
                        删除
                      </button>
                  </div>
                </div>
              ))}
            </div>

            {ruleValidation ? <pre className="validation-box">{ruleValidation}</pre> : null}

            <div className="action-row">
              <button type="button" onClick={() => setDraftRules((current) => [...current, { ...emptyDraftRule(), sortOrder: current.length + 1 }])}>
                添加规则
              </button>
              <button type="submit">{editingRuleSetId ? "更新规则集" : "保存规则集"}</button>
              {editingRuleSetId ? (
                <button type="button" className="ghost-button" onClick={resetRuleEditor}>
                  取消编辑
                </button>
              ) : null}
            </div>
          </form>
        </article>
      </section>
    );
  }

  function renderTemplatesSection() {
    return (
      <section className="page-grid two-column">
        <article className="section-panel">
          <SectionHeader kicker="模板编辑" title="新增或调整模板" note="把模板变量和正文放在同一个表单里，便于后续快速绑定订阅输出。" />
          <form className="stack-form" onSubmit={(event) => void handleCreateTemplate(event)}>
            <input placeholder="模板名称" value={templateName} onChange={(event) => setTemplateName(event.target.value)} />
            <input placeholder="模板说明" value={templateDescription} onChange={(event) => setTemplateDescription(event.target.value)} />
            <input placeholder="变量名，逗号分隔" value={templateVariables} onChange={(event) => setTemplateVariables(event.target.value)} />
            <textarea value={templateContent} onChange={(event) => setTemplateContent(event.target.value)} />
            <button type="submit">创建模板</button>
          </form>
        </article>

        <article className="section-panel">
          <SectionHeader kicker="模板列表" title="内置与私有模板" />
          <div className="template-list">
            {templates.length ? (
              templates.map((template) => (
                <div className="template-row" key={template.id}>
                  <div>
                    <strong>{template.name}</strong>
                    <p>{template.description}</p>
                    <code>{template.variables.join(", ")}</code>
                  </div>
                  <div className="template-actions">
                    <span>{template.kind}</span>
                    {!template.locked ? (
                      <button type="button" className="ghost-button" onClick={() => void deleteTemplate(template.id)}>
                        删除
                      </button>
                    ) : null}
                  </div>
                </div>
              ))
            ) : (
              <EmptyState title="模板库暂时为空" note="创建模板后，可以在订阅页把它绑定到不同客户端输出格式。" />
            )}
          </div>
        </article>
      </section>
    );
  }

  function renderSubscriptionsSection() {
    return (
      <section className="page-grid dashboard-grid">
        <article className="section-panel">
          <SectionHeader kicker="订阅输出" title="创建订阅" />
          <form className="stack-form" onSubmit={(event) => void handleCreateSubscription(event)}>
            <input placeholder="订阅名称" value={subscriptionName} onChange={(event) => setSubscriptionName(event.target.value)} />
            <select value={subscriptionFormat} onChange={(event) => setSubscriptionFormat(event.target.value)}>
              {outputFormats.map((value) => (
                <option key={value} value={value}>
                  {value}
                </option>
              ))}
            </select>
            <select value={subscriptionTemplateId} onChange={(event) => setSubscriptionTemplateId(event.target.value)}>
              {templates.map((template) => (
                <option key={template.id} value={template.id}>
                  {template.name}
                </option>
              ))}
            </select>
            <input placeholder="来源，逗号分隔" value={subscriptionSources} onChange={(event) => setSubscriptionSources(event.target.value)} />
            <button type="submit">创建订阅</button>
          </form>
        </article>

        <article className="section-panel span-2">
          <SectionHeader kicker="订阅列表" title="预览、下载与清理" />
          <div className="entity-list">
            {subscriptions.length ? (
              subscriptions.map((item) => (
                <div className="entity-card" key={item.id}>
                  <div className="entity-head">
                    <strong>{item.name}</strong>
                    <span>{item.outputFormat}</span>
                  </div>
                  <p>模板：{item.templateId}</p>
                  <div className="chip-row">
                    {item.sources.map((source) => (
                      <span className="chip" key={source}>
                        {source}
                      </span>
                    ))}
                  </div>
                  <div className="action-row">
                    <button type="button" className="ghost-button" onClick={() => void handlePreviewSubscription(item.id)}>
                      预览
                    </button>
                    <a className="ghost-link" href={subscriptionDownloadURL(item.id)}>
                      下载
                    </a>
                  <button type="button" className="ghost-button danger-button" onClick={() => void deleteSubscription(item.id)}>
                    删除
                  </button>
                  </div>
                </div>
              ))
            ) : (
              <EmptyState title="还没有订阅" note="先创建一个订阅，再绑定模板和输出格式。" />
            )}
          </div>

          {previewError ? <p className="status error">{previewError}</p> : null}
          {renderedSubscription ? (
            <div className="preview-box">
              <div className="entity-head">
                <strong>{renderedSubscription.fileName}</strong>
                <span>{renderedSubscription.outputFormat}</span>
              </div>
              <pre>{renderedSubscription.content}</pre>
            </div>
          ) : null}
        </article>

        <article className="section-panel">
          <SectionHeader kicker="套餐配置" title="创建套餐" />
          <form className="stack-form" onSubmit={(event) => void handleCreatePackage(event)}>
            <input value={packageName} onChange={(event) => setPackageName(event.target.value)} />
            <input value={packageDescription} onChange={(event) => setPackageDescription(event.target.value)} />
            <div className="inline-form">
              <input value={packageBandwidth} onChange={(event) => setPackageBandwidth(event.target.value)} placeholder="带宽字节数" />
              <input value={packageDevices} onChange={(event) => setPackageDevices(event.target.value)} placeholder="设备数" />
            </div>
            <input value={packageDuration} onChange={(event) => setPackageDuration(event.target.value)} placeholder="时长（天）" />
            <input value={packageFeatures} onChange={(event) => setPackageFeatures(event.target.value)} placeholder="功能标签" />
            <button type="submit">创建套餐</button>
          </form>
          <MiniList
            items={packages.map((item) => ({
              id: item.id,
              title: item.name,
              subtitle: `${item.deviceLimit} 台设备 / ${formatBytes(item.bandwidthBytes)} / ${item.durationDays} 天`,
            }))}
            onDelete={(id) => void runOps(() => deletePackage(id))}
          />
        </article>

        <article className="section-panel">
          <SectionHeader kicker="用户绑定" title="授权套餐" />
          <form className="stack-form" onSubmit={(event) => void handleCreateEntitlement(event)}>
            <select value={entitlementUserId} onChange={(event) => setEntitlementUserId(event.target.value)}>
              {[{ id: "local-admin", username: "local-admin" }, ...users.filter((user) => user.id !== "local-admin")].map((user) => (
                <option key={user.id} value={user.id}>
                  {user.username}
                </option>
              ))}
            </select>
            <select value={entitlementPackageId} onChange={(event) => setEntitlementPackageId(event.target.value)}>
              <option value="">选择套餐</option>
              {packages.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.name}
                </option>
              ))}
            </select>
            <input value={entitlementExpiresAt} onChange={(event) => setEntitlementExpiresAt(event.target.value)} placeholder="到期时间，可留空" />
            <button type="submit">绑定套餐</button>
          </form>
          <MiniList
            items={entitlements.map((item) => ({
              id: item.id,
              title: `${item.userId} -> ${packages.find((pkg) => pkg.id === item.packageId)?.name ?? item.packageId}`,
              subtitle: `${item.status}${item.expiresAt ? ` / 到期 ${item.expiresAt}` : ""}`,
            }))}
            onDelete={(id) => void runOps(() => deleteEntitlement(id))}
          />
        </article>
      </section>
    );
  }

  function renderUsersSection() {
    return (
      <section className="page-grid">
        <article className="section-panel">
          <SectionHeader kicker="账号管理" title="成员与操作员权限" note="创建、禁用或删除成员账号。写操作仍要求管理员会话。" />
          {isAuthenticated ? (
            <>
              <form className="user-form" onSubmit={(event) => void handleCreateUser(event)}>
                <input value={newUsername} onChange={(event) => setNewUsername(event.target.value)} placeholder="用户名" />
                <input value={newUserPassword} onChange={(event) => setNewUserPassword(event.target.value)} placeholder="密码" type="password" />
                <input value={newUserDisplayName} onChange={(event) => setNewUserDisplayName(event.target.value)} placeholder="显示名称" />
                <input value={newUserEmail} onChange={(event) => setNewUserEmail(event.target.value)} placeholder="邮箱" />
                <button type="submit">创建成员</button>
              </form>
              {userError ? <p className="status error">{userError}</p> : null}
              <div className="user-list">
                {users.map((user) => (
                  <div className="mini-row" key={user.id}>
                    <div>
                      <strong>{user.displayName || user.username}</strong>
                      <span>
                        {user.username} / {user.role} / {user.status}
                      </span>
                    </div>
                    <div className="mini-actions">
                      <button type="button" className="ghost-button" onClick={() => void handleToggleUserStatus(user.id)}>
                        {user.status === "active" ? "禁用" : "启用"}
                      </button>
                      {user.id !== "local-admin" ? (
                      <button type="button" className="ghost-button danger-button" onClick={() => void deleteUser(user.id)}>
                        删除
                      </button>
                      ) : null}
                    </div>
                  </div>
                ))}
              </div>
            </>
          ) : (
            <div className="empty-state">
              <strong>当前为只读会话</strong>
              <p>先使用顶部会话条登录，再管理成员账号。</p>
            </div>
          )}
        </article>
      </section>
    );
  }

  function renderTrafficSection() {
    return (
      <section className="page-grid two-column">
        <article className="section-panel">
          <SectionHeader kicker="流量采样" title="记录用量" />
          <form className="stack-form" onSubmit={(event) => void handleCreateTrafficSample(event)}>
            <div className="inline-form">
              <select value={trafficScope} onChange={(event) => setTrafficScope(event.target.value)}>
                {["server", "node", "user"].map((value) => (
                  <option key={value} value={value}>
                    {value}
                  </option>
                ))}
              </select>
              <input value={trafficScopeID} onChange={(event) => setTrafficScopeID(event.target.value)} placeholder="作用域 ID" />
            </div>
            <div className="inline-form">
              <input value={trafficRX} onChange={(event) => setTrafficRX(event.target.value)} placeholder="接收字节" />
              <input value={trafficTX} onChange={(event) => setTrafficTX(event.target.value)} placeholder="发送字节" />
            </div>
            <button type="submit">记录采样</button>
          </form>
          <MiniList
            items={trafficSamples.map((item) => ({
              id: item.id,
              title: `${item.sampleScope}:${item.scopeId}`,
              subtitle: `RX ${item.rxBytes} / TX ${item.txBytes}`,
            }))}
          />
        </article>

        <article className="section-panel">
          <SectionHeader kicker="汇总视图" title="统计结果" />
          <MiniList
            items={trafficRollups.map((item) => ({
              id: `${item.sampleScope}:${item.scopeId}`,
              title: `${item.sampleScope}:${item.scopeId}`,
              subtitle: `${formatBytes(item.rxBytes + item.txBytes)} / ${item.samples} 条样本 / ${item.lastSeenAt || "暂无时间"}`,
            }))}
          />
        </article>
      </section>
    );
  }

  function renderRemoteSection() {
    return (
      <section className="page-grid dashboard-grid">
        <article className="section-panel">
          <SectionHeader kicker="主机纳管" title="注册远程 VPS" />
          <form className="stack-form" onSubmit={(event) => void handleCreateRemoteServer(event)}>
            <input placeholder="主机名称" value={remoteName} onChange={(event) => setRemoteName(event.target.value)} />
            <input placeholder="域名或公网 IP" value={remoteHost} onChange={(event) => setRemoteHost(event.target.value)} />
            <div className="inline-form">
              <select value={remoteConnectionMode} onChange={(event) => setRemoteConnectionMode(event.target.value)}>
                {["pull", "websocket", "http"].map((value) => (
                  <option key={value} value={value}>
                    {value}
                  </option>
                ))}
              </select>
              <input placeholder="标签" value={remoteTags} onChange={(event) => setRemoteTags(event.target.value)} />
            </div>
            <button type="submit">注册主机</button>
          </form>
          {remoteEnrollment ? (
            <div className="token-box">
              <div className="entity-head">
                <strong>{remoteEnrollment.server.name} 的一次性入站令牌</strong>
                <span>仅显示一次</span>
              </div>
              <code>server: {remoteEnrollment.serverToken}</code>
              <code>agent: {remoteEnrollment.agentToken}</code>
            </div>
          ) : null}
          {remoteError ? <p className="status error">{remoteError}</p> : null}
        </article>

        <article className="section-panel">
          <SectionHeader kicker="任务模板" title="任务类型与 JSON 载荷" />
          <div className="stack-form">
            <select value={remoteTaskKind} onChange={(event) => setRemoteTaskKind(event.target.value)}>
              {[
                "reload-config",
                "apply-xray-config",
                "restart-xray",
                "install-xray",
                "install-nginx",
                "renew-certificate",
                "install-warp",
                "shell-script",
              ].map((value) => (
                <option key={value} value={value}>
                  {value}
                </option>
              ))}
            </select>
            <textarea value={remoteTaskPayload} onChange={(event) => setRemoteTaskPayload(event.target.value)} />
            <p className="status">载荷会和任务一起入队，供远程 agent 执行。</p>
          </div>
        </article>

        <article className="section-panel span-2">
          <SectionHeader kicker="远程节点" title="主机列表、任务与日志" />
          <div className="entity-list">
            {remoteServers.length ? (
              remoteServers.map((server) => (
              <div className="entity-card" key={server.id}>
                <div className="entity-head">
                  <strong>{server.name}</strong>
                  <span>{server.status}</span>
                </div>
                <p>
                  {server.host} · {server.connectionMode}
                </p>
                <div className="chip-row">
                  {Array.isArray(server.metadata.tags)
                    ? server.metadata.tags.map((tag) => (
                        <span className="chip" key={String(tag)}>
                          {String(tag)}
                        </span>
                      ))
                    : null}
                </div>
                <div className="action-row">
                  <button type="button" className="ghost-button" onClick={() => void handleSetRemoteStatus(server.id, "online")}>
                    标记在线
                  </button>
                  <button type="button" className="ghost-button" onClick={() => void handleSetRemoteStatus(server.id, "maintenance")}>
                    维护模式
                  </button>
                  <button type="button" className="ghost-button" onClick={() => void handleCreateRemoteTask(server.id)}>
                    队列任务
                  </button>
                  <button type="button" className="ghost-button" onClick={() => void handleLoadRemoteTasks(server.id)}>
                    读取任务
                  </button>
                  <button type="button" className="ghost-button" onClick={() => void handleLoadRemoteLogs(server.id)}>
                    读取日志
                  </button>
                  <button type="button" className="ghost-button danger-button" onClick={() => void deleteRemoteServer(server.id)}>
                    删除
                  </button>
                </div>

                {remoteTasks[server.id]?.length ? (
                  <div className="task-list">
                    {remoteTasks[server.id].map((task) => (
                      <div className="rule-row" key={task.id}>
                        <span>{task.status}</span>
                        <code>{task.taskKind}</code>
                        <small>{task.createdAt}</small>
                        <button type="button" className="ghost-button" onClick={() => void handleLoadRemoteTaskLogs(server.id, task.id)}>
                          任务日志
                        </button>
                      </div>
                    ))}
                  </div>
                ) : null}

                {remoteAgentLogs[server.id]?.length ? (
                  <div className="log-list">
                    <strong>Agent 日志</strong>
                    {remoteAgentLogs[server.id].slice(0, 6).map((logItem) => (
                      <LogRow
                        key={logItem.id}
                        kind={logItem.level}
                        message={logItem.message}
                        createdAt={logItem.createdAt}
                      />
                    ))}
                  </div>
                ) : null}

                {remoteTasks[server.id]?.flatMap((task) => remoteTaskLogs[`${server.id}:${task.id}`] ?? []).length ? (
                  <div className="log-list">
                    <strong>任务日志</strong>
                    {remoteTasks[server.id]
                      .flatMap((task) => remoteTaskLogs[`${server.id}:${task.id}`] ?? [])
                      .slice(0, 8)
                      .map((logItem) => (
                        <LogRow
                          key={logItem.id}
                          kind={logItem.eventKind}
                          message={logItem.message || logItem.remoteTaskId}
                          createdAt={logItem.createdAt}
                        />
                      ))}
                  </div>
                ) : null}
              </div>
              ))
            ) : (
              <EmptyState title="还没有纳管 VPS" note="注册远程主机后，可以在这里入队任务、读取 Agent 日志和应用 Xray 配置。" />
            )}
          </div>
        </article>
      </section>
    );
  }

  function renderXraySection() {
    return (
      <section className="page-grid dashboard-grid">
        <article className="section-panel span-2">
          <SectionHeader
            kicker="配置工作区"
            title="Xray 预览与快照"
            actions={
              <>
                <button type="button" onClick={() => void handlePreviewXray()}>
                  预览配置
                </button>
                <button type="button" className="ghost-button" onClick={() => void handleSaveXraySnapshot()}>
                  保存快照
                </button>
              </>
            }
          />
          {xrayError ? <p className="status error">{xrayError}</p> : null}
          {xrayApplyStatus ? <p className="status">{xrayApplyStatus}</p> : null}
          {xrayPreview ? (
            <div className="preview-box">
              <div className="entity-head">
                <strong>{xrayPreview.summary}</strong>
                <span>json</span>
              </div>
              <pre>{xrayPreview.content}</pre>
            </div>
          ) : (
            <div className="empty-state">
              <strong>还没有预览结果</strong>
              <p>点击“预览配置”后，这里会展示完整 JSON。</p>
            </div>
          )}
          {restoredSnapshot ? (
            <div className="preview-box">
              <div className="entity-head">
                <strong>已恢复快照内容</strong>
                <span>rollback</span>
              </div>
              <pre>{restoredSnapshot}</pre>
            </div>
          ) : null}
        </article>

        <article className="section-panel">
          <SectionHeader kicker="运行模式" title="外置与内联 Xray" />
          <form className="xray-profile-form" onSubmit={(event) => void handleCreateXrayProfile(event)}>
            <input value={xrayProfileName} onChange={(event) => setXrayProfileName(event.target.value)} placeholder="配置名称" />
            <select value={xrayProfileRemoteServerId} onChange={(event) => setXrayProfileRemoteServerId(event.target.value)}>
              <option value="">本地草稿 / 未绑定 VPS</option>
              {remoteServers.map((server) => (
                <option key={server.id} value={server.id}>
                  {server.name} ({server.host})
                </option>
              ))}
            </select>
            <select value={xrayProfileMode} onChange={(event) => setXrayProfileMode(event.target.value)}>
              <option value="external">外置 Xray</option>
              <option value="inline">内联 Xray</option>
            </select>
            <input value={xrayProfileBinary} onChange={(event) => setXrayProfileBinary(event.target.value)} placeholder="xray 可执行文件" />
            <input value={xrayProfileConfigPath} onChange={(event) => setXrayProfileConfigPath(event.target.value)} placeholder="配置路径" />
            <input value={xrayProfileService} onChange={(event) => setXrayProfileService(event.target.value)} placeholder="systemd 服务名" />
            <button type="submit">创建配置</button>
          </form>
          <MiniList
            items={xrayProfiles.map((item) => ({
              id: item.id,
              title: item.name,
              subtitle: `${item.runtimeMode} / ${item.remoteServerId || "未绑定"} / ${item.configPath}`,
            }))}
            onDelete={(id) => void runOps(() => deleteXrayProfile(id))}
            renderActions={(item) => (
              <>
                <button type="button" className="ghost-button" onClick={() => void handleApplyXrayProfile(item.id, true)}>
                  Dry-run
                </button>
                <button type="button" className="ghost-button" onClick={() => void handleApplyXrayProfile(item.id, false)}>
                  应用
                </button>
              </>
            )}
          />
        </article>

        <article className="section-panel">
          <SectionHeader kicker="代理组" title="策略分组" />
          <form className="stack-form" onSubmit={(event) => void handleCreateProxyGroup(event)}>
            <input value={proxyGroupName} onChange={(event) => setProxyGroupName(event.target.value)} />
            <select value={proxyGroupKind} onChange={(event) => setProxyGroupKind(event.target.value)}>
              {["select", "url-test", "fallback", "load-balance", "relay"].map((value) => (
                <option key={value} value={value}>
                  {value}
                </option>
              ))}
            </select>
            <textarea value={proxyGroupConfig} onChange={(event) => setProxyGroupConfig(event.target.value)} />
            <button type="submit">创建分组</button>
          </form>
          <MiniList
            items={proxyGroups.map((item) => ({ id: item.id, title: item.name, subtitle: item.groupKind }))}
            onDelete={(id) => void runOps(() => deleteProxyGroup(id))}
          />
        </article>

        <article className="section-panel">
          <SectionHeader kicker="快照历史" title="回滚素材" />
          <MiniList
            items={xraySnapshots.map((item) => ({
              id: item.id,
              title: item.summary || item.targetId,
              subtitle: `${item.targetKind}/${item.targetId} · ${item.createdAt}`,
            }))}
            renderActions={(item) => (
              <button type="button" className="ghost-button" onClick={() => void handleRestoreXraySnapshot(item.id)}>
                恢复预览
              </button>
            )}
          />
        </article>
      </section>
    );
  }

  function renderSystemSection() {
    return (
      <section className="page-grid dashboard-grid">
        <article className="section-panel">
          <SectionHeader kicker="DNS" title="解析服务商" />
          <form className="stack-form" onSubmit={(event) => void handleCreateDNSProvider(event)}>
            <input value={dnsProviderName} onChange={(event) => setDNSProviderName(event.target.value)} />
            <select value={dnsProviderKind} onChange={(event) => setDNSProviderKind(event.target.value)}>
              {["cloudflare", "alidns", "dnspod", "tencent", "godaddy", "namesilo"].map((value) => (
                <option key={value} value={value}>
                  {value}
                </option>
              ))}
            </select>
            <textarea value={dnsCredentials} onChange={(event) => setDNSCredentials(event.target.value)} />
            <button type="submit">保存服务商</button>
          </form>
          <MiniList
            items={dnsProviders.map((item) => ({ id: item.id, title: item.name, subtitle: item.providerKind }))}
            onDelete={(id) => void runOps(() => deleteDNSProvider(id))}
          />
        </article>

        <article className="section-panel">
          <SectionHeader kicker="证书" title="ACME 资产" />
          <form className="stack-form" onSubmit={(event) => void handleCreateCertificate(event)}>
            <input value={certificateName} onChange={(event) => setCertificateName(event.target.value)} />
            <input value={certificateDomain} onChange={(event) => setCertificateDomain(event.target.value)} />
            <button type="submit">创建证书记录</button>
          </form>
          <MiniList
            items={certificates.map((item) => ({ id: item.id, title: item.name, subtitle: item.domain }))}
            onDelete={(id) => void runOps(() => deleteCertificate(id))}
          />
        </article>

        <article className="section-panel">
          <SectionHeader kicker="通知" title="告警通道" />
          <form className="stack-form" onSubmit={(event) => void handleCreateNotification(event)}>
            <input value={notificationName} onChange={(event) => setNotificationName(event.target.value)} />
            <textarea value={notificationConfig} onChange={(event) => setNotificationConfig(event.target.value)} />
            <button type="submit">创建通道</button>
          </form>
          <MiniList
            items={notificationChannels.map((item) => ({
              id: item.id,
              title: item.name,
              subtitle: `${item.channelKind} / ${item.enabled ? "启用" : "停用"}`,
            }))}
            onDelete={(id) => void runOps(() => deleteNotificationChannel(id))}
            renderActions={(item) => (
              <button
                type="button"
                className="ghost-button"
                onClick={() => void runOps(() => testNotificationChannel(item.id, { message: "HarborX test notification" }))}
              >
                测试
              </button>
            )}
          />
        </article>

        <article className="section-panel">
          <SectionHeader kicker="备份" title="数据库导出" />
          <form className="stack-form" onSubmit={(event) => void handleCreateBackup(event)}>
            <input value={backupPath} onChange={(event) => setBackupPath(event.target.value)} placeholder="导出路径" />
            <button type="submit">导出备份</button>
          </form>
          <MiniList
            items={backups.map((item) => ({ id: item.id, title: item.backupKind, subtitle: item.filePath }))}
            onDelete={(id) => void runOps(() => deleteBackup(id))}
          />
        </article>

        <article className="section-panel">
          <SectionHeader kicker="系统设置" title="运行时配置" />
          <form className="stack-form" onSubmit={(event) => void handleUpsertSetting(event)}>
            <input value={settingKey} onChange={(event) => setSettingKey(event.target.value)} />
            <textarea value={settingValue} onChange={(event) => setSettingValue(event.target.value)} />
            <button type="submit">保存设置</button>
          </form>
          <MiniList
            items={systemSettings.map((item) => ({ id: item.key, title: item.key, subtitle: JSON.stringify(item.value) }))}
            onDelete={(id) => void runOps(() => deleteSystemSetting(id))}
          />
        </article>

        <article className="section-panel span-2">
          <SectionHeader
            kicker="高级自动化"
            title="运维资源编排"
            note="这里集中管理 Xray 入站、证书续期、Nginx 回落和安全策略等自动化资源。"
          />
          <form className="stack-form" onSubmit={(event) => void handleCreateOpsResource(event)}>
            <select value={opsResourceKind} onChange={(event) => handleOpsKindChange(event.target.value)}>
              {opsResourceKinds.map((kind) => (
                <option key={kind} value={kind}>
                  {kind}
                </option>
              ))}
            </select>
            <input value={opsResourceName} onChange={(event) => setOpsResourceName(event.target.value)} />
            <select value={opsRemoteServerId} onChange={(event) => setOpsRemoteServerId(event.target.value)}>
              <option value="">本地草稿 / 未绑定 VPS</option>
              {remoteServers.map((server) => (
                <option key={server.id} value={server.id}>
                  {server.name} ({server.host})
                </option>
              ))}
            </select>
            <input value={opsAction} onChange={(event) => setOpsAction(event.target.value)} placeholder="动作名，可选" />
            <textarea value={opsConfig} onChange={(event) => setOpsConfig(event.target.value)} />
            <button type="submit">保存资源</button>
          </form>
          {opsStatus ? <p className="status">{opsStatus}</p> : null}
          <MiniList
            items={opsResources.map((item) => ({
              id: item.id,
              title: item.name,
              subtitle: `${item.resourceKind} / ${item.remoteServerId || "未绑定"} / ${item.status}`,
            }))}
            onDelete={(id) => void runOps(() => deleteOpsResource(id))}
            renderActions={(item) => (
              <>
                <button type="button" className="ghost-button" onClick={() => void handleExecuteOpsResource(item.id, true)}>
                  Dry-run
                </button>
                <button type="button" className="ghost-button" onClick={() => void handleExecuteOpsResource(item.id, false)}>
                  执行
                </button>
              </>
            )}
          />
          {opsError ? <p className="status error">{opsError}</p> : null}
        </article>
      </section>
    );
  }

  function renderSectionContent() {
    switch (activeSection) {
      case "dashboard":
        return renderDashboardSection();
      case "nodes":
        return renderNodesSection();
      case "subscriptions":
        return renderSubscriptionsSection();
      case "rules":
        return renderRulesSection();
      case "templates":
        return renderTemplatesSection();
      case "users":
        return renderUsersSection();
      case "traffic":
        return renderTrafficSection();
      case "remote":
        return renderRemoteSection();
      case "xray":
        return renderXraySection();
      case "system":
        return renderSystemSection();
      default:
        return renderDashboardSection();
    }
  }

  return (
    <div className="shell">
      <aside className="sidebar">
        <div className="brand-block">
          <div className="brand-mark">HX</div>
          <div>
            <p className="eyebrow">HarborX Console</p>
            <h1>自托管节点与 Xray 运维台</h1>
            <p className="lede">按照运维控制台的逻辑重排信息层级，把节点、订阅、规则、远程执行和系统编排收在一套工作台里。</p>
          </div>
        </div>

        <nav className="nav">
          {consoleNavItems.map((item) => (
            <button
              type="button"
              key={item.key}
              className={item.key === activeSection ? "active" : ""}
              onClick={() => setActiveSection(item.key)}
            >
              <b>{String(consoleNavItems.findIndex((navItem) => navItem.key === item.key) + 1).padStart(2, "0")}</b>
              <span>{item.label}</span>
              <small>{item.helper}</small>
            </button>
          ))}
        </nav>

        <div className="side-status">
          <span className="signal-dot" />
          <div>
            <strong>{isAuthenticated ? "管理员会话已解锁" : "当前为只读浏览模式"}</strong>
            <small>{busy ? "正在保存变更" : `${data?.dashboard.modulesTotal ?? modules.length} 个功能模块可用`}</small>
          </div>
        </div>
      </aside>

      <main className="content">
        <section className="topbar">
          <div className="topbar-copy">
            <p className="eyebrow">Workspace</p>
            <h2>{activeNavItem.label}</h2>
            <p>{activeNavItem.helper}</p>
          </div>
          <div className="topbar-actions">
            <div className="topbar-meta">
              <span>{isAuthenticated ? `会话：${authUser?.username ?? authUsername}` : "会话：只读"}</span>
              <span>{loading ? "数据同步中" : "数据已加载"}</span>
              <span>{busy ? "写入中" : "空闲"}</span>
            </div>
            <button type="button" className="ghost-button" onClick={() => void refresh()}>
              刷新数据
            </button>
          </div>
        </section>

        <section className="section-panel session-strip">
          <div className="section-copy">
            <p className="section-kicker">会话与状态</p>
            <h3 className="section-title">{isAuthenticated ? "当前已登录，可执行写操作" : "当前未登录，仅可预览和下载"}</h3>
            <p className="section-note">创建、更新、删除和应用配置等动作都依赖管理员令牌，会话信息集中放在这里处理。</p>
          </div>
          {isAuthenticated ? (
            <div className="session-actions">
              <StatusLine label="当前账号" value={authUser?.username ?? authUsername} compact />
              <button type="button" className="ghost-button" onClick={handleLogout}>
                退出登录
              </button>
            </div>
          ) : (
            <form className="auth-form" onSubmit={(event) => void handleLogin(event)}>
              <input value={authUsername} onChange={(event) => setAuthUsername(event.target.value)} placeholder="用户名" />
              <input value={authPassword} onChange={(event) => setAuthPassword(event.target.value)} placeholder="密码" type="password" />
              <button type="submit">登录</button>
            </form>
          )}
          {authError ? <p className="status error">{authError}</p> : null}
        </section>

        <div className="status-stack">
          {loading ? <p className="status-banner">正在加载工作区数据…</p> : null}
          {error ? <p className="status-banner error">加载失败：{error}</p> : null}
          {busy ? <p className="status-banner">后台正在写入变更，请稍候。</p> : null}
        </div>

        {renderSectionContent()}
      </main>
    </div>
  );
}

function SectionHeader({
  kicker,
  title,
  note,
  actions,
}: {
  kicker: string;
  title: string;
  note?: string;
  actions?: ReactNode;
}) {
  return (
    <div className="section-head">
      <div className="section-copy">
        <p className="section-kicker">{kicker}</p>
        <h3 className="section-title">{title}</h3>
        {note ? <p className="section-note">{note}</p> : null}
      </div>
      {actions ? <div className="section-actions">{actions}</div> : null}
    </div>
  );
}

function StatusLine({ label, value, compact = false }: { label: string; value: string; compact?: boolean }) {
  return (
    <div className={`status-line${compact ? " compact" : ""}`}>
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function LogRow({ kind, message, createdAt }: { kind: string; message: string; createdAt: string }) {
  return (
    <div className="log-row">
      <span>{kind}</span>
      <code>{message}</code>
      <small>{createdAt}</small>
    </div>
  );
}

function EmptyState({ title, note }: { title: string; note: string }) {
  return (
    <div className="empty-state">
      <strong>{title}</strong>
      <p>{note}</p>
    </div>
  );
}

function MiniList({
  items,
  onDelete,
  renderActions,
}: {
  items: Array<{ id: string; title: string; subtitle: string }>;
  onDelete?: (id: string) => void;
  renderActions?: (item: { id: string; title: string; subtitle: string }) => ReactNode;
}) {
  if (items.length === 0) {
    return <p className="status mini-empty">暂无记录。</p>;
  }

  return (
    <div className="mini-list">
      {items.map((item) => (
        <div className="mini-row" key={item.id}>
          <div>
            <strong>{item.title}</strong>
            <span>{item.subtitle}</span>
          </div>
          <div className="mini-actions">
            {renderActions ? renderActions(item) : null}
            {onDelete ? (
              <button type="button" className="ghost-button danger-button" onClick={() => onDelete(item.id)}>
                删除
              </button>
            ) : null}
          </div>
        </div>
      ))}
    </div>
  );
}

function splitCSV(input: string) {
  return input
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function parseJSONObject(input: string): Record<string, unknown> {
  const parsed = JSON.parse(input) as unknown;
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error("Payload must be a JSON object");
  }
  return parsed as Record<string, unknown>;
}

function createClientId() {
  if (globalThis.crypto?.randomUUID) {
    return globalThis.crypto.randomUUID();
  }
  return `local-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
}

function formatBytes(value: number) {
  if (!Number.isFinite(value) || value <= 0) {
    return "unlimited";
  }
  const units = ["B", "KB", "MB", "GB", "TB", "PB"];
  let current = value;
  let unitIndex = 0;
  while (current >= 1024 && unitIndex < units.length - 1) {
    current /= 1024;
    unitIndex += 1;
  }
  return `${current.toFixed(current >= 10 ? 0 : 1)} ${units[unitIndex]}`;
}
