import { useEffect, useRef } from "react";
import { subscribeKey } from "valtio/utils";

export const useSubscribeKey: typeof subscribeKey = (...args) => {
  const unsubscribeRef = useRef(() => {});

  useEffect(() => {
    unsubscribeRef.current = subscribeKey(...args);

    return unsubscribeRef.current;
  }, []);

  return unsubscribeRef.current;
};
