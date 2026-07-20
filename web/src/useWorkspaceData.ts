import { useEffect, useState } from "react";
import {
  createNode,
  createRuleSet,
  createSubscription,
  createTemplate,
  deleteNode,
  loadWorkspace,
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

  async function runMutation(action: () => Promise<unknown>) {
    setState((current) => ({ ...current, busy: true, error: null }));
    try {
      await action();
      await refresh();
    } catch (error) {
      setState((current) => ({
        ...current,
        busy: false,
        error: error instanceof Error ? error.message : "Unknown error",
      }));
    }
  }

  return {
    ...state,
    refresh,
    createNode: (input: Parameters<typeof createNode>[0]) => runMutation(() => createNode(input)),
    deleteNode: (id: string) => runMutation(() => deleteNode(id)),
    createRuleSet: (input: Parameters<typeof createRuleSet>[0]) => runMutation(() => createRuleSet(input)),
    createTemplate: (input: Parameters<typeof createTemplate>[0]) => runMutation(() => createTemplate(input)),
    createSubscription: (input: Parameters<typeof createSubscription>[0]) =>
      runMutation(() => createSubscription(input)),
  };
}

