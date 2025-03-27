import { Spin } from "antd";
import Manual from "./components/Manual";
import SavePath from "./components/SavePath";

export interface State {
	spinning: boolean;
}

const Backup = () => {
	const state = useReactive<State>({
		spinning: false,
	});

	return (
		<>
			<Spin fullscreen percent="auto" spinning={state.spinning} />

			<SavePath state={state} />

			<Manual state={state} />
		</>
	);
};

export default Backup;
