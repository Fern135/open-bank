"use client";

import { useCallback, useEffect, useState } from "react";
import { api, errorMessage } from "./api";

type State<T> = { path: string; data?: T; error?: string };

// Loads a GET path, and keeps showing the last result during a reload.
export function useData<T>(path: string) {
  const [state, setState] = useState<State<T>>({ path });
  const [version, setVersion] = useState(0);

  useEffect(() => {
    let current = true;
    api
      .get<T>(path)
      .then((response) => {
        if (current) setState({ path, data: response.data });
      })
      .catch((error) => {
        if (current) setState({ path, error: errorMessage(error) });
      });
    return () => {
      current = false;
    };
  }, [path, version]);

  const reload = useCallback(() => setVersion((value) => value + 1), []);
  const fresh = state.path === path;

  return {
    data: fresh ? state.data : undefined,
    error: fresh ? state.error : undefined,
    reload,
  };
}
