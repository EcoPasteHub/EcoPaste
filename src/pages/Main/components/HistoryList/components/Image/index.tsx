import type { FC } from "react";
import LocalImage from "@/components/LocalImage";
import type { DatabaseSchemaHistory } from "@/types/database";

const Image: FC<DatabaseSchemaHistory<"image">> = (props) => {
  const { value } = props;

  return <LocalImage className="max-h-21.5" src={value} />;
};

export default Image;
