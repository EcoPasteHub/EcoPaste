import { Link } from "@heroui/react";
import { getName, getVersion } from "@tauri-apps/api/app";
import { useEffect, useState } from "react";

const REPO_URL = "https://github.com/EcoPasteHub/EcoPaste";

const AboutPanel = () => {
  const [name, setName] = useState<string>("");
  const [version, setVersion] = useState<string>("");

  useEffect(() => {
    getName().then(setName);
    getVersion().then(setVersion);
  }, []);

  return (
    <div className="flex flex-col items-center gap-2 py-6">
      <div className="text-lg">{name || "EcoPaste"}</div>
      <div className="text-default-500 text-sm">v{version || "—"}</div>
      <Link className="text-sm" href={REPO_URL} target="_blank">
        {REPO_URL}
      </Link>
    </div>
  );
};

export default AboutPanel;
