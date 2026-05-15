import { useCallback, useEffect, useRef, useState } from "react";

export function useAsync(asyncFn, options = {}) {
  const {
    immediate = true,
    initialData = null,
    args = [],
  } = options;

  const [data, setData] = useState(initialData);
  const [error, setError] = useState(null);
  const [isLoading, setIsLoading] = useState(immediate);

  const mountedRef = useRef(true);
  const fnRef = useRef(asyncFn);
  fnRef.current = asyncFn;

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const run = useCallback(async (...callArgs) => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await fnRef.current(...callArgs);
      if (mountedRef.current) setData(result);
      return result;
    } catch (err) {
      if (mountedRef.current) setError(err);
      throw err;
    } finally {
      if (mountedRef.current) setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!immediate) return;
    run(...args).catch(() => {
      // error already captured into state
    });
    // run is stable; args are intentionally not in the dep list to keep this
    // hook simple — callers control re-runs via `refetch`.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return { data, error, isLoading, refetch: run, run, setData };
}
