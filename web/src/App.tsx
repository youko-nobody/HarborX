import { useState, type FormEvent, type ReactNode } from "react";
import {
  listAgentLogs,
  listRemoteTasks,
  listRemoteTaskLogs,
  previewSubscription,
  previewXray,
  getAuthToken,
  login,
  setAuthToken,
  subscriptionDownloadURL,
  type AuthUser,
  type AgentLogRecord,
  type RemoteServerEnrollment,
  type RemoteTaskRecord,
  type RemoteTaskLogRecord,
  validateRuleSet,
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

type DraftRule = Omit<RuleRecord, "id"> & { id: string };

const emptyDraftRule = (): DraftRule => ({
  id: crypto.randomUUID(),
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
  const xraySnapshots = data?.xraySnapshots ?? [];
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
  const [restoredSnapshot, setRestoredSnapshot] = useState<string | null>(null);
  const [newUsername, setNewUsername] = useState("member");
  const [newUserPassword, setNewUserPassword] = useState("");
  const [newUserDisplayName, setNewUserDisplayName] = useState("Member");
  const [newUserEmail, setNewUserEmail] = useState("");
  const [userError, setUserError] = useState<string | null>(null);

  const isAuthenticated = Boolean(getAuthToken());

  async function handleLogin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setAuthError(null);
    try {
      const response = await login({ username: authUsername, password: authPassword });
      setAuthToken(response.token);
      setAuthUser(response.user);
      setAuthPassword("");
      await refresh();
    } catch (error) {
      setAuthError(error instanceof Error ? error.message : "Login failed");
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
      setNodeImportStatus(`Imported ${result.created.length} nodes, skipped ${result.skipped.length}.`);
      setNodeImportContent("");
    } catch (error) {
      setNodeImportStatus(error instanceof Error ? error.message : "Node import failed");
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
      item.rules.length
        ? item.rules.map((rule) => ({ ...rule, id: rule.id || crypto.randomUUID() }))
        : [emptyDraftRule()],
    );
    setRuleValidation(null);
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
    } catch (error) {
      setPreviewError(error instanceof Error ? error.message : "Failed to render subscription");
    }
  }

  async function handlePreviewXray() {
    setXrayError(null);
    try {
      setXrayPreview(await previewXray());
    } catch (error) {
      setXrayError(error instanceof Error ? error.message : "Failed to preview Xray config");
    }
  }

  async function handleSaveXraySnapshot() {
    setXrayError(null);
    try {
      await saveXraySnapshot({ targetKind: "local", targetId: "default" });
      await refresh();
    } catch (error) {
      setXrayError(error instanceof Error ? error.message : "Failed to save Xray snapshot");
    }
  }

  async function handleRestoreXraySnapshot(id: string) {
    setXrayError(null);
    try {
      const snapshot = await restoreXraySnapshot(id);
      setRestoredSnapshot(snapshot.config);
    } catch (error) {
      setXrayError(error instanceof Error ? error.message : "Failed to restore Xray snapshot");
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
    } catch (error) {
      setRemoteError(error instanceof Error ? error.message : "Failed to create remote server");
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
    } catch (error) {
      setRemoteError(error instanceof Error ? error.message : "Failed to load remote tasks");
    }
  }

  async function handleLoadRemoteLogs(serverId: string) {
    setRemoteError(null);
    try {
      const [agentLogs, taskLogs] = await Promise.all([
        listAgentLogs(serverId),
        remoteTasks[serverId]?.[0]?.id ? listRemoteTaskLogs(serverId, remoteTasks[serverId][0].id) : Promise.resolve([]),
      ]);
      setRemoteAgentLogs((current) => ({ ...current, [serverId]: agentLogs }));
      if (remoteTasks[serverId]?.[0]?.id) {
        setRemoteTaskLogs((current) => ({ ...current, [serverId]: taskLogs }));
      }
    } catch (error) {
      setRemoteError(error instanceof Error ? error.message : "Failed to load remote logs");
    }
  }

  async function handleLoadRemoteTaskLogs(serverId: string, taskId: string) {
    setRemoteError(null);
    try {
      const items = await listRemoteTaskLogs(serverId, taskId);
      setRemoteTaskLogs((current) => ({ ...current, [`${serverId}:${taskId}`]: items }));
    } catch (error) {
      setRemoteError(error instanceof Error ? error.message : "Failed to load remote task logs");
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
    } catch (error) {
      setRemoteError(error instanceof Error ? error.message : "Failed to create remote task");
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

  async function runOps(action: () => Promise<unknown>) {
    setOpsError(null);
    try {
      await action();
    } catch (error) {
      setOpsError(error instanceof Error ? error.message : "Operation failed");
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
    } catch (error) {
      setUserError(error instanceof Error ? error.message : "Failed to create user");
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
    } catch (error) {
      setUserError(error instanceof Error ? error.message : "Failed to update user");
    }
  }

  return (
    <div className="shell">
      <aside className="sidebar">
        <div>
          <p className="eyebrow">HarborX</p>
          <h1>Control plane for your own Xray stack</h1>
          <p className="lede">
            This rebuild keeps the feature breadth you want, but removes licensing and pro gating from the architecture.
          </p>
        </div>

        <nav className="nav">
          {[
            "Dashboard",
            "Nodes",
            "Subscriptions",
            "Rules Studio",
            "Templates",
            "Remote Servers",
            "Xray",
            "Certificates",
            "Notifications",
            "Settings",
          ].map((item) => (
            <a href="/" key={item} onClick={(event) => event.preventDefault()}>
              {item}
            </a>
          ))}
        </nav>
      </aside>

      <main className="content">
        <section className="hero">
          <div>
            <p className="eyebrow">Phase 1 scaffold</p>
            <h2>Core CRUD is now live for the first slice</h2>
            <p>
              HarborX now has real SQLite-backed CRUD for nodes, rule sets, templates, and subscriptions, while keeping the rest of the control-plane surface ready for the next implementation passes.
            </p>
          </div>
          <div className="pillbox">
            <span>No license module</span>
            <span>No pro checks</span>
            <span>Selfhost-first</span>
            <span>{isAuthenticated ? "Authenticated" : "Read-only"}</span>
            {data ? <span>{data.dashboard.modulesTotal} modules</span> : null}
          </div>
        </section>

        <section className="panel auth-panel">
          <div>
            <p className="eyebrow">Access</p>
            <h3>{isAuthenticated ? `Signed in as ${authUser?.username ?? "admin"}` : "Sign in to change data"}</h3>
            <p>
              Preview and download stay available, but create/update/delete actions require an admin session token.
            </p>
          </div>
          {isAuthenticated ? (
            <button type="button" className="ghost-button" onClick={handleLogout}>
              Sign out
            </button>
          ) : (
            <form className="auth-form" onSubmit={(event) => void handleLogin(event)}>
              <input value={authUsername} onChange={(event) => setAuthUsername(event.target.value)} placeholder="username" />
              <input
                value={authPassword}
                onChange={(event) => setAuthPassword(event.target.value)}
                placeholder="password"
                type="password"
              />
              <button type="submit">Sign in</button>
              {authError ? <p className="status error">{authError}</p> : null}
            </form>
          )}
        </section>

        {isAuthenticated ? (
          <section className="panel users-panel">
            <div className="panel-head">
              <div>
                <p className="eyebrow">Users</p>
                <h3>Members and operator access</h3>
              </div>
            </div>
            <form className="user-form" onSubmit={(event) => void handleCreateUser(event)}>
              <input value={newUsername} onChange={(event) => setNewUsername(event.target.value)} placeholder="username" />
              <input
                value={newUserPassword}
                onChange={(event) => setNewUserPassword(event.target.value)}
                placeholder="password"
                type="password"
              />
              <input
                value={newUserDisplayName}
                onChange={(event) => setNewUserDisplayName(event.target.value)}
                placeholder="display name"
              />
              <input value={newUserEmail} onChange={(event) => setNewUserEmail(event.target.value)} placeholder="email" />
              <button type="submit">Create member</button>
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
                  <div className="action-row">
                    <button type="button" className="ghost-button" onClick={() => void handleToggleUserStatus(user.id)}>
                      {user.status === "active" ? "Disable" : "Enable"}
                    </button>
                    {user.id !== "local-admin" ? (
                      <button type="button" className="ghost-button" onClick={() => void deleteUser(user.id)}>
                        Delete
                      </button>
                    ) : null}
                  </div>
                </div>
              ))}
            </div>
          </section>
        ) : null}

        <section className="stats">
          <article className="stat-card">
            <strong>{data?.dashboard.modulesTotal ?? "--"}</strong>
            <span>Feature domains</span>
          </article>
          <article className="stat-card">
            <strong>{data?.dashboard.modulesInProgress ?? "--"}</strong>
            <span>In progress now</span>
          </article>
          <article className="stat-card">
            <strong>{data?.dashboard.platformMode ?? "selfhost"}</strong>
            <span>Platform mode</span>
          </article>
          <article className="stat-card">
            <strong>{data?.dashboard.gatingModel ?? "none"}</strong>
            <span>Feature gating</span>
          </article>
        </section>

        {loading ? <p className="status">Loading bootstrap data...</p> : null}
        {error ? <p className="status error">Bootstrap load failed: {error}</p> : null}
        {busy ? <p className="status">Saving changes...</p> : null}

        <section className="grid">
          {modules.map((module) => (
            <article className="card" key={module.key}>
              <div className="card-head">
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
          ))}
        </section>

        <section className="studio">
          <div className="panel">
            <div className="panel-head">
              <div>
                <p className="eyebrow">Rules Studio</p>
                <h3>Rule sets with live persistence</h3>
              </div>
            </div>

            <div className="rule-list">
              {ruleSets.length > 0 ? (
                ruleSets.map((item) => (
                  <div className="entity-card" key={item.id}>
                    <div className="entity-head">
                      <strong>{item.name}</strong>
                      <span>{item.scope}</span>
                    </div>
                    <p>{item.description || "No description yet."}</p>
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
                        Edit
                      </button>
                      <button type="button" className="ghost-button" onClick={() => void deleteRuleSet(item.id)}>
                        Delete
                      </button>
                    </div>
                  </div>
                ))
              ) : (
                starterRules.map((rule, index) => (
                  <div className="rule-row" key={rule}>
                    <span>{String(index + 1).padStart(2, "0")}</span>
                    <code>{rule}</code>
                  </div>
                ))
              )}
            </div>
          </div>

          <div className="panel">
            <div className="panel-head">
              <div>
                <p className="eyebrow">Rule Form</p>
                <h3>{editingRuleSetId ? "Edit saved rule set" : "Create a saved rule set"}</h3>
              </div>
            </div>

            <form className="rule-form" onSubmit={(event) => void handleCreateRuleSet(event)}>
              <label>
                Rule set name
                <input value={ruleSetName} onChange={(event) => setRuleSetName(event.target.value)} />
              </label>
              <label>
                Description
                <textarea value={ruleSetDescription} onChange={(event) => setRuleSetDescription(event.target.value)} />
              </label>

              <div className="draft-rule-list">
                {draftRules.map((rule, index) => (
                  <div className="draft-rule" key={rule.id}>
                    <div className="entity-head">
                      <strong>Rule {index + 1}</strong>
                      <span>{rule.enabled ? "enabled" : "disabled"}</span>
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
                    <input value={rule.note} onChange={(event) => updateDraftRule(rule.id, { note: event.target.value })} placeholder="note" />
                    <label className="check-row">
                      <input
                        type="checkbox"
                        checked={rule.enabled}
                        onChange={(event) => updateDraftRule(rule.id, { enabled: event.target.checked })}
                      />
                      Enabled
                    </label>
                    <div className="action-row">
                      <button type="button" className="ghost-button" onClick={() => moveDraftRule(rule.id, -1)}>
                        Up
                      </button>
                      <button type="button" className="ghost-button" onClick={() => moveDraftRule(rule.id, 1)}>
                        Down
                      </button>
                      <button type="button" className="ghost-button" onClick={() => removeDraftRule(rule.id)}>
                        Remove
                      </button>
                    </div>
                  </div>
                ))}
              </div>

              {ruleValidation ? <pre className="validation-box">{ruleValidation}</pre> : null}

              <div className="action-row">
                <button type="button" onClick={() => setDraftRules((current) => [...current, { ...emptyDraftRule(), sortOrder: current.length + 1 }])}>
                  Add rule
                </button>
                <button type="submit">{editingRuleSetId ? "Update rule set" : "Save rule set"}</button>
                {editingRuleSetId ? (
                  <button type="button" className="ghost-button" onClick={resetRuleEditor}>
                    Cancel edit
                  </button>
                ) : null}
              </div>
            </form>
          </div>
        </section>

        <section className="workspace-grid">
          <article className="panel">
            <div className="panel-head">
              <div>
                <p className="eyebrow">Nodes</p>
                <h3>Add and inspect nodes</h3>
              </div>
            </div>

            <form className="stack-form" onSubmit={(event) => void handleCreateNode(event)}>
              <input placeholder="Node name" value={nodeName} onChange={(event) => setNodeName(event.target.value)} />
              <input placeholder="Server host" value={nodeHost} onChange={(event) => setNodeHost(event.target.value)} />
              <div className="inline-form">
                <select value={nodeProtocol} onChange={(event) => setNodeProtocol(event.target.value)}>
                  {["vmess", "vless", "trojan", "shadowsocks", "hysteria2", "tuic", "snell", "socks5"].map((value) => (
                    <option key={value} value={value}>
                      {value}
                    </option>
                  ))}
                </select>
                <input value={nodePort} onChange={(event) => setNodePort(event.target.value)} />
              </div>
              <input
                placeholder="tags,comma,separated"
                value={nodeTags}
                onChange={(event) => setNodeTags(event.target.value)}
              />
              <button type="submit">Create node</button>
            </form>

            <form className="stack-form import-form" onSubmit={(event) => void handleImportNodes(event)}>
              <textarea
                placeholder="Paste vmess://, vless://, trojan://, or ss:// links here, one per line."
                value={nodeImportContent}
                onChange={(event) => setNodeImportContent(event.target.value)}
              />
              <button type="submit">Import share links</button>
              {nodeImportStatus ? <p className="status">{nodeImportStatus}</p> : null}
            </form>

            <div className="entity-list">
              {nodes.map((item) => (
                <div className="entity-card" key={item.id}>
                  <div className="entity-head">
                    <strong>{item.name}</strong>
                    <span>{item.protocol}</span>
                  </div>
                  <p>
                    {item.serverHost}:{item.serverPort}
                  </p>
                  <div className="chip-row">
                    {item.tags.map((tag) => (
                      <span className="chip" key={tag}>
                        {tag}
                      </span>
                    ))}
                  </div>
                  <button type="button" className="ghost-button" onClick={() => void deleteNode(item.id)}>
                    Delete
                  </button>
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
                    {item.enabled ? "Disable" : "Enable"}
                  </button>
                </div>
              ))}
            </div>
          </article>

          <article className="panel">
            <div className="template-box">
              <p className="eyebrow">Templates</p>
              <h3>Built-in and private templates</h3>
              <form className="stack-form" onSubmit={(event) => void handleCreateTemplate(event)}>
                <input
                  placeholder="Template name"
                  value={templateName}
                  onChange={(event) => setTemplateName(event.target.value)}
                />
                <input
                  placeholder="Description"
                  value={templateDescription}
                  onChange={(event) => setTemplateDescription(event.target.value)}
                />
                <input
                  placeholder="var1,var2,var3"
                  value={templateVariables}
                  onChange={(event) => setTemplateVariables(event.target.value)}
                />
                <textarea value={templateContent} onChange={(event) => setTemplateContent(event.target.value)} />
                <button type="submit">Create template</button>
              </form>
              <div className="template-list">
                {templates.map((template) => (
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
                          Delete
                        </button>
                      ) : null}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </article>

          <article className="panel">
            <div className="panel-head">
              <div>
                <p className="eyebrow">Subscriptions</p>
                <h3>Bind formats to templates</h3>
              </div>
            </div>
            <form className="stack-form" onSubmit={(event) => void handleCreateSubscription(event)}>
              <input
                placeholder="Subscription name"
                value={subscriptionName}
                onChange={(event) => setSubscriptionName(event.target.value)}
              />
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
              <input
                placeholder="manual,imported,remote-sync"
                value={subscriptionSources}
                onChange={(event) => setSubscriptionSources(event.target.value)}
              />
              <button type="submit">Create subscription</button>
            </form>

            <div className="entity-list">
              {subscriptions.map((item) => (
                <div className="entity-card" key={item.id}>
                  <div className="entity-head">
                    <strong>{item.name}</strong>
                    <span>{item.outputFormat}</span>
                  </div>
                  <p>{item.templateId}</p>
                  <div className="chip-row">
                    {item.sources.map((source) => (
                      <span className="chip" key={source}>
                        {source}
                      </span>
                    ))}
                  </div>
                  <div className="action-row">
                    <button type="button" className="ghost-button" onClick={() => void handlePreviewSubscription(item.id)}>
                      Preview
                    </button>
                    <a className="ghost-link" href={subscriptionDownloadURL(item.id)}>
                      Download
                    </a>
                    <button type="button" className="ghost-button" onClick={() => void deleteSubscription(item.id)}>
                      Delete
                    </button>
                  </div>
                </div>
              ))}
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

          <article className="panel">
            <div className="panel-head">
              <div>
                <p className="eyebrow">Packages</p>
                <h3>Plans and user entitlements</h3>
              </div>
            </div>
            <form className="stack-form" onSubmit={(event) => void handleCreatePackage(event)}>
              <input value={packageName} onChange={(event) => setPackageName(event.target.value)} />
              <input value={packageDescription} onChange={(event) => setPackageDescription(event.target.value)} />
              <div className="inline-form">
                <input value={packageBandwidth} onChange={(event) => setPackageBandwidth(event.target.value)} placeholder="bytes" />
                <input value={packageDevices} onChange={(event) => setPackageDevices(event.target.value)} placeholder="devices" />
              </div>
              <input value={packageDuration} onChange={(event) => setPackageDuration(event.target.value)} placeholder="duration days" />
              <input value={packageFeatures} onChange={(event) => setPackageFeatures(event.target.value)} placeholder="features" />
              <button type="submit">Create package</button>
            </form>

            <form className="stack-form" onSubmit={(event) => void handleCreateEntitlement(event)}>
              <select value={entitlementUserId} onChange={(event) => setEntitlementUserId(event.target.value)}>
                {[{ id: "local-admin", username: "local-admin" }, ...users.filter((user) => user.id !== "local-admin")].map((user) => (
                  <option key={user.id} value={user.id}>
                    {user.username}
                  </option>
                ))}
              </select>
              <select value={entitlementPackageId} onChange={(event) => setEntitlementPackageId(event.target.value)}>
                <option value="">Select package</option>
                {packages.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.name}
                  </option>
                ))}
              </select>
              <input value={entitlementExpiresAt} onChange={(event) => setEntitlementExpiresAt(event.target.value)} placeholder="expires at, optional" />
              <button type="submit">Bind entitlement</button>
            </form>

            <MiniList
              items={packages.map((item) => ({
                id: item.id,
                title: item.name,
                subtitle: `${item.deviceLimit} devices / ${formatBytes(item.bandwidthBytes)} / ${item.durationDays} days`,
              }))}
              onDelete={(id) => void runOps(() => deletePackage(id))}
            />
            <MiniList
              items={entitlements.map((item) => ({
                id: item.id,
                title: `${item.userId} -> ${packages.find((pkg) => pkg.id === item.packageId)?.name ?? item.packageId}`,
                subtitle: `${item.status}${item.expiresAt ? ` until ${item.expiresAt}` : ""}`,
              }))}
              onDelete={(id) => void runOps(() => deleteEntitlement(id))}
            />
          </article>
        </section>

        <section className="remote-layout">
          <article className="panel">
            <div className="panel-head">
              <div>
                <p className="eyebrow">Remote Servers</p>
                <h3>Register VPS and queue operations</h3>
              </div>
            </div>

            <form className="stack-form" onSubmit={(event) => void handleCreateRemoteServer(event)}>
              <input
                placeholder="Server name"
                value={remoteName}
                onChange={(event) => setRemoteName(event.target.value)}
              />
              <input
                placeholder="Host or public IP"
                value={remoteHost}
                onChange={(event) => setRemoteHost(event.target.value)}
              />
              <div className="inline-form">
                <select value={remoteConnectionMode} onChange={(event) => setRemoteConnectionMode(event.target.value)}>
                  {["pull", "websocket", "http"].map((value) => (
                    <option key={value} value={value}>
                      {value}
                    </option>
                  ))}
                </select>
                <input
                  placeholder="tags"
                  value={remoteTags}
                  onChange={(event) => setRemoteTags(event.target.value)}
                />
              </div>
              <button type="submit">Register server</button>
            </form>

            {remoteEnrollment ? (
              <div className="token-box">
                <div className="entity-head">
                  <strong>Enrollment tokens for {remoteEnrollment.server.name}</strong>
                  <span>show once</span>
                </div>
                <code>server: {remoteEnrollment.serverToken}</code>
                <code>agent: {remoteEnrollment.agentToken}</code>
              </div>
            ) : null}

            {remoteError ? <p className="status error">{remoteError}</p> : null}
          </article>

          <article className="panel">
            <div className="panel-head">
              <div>
                <p className="eyebrow">Task Queue</p>
                <h3>Choose an operation payload</h3>
              </div>
            </div>
            <div className="stack-form">
              <select value={remoteTaskKind} onChange={(event) => setRemoteTaskKind(event.target.value)}>
                {[
                  "reload-config",
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
              <p>Payload is JSON and will be stored with the queued task for the agent executor.</p>
            </div>
          </article>

          <article className="panel remote-list-panel">
            <div className="entity-list">
              {remoteServers.map((server) => (
                <div className="entity-card" key={server.id}>
                  <div className="entity-head">
                    <strong>{server.name}</strong>
                    <span>{server.status}</span>
                  </div>
                  <p>
                    {server.host} via {server.connectionMode}
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
                      Mark online
                    </button>
                    <button type="button" className="ghost-button" onClick={() => void handleSetRemoteStatus(server.id, "maintenance")}>
                      Maintenance
                    </button>
                    <button type="button" className="ghost-button" onClick={() => void handleCreateRemoteTask(server.id)}>
                      Queue task
                    </button>
                    <button type="button" className="ghost-button" onClick={() => void handleLoadRemoteTasks(server.id)}>
                      Load tasks
                    </button>
                    <button type="button" className="ghost-button" onClick={() => void handleLoadRemoteLogs(server.id)}>
                      Load logs
                    </button>
                    <button type="button" className="ghost-button" onClick={() => void deleteRemoteServer(server.id)}>
                      Delete
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
                            Logs
                          </button>
                        </div>
                      ))}
                    </div>
                  ) : null}

                  {remoteAgentLogs[server.id]?.length ? (
                    <div className="log-list">
                      <strong>Agent logs</strong>
                      {remoteAgentLogs[server.id].slice(0, 6).map((logItem) => (
                        <div className="log-row" key={logItem.id}>
                          <span>{logItem.level}</span>
                          <code>{logItem.message}</code>
                          <small>{logItem.createdAt}</small>
                        </div>
                      ))}
                    </div>
                  ) : null}

                  {remoteTasks[server.id]?.flatMap((task) => remoteTaskLogs[`${server.id}:${task.id}`] ?? []).length ? (
                    <div className="log-list">
                      <strong>Task logs</strong>
                      {remoteTasks[server.id]
                        .flatMap((task) => remoteTaskLogs[`${server.id}:${task.id}`] ?? [])
                        .slice(0, 8)
                        .map((logItem) => (
                          <div className="log-row" key={logItem.id}>
                            <span>{logItem.eventKind}</span>
                            <code>{logItem.message || logItem.remoteTaskId}</code>
                            <small>{logItem.createdAt}</small>
                          </div>
                        ))}
                    </div>
                  ) : null}
                </div>
              ))}
            </div>
          </article>
        </section>

        <section className="operations-grid">
          <article className="panel">
            <p className="eyebrow">Proxy Groups</p>
            <h3>Policy groups</h3>
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
              <button type="submit">Create group</button>
            </form>
            <MiniList
              items={proxyGroups.map((item) => ({ id: item.id, title: item.name, subtitle: item.groupKind }))}
              onDelete={(id) => void runOps(() => deleteProxyGroup(id))}
            />
          </article>

          <article className="panel">
            <p className="eyebrow">DNS</p>
            <h3>Provider accounts</h3>
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
              <button type="submit">Save provider</button>
            </form>
            <MiniList
              items={dnsProviders.map((item) => ({ id: item.id, title: item.name, subtitle: item.providerKind }))}
              onDelete={(id) => void runOps(() => deleteDNSProvider(id))}
            />
          </article>

          <article className="panel">
            <p className="eyebrow">Certificates</p>
            <h3>ACME inventory</h3>
            <form className="stack-form" onSubmit={(event) => void handleCreateCertificate(event)}>
              <input value={certificateName} onChange={(event) => setCertificateName(event.target.value)} />
              <input value={certificateDomain} onChange={(event) => setCertificateDomain(event.target.value)} />
              <button type="submit">Create certificate record</button>
            </form>
            <MiniList
              items={certificates.map((item) => ({ id: item.id, title: item.name, subtitle: item.domain }))}
              onDelete={(id) => void runOps(() => deleteCertificate(id))}
            />
          </article>

            <article className="panel">
              <p className="eyebrow">Notifications</p>
              <h3>Alert channels</h3>
              <form className="stack-form" onSubmit={(event) => void handleCreateNotification(event)}>
                <input value={notificationName} onChange={(event) => setNotificationName(event.target.value)} />
                <textarea value={notificationConfig} onChange={(event) => setNotificationConfig(event.target.value)} />
                <button type="submit">Create channel</button>
              </form>
              <MiniList
                items={notificationChannels.map((item) => ({
                  id: item.id,
                  title: item.name,
                  subtitle: `${item.channelKind} ${item.enabled ? "enabled" : "disabled"}`,
                }))}
                onDelete={(id) => void runOps(() => deleteNotificationChannel(id))}
                renderActions={(item) => (
                  <button
                    type="button"
                    className="ghost-button"
                    onClick={() => void runOps(() => testNotificationChannel(item.id, { message: "HarborX test notification" }))}
                  >
                    Test
                  </button>
                )}
              />
            </article>

            <article className="panel">
              <p className="eyebrow">Backups</p>
              <h3>Backup ledger</h3>
              <form className="stack-form" onSubmit={(event) => void handleCreateBackup(event)}>
                <input
                  value={backupPath}
                  onChange={(event) => setBackupPath(event.target.value)}
                  placeholder="Export path is generated automatically"
                />
                <button type="submit">Export database</button>
              </form>
              <MiniList
                items={backups.map((item) => ({ id: item.id, title: item.backupKind, subtitle: item.filePath }))}
                onDelete={(id) => void runOps(() => deleteBackup(id))}
              />
          </article>

          <article className="panel">
            <p className="eyebrow">System</p>
            <h3>Runtime settings</h3>
            <form className="stack-form" onSubmit={(event) => void handleUpsertSetting(event)}>
              <input value={settingKey} onChange={(event) => setSettingKey(event.target.value)} />
              <textarea value={settingValue} onChange={(event) => setSettingValue(event.target.value)} />
              <button type="submit">Save setting</button>
            </form>
            <MiniList
              items={systemSettings.map((item) => ({ id: item.key, title: item.key, subtitle: JSON.stringify(item.value) }))}
              onDelete={(id) => void runOps(() => deleteSystemSetting(id))}
            />
          </article>

          <article className="panel">
            <p className="eyebrow">Traffic</p>
            <h3>Usage samples</h3>
            <form className="stack-form" onSubmit={(event) => void handleCreateTrafficSample(event)}>
              <div className="inline-form">
                <select value={trafficScope} onChange={(event) => setTrafficScope(event.target.value)}>
                  {["server", "node", "user"].map((value) => (
                    <option key={value} value={value}>
                      {value}
                    </option>
                  ))}
                </select>
                <input value={trafficScopeID} onChange={(event) => setTrafficScopeID(event.target.value)} />
              </div>
              <div className="inline-form">
                <input value={trafficRX} onChange={(event) => setTrafficRX(event.target.value)} placeholder="rx bytes" />
                <input value={trafficTX} onChange={(event) => setTrafficTX(event.target.value)} placeholder="tx bytes" />
              </div>
              <button type="submit">Record sample</button>
            </form>
            <MiniList
              items={trafficSamples.map((item) => ({
                id: item.id,
                title: `${item.sampleScope}:${item.scopeId}`,
                subtitle: `rx ${item.rxBytes} / tx ${item.txBytes}`,
              }))}
            />
          </article>

          {opsError ? <p className="status error ops-error">{opsError}</p> : null}
        </section>

        <section className="panel">
          <div className="panel-head">
            <div>
              <p className="eyebrow">Xray</p>
              <h3>Configuration preview</h3>
            </div>
            <div className="action-row">
              <button type="button" onClick={() => void handlePreviewXray()}>
                Preview Xray config
              </button>
              <button type="button" className="ghost-button" onClick={() => void handleSaveXraySnapshot()}>
                Save snapshot
              </button>
            </div>
          </div>
          {xrayError ? <p className="status error">{xrayError}</p> : null}
          <MiniList
            items={xraySnapshots.map((item) => ({
              id: item.id,
              title: item.summary || item.targetId,
              subtitle: `${item.targetKind}/${item.targetId} ${item.createdAt}`,
            }))}
            renderActions={(item) => (
              <button type="button" className="ghost-button" onClick={() => void handleRestoreXraySnapshot(item.id)}>
                Restore preview
              </button>
            )}
          />
          {xrayPreview ? (
            <div className="preview-box">
              <div className="entity-head">
                <strong>{xrayPreview.summary}</strong>
                <span>json</span>
              </div>
              <pre>{xrayPreview.content}</pre>
            </div>
          ) : null}
          {restoredSnapshot ? (
            <div className="preview-box">
              <div className="entity-head">
                <strong>Restored snapshot content</strong>
                <span>rollback</span>
              </div>
              <pre>{restoredSnapshot}</pre>
            </div>
          ) : null}
        </section>
      </main>
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
    return <p className="status mini-empty">No records yet.</p>;
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
                <button type="button" className="ghost-button" onClick={() => onDelete(item.id)}>
                  Delete
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
