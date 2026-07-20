import { useState, type FormEvent } from "react";
import {
  previewSubscription,
  previewXray,
  getAuthToken,
  login,
  setAuthToken,
  subscriptionDownloadURL,
  type AuthUser,
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
    createNode,
    createRuleSet,
    updateRuleSet,
    deleteRuleSet,
    createSubscription,
    createTemplate,
    deleteNode,
  } = useWorkspaceData();
  const modules = data?.modules ?? [];
  const starterRules = data?.rules.defaultRules ?? [];
  const policyOptions = data?.rules.policies ?? [];
  const templates = data?.templates ?? [];
  const nodes = data?.nodes ?? [];
  const ruleSets = data?.ruleSets ?? [];
  const subscriptions = data?.subscriptions ?? [];
  const ruleTypes = data?.rules.ruleTypes ?? [];

  const [nodeName, setNodeName] = useState("");
  const [nodeHost, setNodeHost] = useState("");
  const [nodePort, setNodePort] = useState("443");
  const [nodeProtocol, setNodeProtocol] = useState("vless");
  const [nodeTags, setNodeTags] = useState("");

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
  const [renderedSubscription, setRenderedSubscription] = useState<RenderedSubscription | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [xrayPreview, setXrayPreview] = useState<XrayPreview | null>(null);
  const [xrayError, setXrayError] = useState<string | null>(null);
  const [authUser, setAuthUser] = useState<AuthUser | null>(null);
  const [authUsername, setAuthUsername] = useState("admin");
  const [authPassword, setAuthPassword] = useState("");
  const [authError, setAuthError] = useState<string | null>(null);

  const isAuthenticated = Boolean(getAuthToken());

  async function handleLogin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setAuthError(null);
    try {
      const response = await login({ username: authUsername, password: authPassword });
      setAuthToken(response.token);
      setAuthUser(response.user);
      setAuthPassword("");
    } catch (error) {
      setAuthError(error instanceof Error ? error.message : "Login failed");
    }
  }

  function handleLogout() {
    setAuthToken("");
    setAuthUser(null);
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
                    <span>{template.kind}</span>
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
        </section>

        <section className="panel">
          <div className="panel-head">
            <div>
              <p className="eyebrow">Xray</p>
              <h3>Configuration preview</h3>
            </div>
            <button type="button" onClick={() => void handlePreviewXray()}>
              Preview Xray config
            </button>
          </div>
          {xrayError ? <p className="status error">{xrayError}</p> : null}
          {xrayPreview ? (
            <div className="preview-box">
              <div className="entity-head">
                <strong>{xrayPreview.summary}</strong>
                <span>json</span>
              </div>
              <pre>{xrayPreview.content}</pre>
            </div>
          ) : null}
        </section>
      </main>
    </div>
  );
}

function splitCSV(input: string) {
  return input
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}
