import { useEffect } from "react";
import { subscribe } from "valtio";

export const useImmediate = (...args: Parameters<typeof subscribe>): void => {
  useEffect(() => {
    const [, callback] = args;

    callback([]);

    const unsubscribe = subscribe(...args);

    return unsubscribe;
  }, []);
};
