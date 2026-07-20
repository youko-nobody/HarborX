import { useState, type FormEvent } from "react";
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

export function App() {
  const { data, loading, error, busy, createNode, createRuleSet, createSubscription, createTemplate, deleteNode } =
    useWorkspaceData();
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
  const [rulePattern, setRulePattern] = useState("google.com");
  const [rulePolicy, setRulePolicy] = useState("Proxy");
  const [ruleType, setRuleType] = useState("DOMAIN-SUFFIX");

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
    await createRuleSet({
      name: ruleSetName,
      scope: "global",
      description: "Created from the HarborX operator console.",
      rules: [
        {
          ruleType,
          pattern: ruleType === "MATCH" ? "" : rulePattern,
          policy: rulePolicy,
          sortOrder: 1,
          enabled: true,
          note: "Created from the web form",
        },
      ],
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
            {data ? <span>{data.dashboard.modulesTotal} modules</span> : null}
          </div>
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
                <h3>Create the first saved rule set</h3>
              </div>
            </div>

            <form className="rule-form" onSubmit={(event) => void handleCreateRuleSet(event)}>
              <label>
                Rule set name
                <input value={ruleSetName} onChange={(event) => setRuleSetName(event.target.value)} />
              </label>

              <label>
                Rule type
                <select value={ruleType} onChange={(event) => setRuleType(event.target.value)}>
                  {ruleTypes.map((value) => (
                    <option key={value.key} value={value.key}>
                      {value.key}
                    </option>
                  ))}
                </select>
              </label>

              <label>
                Pattern
                <input
                  value={rulePattern}
                  onChange={(event) => setRulePattern(event.target.value)}
                  placeholder={ruleTypes.find((item) => item.key === ruleType)?.patternHint ?? ""}
                />
              </label>

              <label>
                Policy
                <select value={rulePolicy} onChange={(event) => setRulePolicy(event.target.value)}>
                  {policyOptions.map((value) => (
                    <option key={value} value={value}>
                      {value}
                    </option>
                  ))}
                </select>
              </label>

              <label>
                Notes
                <textarea value="This will create one persisted rule inside a rule set." readOnly />
              </label>

              <button type="submit">Save rule set</button>
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
                </div>
              ))}
            </div>
          </article>
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
