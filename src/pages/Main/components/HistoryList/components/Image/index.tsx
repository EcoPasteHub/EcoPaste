import LocalImage from "@/components/LocalImage";
import type { DatabaseSchemaHistory } from "@/types/database";
import type { FC } from "react";

const Image: FC<DatabaseSchemaHistory<"image">> = (props) => {
	const { value } = props;

	return <LocalImage src={value} className="max-h-21.5" />;
};

export default Image;
