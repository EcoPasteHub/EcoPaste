import { useMount } from "ahooks";
import { type FC, useState } from "react";
import { EMFJS, RTFJS, WMFJS } from "rtf.js";
import SafeHtml from "@/components/SafeHtml";
import type { DatabaseSchemaHistory } from "@/types/database";

RTFJS.loggingEnabled(false);
WMFJS.loggingEnabled(false);
EMFJS.loggingEnabled(false);

const Rtf: FC<DatabaseSchemaHistory<"rtf">> = (props) => {
  const { value } = props;

  const [parsedHTML, setParsedHTML] = useState("");

  useMount(async () => {
    const doc = new RTFJS.Document(stringToArrayBuffer(value), {});

    const elements = await doc.render();

    pt2px(elements);

    const parsedHTML = elements.map(({ outerHTML }) => outerHTML).join("");

    setParsedHTML(parsedHTML);
  });

  const stringToArrayBuffer = (value: string) => {
    const buffer = new ArrayBuffer(value.length);

    const bufferView = new Uint8Array(buffer);

    for (let i = 0; i < value.length; i++) {
      bufferView[i] = value.charCodeAt(i);
    }

    return buffer;
  };

  const pt2px = (elements: Element[]) => {
    for (const element of elements) {
      let style = element.getAttribute("style");

      style = style?.replace(/(\d+)pt/g, "px") ?? "";

      element.setAttribute("style", style);

      pt2px([...element.children]);
    }
  };

  return <SafeHtml value={parsedHTML} />;
};

export default Rtf;
