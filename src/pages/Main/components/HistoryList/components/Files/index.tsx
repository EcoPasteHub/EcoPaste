import type { FC } from "react";
import type { DatabaseSchemaHistory } from "@/types/database";
import { isImage } from "@/utils/is";
import File from "./components/File";

const Files: FC<DatabaseSchemaHistory<"files">> = (props) => {
  const { value } = props;

  const getClassName = () => {
    if (value.length === 1) {
      if (isImage(value[0])) {
        return "max-h-21.5";
      }

      return "h-7";
    }

    if (value.length === 2) {
      return "h-14";
    }

    return "h-21.5";
  };

  return (
    <div className={getClassName()}>
      {value.map((path) => {
        return <File count={value.length} key={path} path={path} />;
      })}
    </div>
  );
};

export default Files;
