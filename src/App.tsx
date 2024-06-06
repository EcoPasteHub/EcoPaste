import { RouterProvider } from "react-router-dom";

const App = () => {
	useMount(() => {
		initDatabase();

		generateColorVars();
	});

	return <RouterProvider router={router} />;
};

export default App;
