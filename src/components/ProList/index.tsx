import { Flex, List, type ListProps } from "antd";
import type { FC } from "react";
import styles from "./index.module.scss";

const ProList: FC<ListProps<unknown>> = (props) => {
	const { header, children, ...rest } = props;

	return (
		<Flex vertical gap="small" className={styles.normal}>
			<span data-tauri-drag-region className={styles.title}>
				{header}
			</span>
			<List bordered {...rest}>
				{children}
			</List>
		</Flex>
	);
};

export default ProList;
