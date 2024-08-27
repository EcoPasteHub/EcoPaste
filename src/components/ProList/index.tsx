import { Flex, List, type ListProps } from "antd";
import type { FC } from "react";
import styles from "./index.module.scss";

const ProList: FC<ListProps<unknown>> = (props) => {
	const { header, children, ...rest } = props;

	return (
		<Flex vertical gap="small" className={styles.root}>
			{header && (
				<div data-tauri-drag-region className={styles.title}>
					<span>{header}</span>
				</div>
			)}

			<List bordered {...rest}>
				{children}
			</List>
		</Flex>
	);
};

export default ProList;
