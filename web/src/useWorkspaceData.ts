import { useEffect, useState } from "react";
import {
  createBackup,
  createCertificate,
  createDNSProvider,
  createNode,
  createNotificationChannel,
  createEntitlement,
  createProxyGroup,
  createPackage,
  createRemoteServer,
  createRemoteTask,
  createRuleSet,
  createSubscription,
  createTemplate,
  createTrafficSample,
  createUser,
  deleteBackup,
  deleteCertificate,
  deleteDNSProvider,
  deleteNode,
  deleteNotificationChannel,
  deleteEntitlement,
  deleteProxyGroup,
  deletePackage,
  deleteRemoteServer,
  deleteRuleSet,
  deleteSystemSetting,
  deleteSubscription,
  deleteTemplate,
  deleteUser,
  exportBackup,
  importNodes,
  loadWorkspace,
  restoreXraySnapshot,
  saveXraySnapshot,
  testNotificationChannel,
  updateNode,
  updateRemoteServer,
  updateRuleSet,
  updateSubscription,
  updateTemplate,
  updateUser,
  upsertSystemSetting,
  type AppBootstrap,
} from "./api";

type WorkspaceState = {
  data: AppBootstrap | null;
  loading: boolean;
  error: string | null;
  busy: boolean;
};

export function useWorkspaceData() {
  const [state, setState] = useState<WorkspaceState>({
    data: null,
    loading: true,
    error: null,
    busy: false,
  });

  async function refresh() {
    setState((current) => ({ ...current, loading: true, error: null }));
    try {
      const data = await loadWorkspace();
      setState({ data, loading: false, error: null, busy: false });
    } catch (error) {
      setState({
        data: null,
        loading: false,
        error: error instanceof Error ? error.message : "Unknown error",
        busy: false,
      });
    }
  }

  useEffect(() => {
    void refresh();
  }, []);

  async function runMutation<T>(action: () => Promise<T>) {
    setState((current) => ({ ...current, busy: true, error: null }));
    try {
      const result = await action();
      await refresh();
      return result;
    } catch (error) {
      setState((current) => ({
        ...current,
        busy: false,
        error: error instanceof Error ? error.message : "Unknown error",
      }));
      throw error;
    }
  }

  return {
    ...state,
    refresh,
    createNode: (input: Parameters<typeof createNode>[0]) => runMutation(() => createNode(input)),
    importNodes: (input: Parameters<typeof importNodes>[0]) => runMutation(() => importNodes(input)),
    updateNode: (id: string, input: Parameters<typeof updateNode>[1]) => runMutation(() => updateNode(id, input)),
    deleteNode: (id: string) => runMutation(() => deleteNode(id)),
    createRuleSet: (input: Parameters<typeof createRuleSet>[0]) => runMutation(() => createRuleSet(input)),
    updateRuleSet: (id: string, input: Parameters<typeof updateRuleSet>[1]) =>
      runMutation(() => updateRuleSet(id, input)),
    deleteRuleSet: (id: string) => runMutation(() => deleteRuleSet(id)),
    createTemplate: (input: Parameters<typeof createTemplate>[0]) => runMutation(() => createTemplate(input)),
    updateTemplate: (id: string, input: Parameters<typeof updateTemplate>[1]) =>
      runMutation(() => updateTemplate(id, input)),
    deleteTemplate: (id: string) => runMutation(() => deleteTemplate(id)),
    createSubscription: (input: Parameters<typeof createSubscription>[0]) =>
      runMutation(() => createSubscription(input)),
    updateSubscription: (id: string, input: Parameters<typeof updateSubscription>[1]) =>
      runMutation(() => updateSubscription(id, input)),
    deleteSubscription: (id: string) => runMutation(() => deleteSubscription(id)),
    createPackage: (input: Parameters<typeof createPackage>[0]) => runMutation(() => createPackage(input)),
    deletePackage: (id: string) => runMutation(() => deletePackage(id)),
    createEntitlement: (input: Parameters<typeof createEntitlement>[0]) => runMutation(() => createEntitlement(input)),
    deleteEntitlement: (id: string) => runMutation(() => deleteEntitlement(id)),
    saveXraySnapshot: (input: Parameters<typeof saveXraySnapshot>[0]) => runMutation(() => saveXraySnapshot(input)),
    restoreXraySnapshot: (id: string) => runMutation(() => restoreXraySnapshot(id)),
    createRemoteServer: (input: Parameters<typeof createRemoteServer>[0]) => runMutation(() => createRemoteServer(input)),
    updateRemoteServer: (id: string, input: Parameters<typeof updateRemoteServer>[1]) =>
      runMutation(() => updateRemoteServer(id, input)),
    deleteRemoteServer: (id: string) => runMutation(() => deleteRemoteServer(id)),
    createRemoteTask: (serverId: string, input: Parameters<typeof createRemoteTask>[1]) =>
      runMutation(() => createRemoteTask(serverId, input)),
    createProxyGroup: (input: Parameters<typeof createProxyGroup>[0]) => runMutation(() => createProxyGroup(input)),
    deleteProxyGroup: (id: string) => runMutation(() => deleteProxyGroup(id)),
    createDNSProvider: (input: Parameters<typeof createDNSProvider>[0]) => runMutation(() => createDNSProvider(input)),
    deleteDNSProvider: (id: string) => runMutation(() => deleteDNSProvider(id)),
    createCertificate: (input: Parameters<typeof createCertificate>[0]) => runMutation(() => createCertificate(input)),
    deleteCertificate: (id: string) => runMutation(() => deleteCertificate(id)),
    createNotificationChannel: (input: Parameters<typeof createNotificationChannel>[0]) =>
      runMutation(() => createNotificationChannel(input)),
    deleteNotificationChannel: (id: string) => runMutation(() => deleteNotificationChannel(id)),
    testNotificationChannel: (id: string, input: Parameters<typeof testNotificationChannel>[1]) =>
      runMutation(() => testNotificationChannel(id, input)),
    createBackup: (input: Parameters<typeof createBackup>[0]) => runMutation(() => createBackup(input)),
    exportBackup: (input: Parameters<typeof exportBackup>[0]) => runMutation(() => exportBackup(input)),
    deleteBackup: (id: string) => runMutation(() => deleteBackup(id)),
    upsertSystemSetting: (key: string, input: Parameters<typeof upsertSystemSetting>[1]) =>
      runMutation(() => upsertSystemSetting(key, input)),
    deleteSystemSetting: (key: string) => runMutation(() => deleteSystemSetting(key)),
    createTrafficSample: (input: Parameters<typeof createTrafficSample>[0]) =>
      runMutation(() => createTrafficSample(input)),
    createUser: (input: Parameters<typeof createUser>[0]) => runMutation(() => createUser(input)),
    updateUser: (id: string, input: Parameters<typeof updateUser>[1]) => runMutation(() => updateUser(id, input)),
    deleteUser: (id: string) => runMutation(() => deleteUser(id)),
  };
}
