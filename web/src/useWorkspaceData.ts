import { useEffect, useState } from "react";
import {
  createBackup,
  createCertificate,
  createDNSProvider,
  createNode,
  createNotificationChannel,
  createProxyGroup,
  createRemoteServer,
  createRemoteTask,
  createRuleSet,
  createSubscription,
  createTemplate,
  createTrafficSample,
  deleteBackup,
  deleteCertificate,
  deleteDNSProvider,
  deleteNode,
  deleteNotificationChannel,
  deleteProxyGroup,
  deleteRemoteServer,
  deleteRuleSet,
  deleteSystemSetting,
  loadWorkspace,
  updateRemoteServer,
  updateRuleSet,
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
    deleteNode: (id: string) => runMutation(() => deleteNode(id)),
    createRuleSet: (input: Parameters<typeof createRuleSet>[0]) => runMutation(() => createRuleSet(input)),
    updateRuleSet: (id: string, input: Parameters<typeof updateRuleSet>[1]) =>
      runMutation(() => updateRuleSet(id, input)),
    deleteRuleSet: (id: string) => runMutation(() => deleteRuleSet(id)),
    createTemplate: (input: Parameters<typeof createTemplate>[0]) => runMutation(() => createTemplate(input)),
    createSubscription: (input: Parameters<typeof createSubscription>[0]) =>
      runMutation(() => createSubscription(input)),
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
    createBackup: (input: Parameters<typeof createBackup>[0]) => runMutation(() => createBackup(input)),
    deleteBackup: (id: string) => runMutation(() => deleteBackup(id)),
    upsertSystemSetting: (key: string, input: Parameters<typeof upsertSystemSetting>[1]) =>
      runMutation(() => upsertSystemSetting(key, input)),
    deleteSystemSetting: (key: string) => runMutation(() => deleteSystemSetting(key)),
    createTrafficSample: (input: Parameters<typeof createTrafficSample>[0]) =>
      runMutation(() => createTrafficSample(input)),
  };
}
