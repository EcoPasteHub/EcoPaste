import { RouterProvider } from "react-router-dom";

const App = () => {
	useMount(() => {
		generateColorVars();
	});

	return <RouterProvider router={router} />;
};

export default App;
