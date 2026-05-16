import { useEffect, useRef } from "react";
import { subscribeKey } from "valtio/utils";

export const useImmediateKey: typeof subscribeKey = (...args) => {
  const unsubscribeRef = useRef(() => {});

  useEffect(() => {
    const [object, key, callback] = args;

    callback(object[key]);

    unsubscribeRef.current = subscribeKey(...args);

    return unsubscribeRef.current;
  }, []);

  return unsubscribeRef.current;
};
