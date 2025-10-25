import { useEffect } from "react";
import { subscribe } from "valtio";

export const useSubscribe = (...args: Parameters<typeof subscribe>) => {
  useEffect(() => {
    const unsubscribe = subscribe(...args);

    return unsubscribe;
  }, []);
};
